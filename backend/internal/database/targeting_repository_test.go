package database

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/flagstack/flagstack/backend/internal/evaluation"
	coretargeting "github.com/flagstack/flagstack/backend/internal/targeting"
)

func TestTargetingRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("FLAGSTACK_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("FLAGSTACK_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer pool.Close()

	entClient := NewEntClient(pool)
	defer entClient.Close()
	if err := Migrate(ctx, pool, entClient); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	if _, err := pool.Exec(ctx, `TRUNCATE organisations CASCADE`); err != nil {
		t.Fatalf("reset targeting tables: %v", err)
	}

	var organisationID, projectID, environmentID, featureFlagID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO organisations (name, slug)
		VALUES ('Example', 'example')
		RETURNING id::text
	`).Scan(&organisationID); err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO projects (organisation_id, name, key)
		VALUES ($1, 'Web application', 'web-app')
		RETURNING id::text
	`, organisationID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO environments (organisation_id, project_id, name, key)
		VALUES ($1, $2, 'Production', 'production')
		RETURNING id::text
	`, organisationID, projectID).Scan(&environmentID); err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO feature_flags (organisation_id, project_id, name, key, kind, default_value)
		VALUES ($1, $2, 'Checkout', 'checkout', 'boolean', 'false'::jsonb)
		RETURNING id::text
	`, organisationID, projectID).Scan(&featureFlagID); err != nil {
		t.Fatalf("create feature flag: %v", err)
	}

	repository := NewTargetingRepository(entClient)
	service, err := coretargeting.NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	segment, err := service.CreateSegment(ctx, organisationID, projectID, coretargeting.SegmentInput{
		Name:  "Beta customers",
		Key:   "beta-customers",
		Match: evaluation.MatchAll,
		Conditions: []evaluation.Condition{{
			Attribute: "groups",
			Operator:  evaluation.OperatorContains,
			Value:     json.RawMessage(`"beta-testers"`),
		}},
	})
	if err != nil {
		t.Fatalf("CreateSegment() error = %v", err)
	}
	if segment.Key != "beta-customers" {
		t.Fatalf("CreateSegment() = %#v", segment)
	}

	policy := evaluation.Policy{
		Rules: []evaluation.Rule{{
			ID:    "beta-rollout",
			Match: evaluation.MatchAll,
			Conditions: []evaluation.Condition{{
				Operator: evaluation.OperatorInSegment,
				Value:    json.RawMessage(`"beta-customers"`),
			}},
			Outcome: evaluation.Outcome{Variant: "on"},
		}},
		Fallthrough: evaluation.Outcome{Variant: "off"},
	}
	state, err := service.SetPolicy(ctx, organisationID, projectID, environmentID, featureFlagID, policy)
	if err != nil {
		t.Fatalf("SetPolicy() error = %v", err)
	}
	if state.Revision != 1 || state.Enabled {
		t.Fatalf("SetPolicy() state = %#v", state)
	}

	config, err := NewFlagConfigRepository(entClient).SetEnabled(ctx, organisationID, projectID, environmentID, featureFlagID, true)
	if err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	if !config.Enabled || config.Revision != 2 {
		t.Fatalf("SetEnabled() config = %#v", config)
	}

	betaResult, err := service.Preview(ctx, organisationID, projectID, environmentID, featureFlagID, evaluation.Context{
		TargetingKey: "user-beta",
		Attributes:   map[string]any{"groups": []any{"customer", "beta-testers"}},
	})
	if err != nil {
		t.Fatalf("Preview(beta) error = %v", err)
	}
	if string(betaResult.Value) != "true" || betaResult.Reason != evaluation.ReasonTargetingMatch || betaResult.RuleID != "beta-rollout" {
		t.Fatalf("Preview(beta) = %#v", betaResult)
	}

	otherResult, err := service.Preview(ctx, organisationID, projectID, environmentID, featureFlagID, evaluation.Context{
		TargetingKey: "user-other",
		Attributes:   map[string]any{"groups": []any{"customer"}},
	})
	if err != nil {
		t.Fatalf("Preview(other) error = %v", err)
	}
	if string(otherResult.Value) != "false" || otherResult.Variant != "off" {
		t.Fatalf("Preview(other) = %#v", otherResult)
	}

	disable := false
	disableSchedule, err := repository.CreateScheduledChange(ctx, organisationID, projectID, coretargeting.CreateScheduleInput{
		EnvironmentID: environmentID,
		FeatureFlagID: featureFlagID,
		ExecuteAt:     time.Now().Add(-time.Minute),
		Patch:         coretargeting.SchedulePatch{Enabled: &disable},
	})
	if err != nil {
		t.Fatalf("CreateScheduledChange(disable) error = %v", err)
	}

	completed, err := service.RunDueScheduledChanges(ctx, 50)
	if err != nil {
		t.Fatalf("RunDueScheduledChanges() error = %v", err)
	}
	if completed != 1 {
		t.Fatalf("RunDueScheduledChanges() completed = %d, want 1", completed)
	}

	flag, _, err := repository.LoadEvaluation(ctx, organisationID, projectID, environmentID, featureFlagID)
	if err != nil {
		t.Fatalf("LoadEvaluation() after disable error = %v", err)
	}
	if flag.Enabled {
		t.Fatal("scheduled disable left flag enabled")
	}

	changes, err := repository.ListScheduledChanges(ctx, organisationID, projectID)
	if err != nil {
		t.Fatalf("ListScheduledChanges() error = %v", err)
	}
	var executed bool
	for _, change := range changes {
		if change.ID == disableSchedule.ID && change.Status == "executed" && change.ExecutedAt != nil {
			executed = true
		}
	}
	if !executed {
		t.Fatalf("disable schedule was not marked executed: %#v", changes)
	}

	enable := true
	staleSchedule, err := repository.CreateScheduledChange(ctx, organisationID, projectID, coretargeting.CreateScheduleInput{
		EnvironmentID: environmentID,
		FeatureFlagID: featureFlagID,
		ExecuteAt:     time.Now().Add(-time.Minute),
		Patch:         coretargeting.SchedulePatch{Enabled: &enable},
	})
	if err != nil {
		t.Fatalf("CreateScheduledChange(enable) error = %v", err)
	}

	now := time.Now().UTC()
	firstClaim, err := repository.ClaimDueScheduledChanges(ctx, now, 10)
	if err != nil {
		t.Fatalf("ClaimDueScheduledChanges(first) error = %v", err)
	}
	if len(firstClaim) != 1 || firstClaim[0].ID != staleSchedule.ID {
		t.Fatalf("first claim = %#v", firstClaim)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE scheduled_flag_changes
		SET claimed_at = $1
		WHERE id = $2
	`, now.Add(-3*time.Minute), staleSchedule.ID); err != nil {
		t.Fatalf("age schedule claim: %v", err)
	}

	secondClaim, err := repository.ClaimDueScheduledChanges(ctx, now, 10)
	if err != nil {
		t.Fatalf("ClaimDueScheduledChanges(second) error = %v", err)
	}
	if len(secondClaim) != 1 || secondClaim[0].ID != staleSchedule.ID {
		t.Fatalf("second claim = %#v", secondClaim)
	}
	if secondClaim[0].ClaimToken == firstClaim[0].ClaimToken {
		t.Fatalf("stale claim token was not replaced: %q", secondClaim[0].ClaimToken)
	}

	if err := repository.ApplyClaimedScheduledChange(ctx, firstClaim[0]); !errors.Is(err, coretargeting.ErrScheduleNotPending) {
		t.Fatalf("stale worker ApplyClaimedScheduledChange() error = %v, want ErrScheduleNotPending", err)
	}
	if err := repository.ApplyClaimedScheduledChange(ctx, secondClaim[0]); err != nil {
		t.Fatalf("reclaimed ApplyClaimedScheduledChange() error = %v", err)
	}

	flag, _, err = repository.LoadEvaluation(ctx, organisationID, projectID, environmentID, featureFlagID)
	if err != nil {
		t.Fatalf("LoadEvaluation() after reclaim error = %v", err)
	}
	if !flag.Enabled {
		t.Fatal("reclaimed scheduled enable left flag disabled")
	}
}
