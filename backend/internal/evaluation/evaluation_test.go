package evaluation

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBucketHasStableCrossSDKVector(t *testing.T) {
	if got := Bucket("env-1", "flag-1", "user-123"); got != 22683 {
		t.Fatalf("Bucket() = %d, want 22683", got)
	}
}

func TestEvaluateBooleanDisabledAndEnabledDefaults(t *testing.T) {
	flag := booleanFlag()

	disabled := Evaluate(flag, Context{}, nil)
	if string(disabled.Value) != "false" || disabled.Reason != ReasonDisabled || disabled.Variant != "default" {
		t.Fatalf("disabled result = %#v", disabled)
	}

	flag.Enabled = true
	enabled := Evaluate(flag, Context{}, nil)
	if string(enabled.Value) != "true" || enabled.Reason != ReasonStatic || enabled.Variant != "on" {
		t.Fatalf("enabled result = %#v", enabled)
	}
}

func TestEvaluateTargetingRulesAndGroups(t *testing.T) {
	flag := booleanFlag()
	flag.Enabled = true
	flag.Policy.Rules = []Rule{
		{
			ID:    "staff",
			Match: MatchAll,
			Conditions: []Condition{{
				Attribute: "email",
				Operator:  OperatorEndsWith,
				Value:     raw(`"@example.com"`),
			}},
			Outcome: Outcome{Variant: "on"},
		},
		{
			ID:    "beta",
			Match: MatchAll,
			Conditions: []Condition{{
				Attribute: "groups",
				Operator:  OperatorContains,
				Value:     raw(`"beta-testers"`),
			}},
			Outcome: Outcome{Variant: "on"},
		},
	}
	flag.Policy.Fallthrough = Outcome{Variant: "off"}

	staff := Evaluate(flag, Context{Attributes: map[string]any{"email": "adam@example.com"}}, nil)
	if string(staff.Value) != "true" || staff.RuleID != "staff" || staff.Reason != ReasonTargetingMatch {
		t.Fatalf("staff result = %#v", staff)
	}

	beta := Evaluate(flag, Context{Attributes: map[string]any{"email": "elsewhere.test", "groups": []any{"customer", "beta-testers"}}}, nil)
	if string(beta.Value) != "true" || beta.RuleID != "beta" {
		t.Fatalf("beta result = %#v", beta)
	}

	other := Evaluate(flag, Context{Attributes: map[string]any{"email": "elsewhere.test", "groups": []any{"customer"}}}, nil)
	if string(other.Value) != "false" || other.Reason != ReasonStatic {
		t.Fatalf("other result = %#v", other)
	}
}

func TestEvaluateReusableSegments(t *testing.T) {
	flag := booleanFlag()
	flag.Enabled = true
	flag.Policy.Rules = []Rule{{
		ID:    "premium",
		Match: MatchAll,
		Conditions: []Condition{{
			Operator: OperatorInSegment,
			Value:    raw(`"premium-customers"`),
		}},
		Outcome: Outcome{Variant: "on"},
	}}
	flag.Policy.Fallthrough = Outcome{Variant: "off"}

	segments := []Segment{{
		Key:   "premium-customers",
		Name:  "Premium customers",
		Match: MatchAll,
		Conditions: []Condition{
			{Attribute: "plan", Operator: OperatorIn, Value: raw(`["pro","enterprise"]`)},
			{Attribute: "account.active", Operator: OperatorEquals, Value: raw(`true`)},
		},
	}}

	result := Evaluate(flag, Context{Attributes: map[string]any{
		"plan":    "enterprise",
		"account": map[string]any{"active": true},
	}}, segments)
	if string(result.Value) != "true" || result.RuleID != "premium" {
		t.Fatalf("segment result = %#v", result)
	}
}

func TestEvaluatePercentageRolloutIsDeterministicAndProgressive(t *testing.T) {
	tenPercent := booleanFlag()
	tenPercent.Enabled = true
	tenPercent.Policy.Fallthrough = percentageOutcome(10_000)

	twentyFivePercent := tenPercent
	twentyFivePercent.Policy.Fallthrough = percentageOutcome(25_000)

	onAtTen := 0
	onAtTwentyFive := 0
	for i := 0; i < 1000; i++ {
		ctx := Context{TargetingKey: fmt.Sprintf("user-%d", i)}
		first := Evaluate(tenPercent, ctx, nil)
		second := Evaluate(tenPercent, ctx, nil)
		if string(first.Value) != string(second.Value) || first.Variant != second.Variant {
			t.Fatalf("rollout was not deterministic for %q", ctx.TargetingKey)
		}

		expanded := Evaluate(twentyFivePercent, ctx, nil)
		if first.Variant == "on" && expanded.Variant != "on" {
			t.Fatalf("progressive rollout removed %q from the enabled cohort", ctx.TargetingKey)
		}
		if first.Variant == "on" {
			onAtTen++
		}
		if expanded.Variant == "on" {
			onAtTwentyFive++
		}
	}

	if onAtTen < 70 || onAtTen > 130 {
		t.Fatalf("10%% rollout selected %d/1000 subjects", onAtTen)
	}
	if onAtTwentyFive < 210 || onAtTwentyFive > 290 {
		t.Fatalf("25%% rollout selected %d/1000 subjects", onAtTwentyFive)
	}
}

func TestEvaluateVariantsSupportABAssignmentWithoutAnalytics(t *testing.T) {
	flag := Flag{
		ID:            "flag-variant",
		EnvironmentID: "env-1",
		Key:           "checkout-layout",
		Kind:          "string",
		DefaultValue:  raw(`"control"`),
		Enabled:       true,
		Variants: []Variant{
			{Key: "control", Value: raw(`"control"`)},
			{Key: "compact", Value: raw(`"compact"`)},
			{Key: "new-design", Value: raw(`"new-design"`)},
		},
		Policy: Policy{Fallthrough: Outcome{Rollout: []Allocation{
			{Variant: "control", Weight: 50_000},
			{Variant: "compact", Weight: 25_000},
			{Variant: "new-design", Weight: 25_000},
		}}},
	}

	ctx := Context{TargetingKey: "user-123"}
	first := Evaluate(flag, ctx, nil)
	second := Evaluate(flag, ctx, nil)
	if first.Reason != ReasonSplit || first.Variant == "" || string(first.Value) == "" {
		t.Fatalf("variant result = %#v", first)
	}
	if first.Variant != second.Variant || string(first.Value) != string(second.Value) {
		t.Fatalf("variant assignment changed: %#v then %#v", first, second)
	}
}

func TestEvaluateCanBucketByArbitraryContextAttribute(t *testing.T) {
	flag := booleanFlag()
	flag.Enabled = true
	flag.Policy.Fallthrough = Outcome{
		BucketBy: "organisation_id",
		Rollout: []Allocation{
			{Variant: "on", Weight: 50_000},
			{Variant: "off", Weight: 50_000},
		},
	}

	first := Evaluate(flag, Context{TargetingKey: "user-a", Attributes: map[string]any{"organisation_id": "org-42"}}, nil)
	second := Evaluate(flag, Context{TargetingKey: "user-b", Attributes: map[string]any{"organisation_id": "org-42"}}, nil)
	if first.Variant != second.Variant {
		t.Fatalf("shared bucket attribute produced different variants: %#v %#v", first, second)
	}
}

func TestEvaluateRolloutWithoutTargetingKeyFailsSafe(t *testing.T) {
	flag := booleanFlag()
	flag.Enabled = true
	flag.Policy.Fallthrough = percentageOutcome(10_000)

	result := Evaluate(flag, Context{}, nil)
	if result.Reason != ReasonError || result.ErrorCode != ErrorTargetingKeyMissing || string(result.Value) != "false" {
		t.Fatalf("missing targeting key result = %#v", result)
	}
}

func TestEvaluateConditionOperators(t *testing.T) {
	ctx := Context{Attributes: map[string]any{
		"age":         30,
		"email":       "adam@example.com",
		"app_version": "2.4.0",
		"country":     "GB",
		"roles":       []any{"admin", "developer"},
	}}

	tests := []struct {
		name      string
		condition Condition
	}{
		{"numeric", Condition{Attribute: "age", Operator: OperatorGreaterThanOrEqual, Value: raw(`21`)}},
		{"regex", Condition{Attribute: "email", Operator: OperatorMatchesRegex, Value: raw(`"^[^@]+@example\\.com$"`)}},
		{"semver", Condition{Attribute: "app_version", Operator: OperatorSemverGreaterThan, Value: raw(`"2.3.1"`)}},
		{"in", Condition{Attribute: "country", Operator: OperatorIn, Value: raw(`["GB","IE"]`)}},
		{"contains", Condition{Attribute: "roles", Operator: OperatorContains, Value: raw(`"admin"`)}},
		{"exists", Condition{Attribute: "email", Operator: OperatorExists}},
		{"not exists", Condition{Attribute: "missing", Operator: OperatorNotExists}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, err := conditionMatches(test.condition, ctx, nil, map[string]bool{})
			if err != nil {
				t.Fatalf("conditionMatches() error = %v", err)
			}
			if !matched {
				t.Fatalf("condition %#v did not match", test.condition)
			}
		})
	}
}

func TestEvaluateDetectsSegmentCycles(t *testing.T) {
	flag := booleanFlag()
	flag.Enabled = true
	flag.Policy.Rules = []Rule{{
		ID:         "segment-rule",
		Match:      MatchAll,
		Conditions: []Condition{{Operator: OperatorInSegment, Value: raw(`"a"`)}},
		Outcome:    Outcome{Variant: "on"},
	}}

	segments := []Segment{
		{Key: "a", Name: "A", Match: MatchAll, Conditions: []Condition{{Operator: OperatorInSegment, Value: raw(`"b"`)}}},
		{Key: "b", Name: "B", Match: MatchAll, Conditions: []Condition{{Operator: OperatorInSegment, Value: raw(`"a"`)}}},
	}
	result := Evaluate(flag, Context{}, segments)
	if result.Reason != ReasonError || result.ErrorCode != ErrorParse {
		t.Fatalf("cycle result = %#v", result)
	}
}

func TestValidateFlagRejectsInvalidRollout(t *testing.T) {
	flag := booleanFlag()
	flag.Enabled = true
	flag.Policy.Fallthrough = Outcome{Rollout: []Allocation{
		{Variant: "on", Weight: 10_000},
		{Variant: "off", Weight: 10_000},
	}}
	if err := ValidateFlag(flag); err == nil {
		t.Fatal("ValidateFlag() error = nil, want invalid rollout total")
	}
}

func booleanFlag() Flag {
	return Flag{
		ID:            "flag-1",
		EnvironmentID: "env-1",
		Key:           "new-checkout",
		Kind:          "boolean",
		DefaultValue:  raw(`false`),
	}
}

func percentageOutcome(onWeight int) Outcome {
	return Outcome{Rollout: []Allocation{
		{Variant: "on", Weight: onWeight},
		{Variant: "off", Weight: BucketScale - onWeight},
	}}
}

func raw(value string) json.RawMessage {
	return json.RawMessage(value)
}
