package sdkconfig

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flagstack/flagstack/backend/internal/evaluation"
	"github.com/google/uuid"
)

type fakeRepository struct {
	records       map[string]CredentialRecord
	credentials   map[string]Credential
	clientKeys    map[string]string
	configuration Configuration
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		records:     make(map[string]CredentialRecord),
		credentials: make(map[string]Credential),
		clientKeys:  make(map[string]string),
	}
}

func (f *fakeRepository) ListCredentials(context.Context, string, string) ([]Credential, error) {
	result := make([]Credential, 0, len(f.credentials))
	for _, credential := range f.credentials {
		result = append(result, credential)
	}
	return result, nil
}

func (f *fakeRepository) CreateCredential(_ context.Context, record CredentialRecord) (Credential, error) {
	id := record.ID.String()
	credential := Credential{
		ID:             id,
		OrganisationID: record.OrganisationID,
		ProjectID:      record.ProjectID,
		EnvironmentID:  record.EnvironmentID,
		Name:           record.Name,
		Kind:           record.Kind,
		ClientKey:      record.ClientKey,
		CreatedAt:      time.Now().UTC(),
	}
	f.records[id] = record
	f.credentials[id] = credential
	if record.ClientKey != "" {
		f.clientKeys[record.ClientKey] = id
	}
	return credential, nil
}

func (f *fakeRepository) RevokeCredential(_ context.Context, _, _, credentialID string, revokedAt time.Time) (Credential, error) {
	credential, ok := f.credentials[credentialID]
	if !ok {
		return Credential{}, ErrCredentialNotFound
	}
	credential.RevokedAt = &revokedAt
	f.credentials[credentialID] = credential
	return credential, nil
}

func (f *fakeRepository) FindServerCredential(_ context.Context, credentialID string) (StoredServerCredential, error) {
	credential, ok := f.credentials[credentialID]
	if !ok || credential.Kind != KindServer || credential.RevokedAt != nil {
		return StoredServerCredential{}, ErrCredentialNotFound
	}
	record := f.records[credentialID]
	return StoredServerCredential{Credential: credential, SecretDigest: append([]byte(nil), record.SecretDigest...)}, nil
}

func (f *fakeRepository) FindClientCredential(_ context.Context, clientKey string) (Credential, error) {
	credentialID, ok := f.clientKeys[clientKey]
	if !ok {
		return Credential{}, ErrCredentialNotFound
	}
	credential := f.credentials[credentialID]
	if credential.RevokedAt != nil {
		return Credential{}, ErrCredentialNotFound
	}
	return credential, nil
}

func (f *fakeRepository) SetClientVisible(context.Context, string, string, string, bool) (bool, error) {
	return false, nil
}

func (f *fakeRepository) LoadConfiguration(context.Context, Credential) (Configuration, error) {
	return f.configuration, nil
}

func TestServerCredentialStoresOnlyDigestAndAuthenticates(t *testing.T) {
	repository := newFakeRepository()
	repository.configuration = Configuration{Environment: Environment{ID: "env-1", Key: "production"}}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	created, err := service.CreateCredential(context.Background(), "org-1", "project-1", CreateInput{
		Name: "Production backend", Kind: KindServer, EnvironmentID: "env-1",
	})
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}
	if !strings.HasPrefix(created.Key, serverKeyPrefix) {
		t.Fatalf("server key = %q, want %q prefix", created.Key, serverKeyPrefix)
	}
	record := repository.records[created.Credential.ID]
	if len(record.SecretDigest) != 32 {
		t.Fatalf("secret digest length = %d, want 32", len(record.SecretDigest))
	}
	if strings.Contains(string(record.SecretDigest), created.Key) {
		t.Fatal("stored digest unexpectedly contains full server key")
	}
	if record.ClientKey != "" {
		t.Fatalf("server client key = %q, want empty", record.ClientKey)
	}

	configuration, err := service.ConfigurationForKey(context.Background(), created.Key)
	if err != nil {
		t.Fatalf("ConfigurationForKey() error = %v", err)
	}
	if configuration.SchemaVersion != SchemaVersion {
		t.Fatalf("schema version = %d, want %d", configuration.SchemaVersion, SchemaVersion)
	}

	parts := strings.SplitN(created.Key, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("server key = %q, want id.secret format", created.Key)
	}
	if _, err := service.ConfigurationForKey(context.Background(), parts[0]+".wrong-secret"); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("wrong secret error = %v, want ErrInvalidCredential", err)
	}
}

func TestClientCredentialAuthenticatesAndRevocationInvalidatesKey(t *testing.T) {
	repository := newFakeRepository()
	repository.configuration = Configuration{Environment: Environment{ID: "env-1", Key: "production"}}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	created, err := service.CreateCredential(context.Background(), "org-1", "project-1", CreateInput{
		Name: "Browser", Kind: KindClient, EnvironmentID: "env-1",
	})
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}
	if !strings.HasPrefix(created.Key, clientKeyPrefix) || created.Credential.ClientKey != created.Key {
		t.Fatalf("client credential = %#v, key = %q", created.Credential, created.Key)
	}
	if _, err := service.ConfigurationForKey(context.Background(), created.Key); err != nil {
		t.Fatalf("ConfigurationForKey() error = %v", err)
	}
	if _, err := service.RevokeCredential(context.Background(), "org-1", "project-1", created.Credential.ID); err != nil {
		t.Fatalf("RevokeCredential() error = %v", err)
	}
	if _, err := service.ConfigurationForKey(context.Background(), created.Key); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("revoked key error = %v, want ErrInvalidCredential", err)
	}
}

func TestConfigurationPrunesUnreferencedSegmentsTransitively(t *testing.T) {
	segmentRef := func(key string) json.RawMessage {
		value, err := json.Marshal(key)
		if err != nil {
			t.Fatalf("json.Marshal(%q) error = %v", key, err)
		}
		return value
	}

	repository := newFakeRepository()
	repository.configuration = Configuration{
		Environment: Environment{ID: "env-1", Key: "production"},
		Flags: []Flag{{
			ID: "flag-1", Key: "new-checkout", Kind: "boolean", DefaultValue: json.RawMessage("false"), Enabled: true,
			Policy: evaluation.Policy{Rules: []evaluation.Rule{{
				ID: "staff-rule", Match: evaluation.MatchAll,
				Conditions: []evaluation.Condition{{Operator: evaluation.OperatorInSegment, Value: segmentRef("staff")}},
				Outcome: evaluation.Outcome{Variant: "on"},
			}}},
		}},
		Segments: []evaluation.Segment{
			{Key: "staff", Name: "Staff", Match: evaluation.MatchAll, Conditions: []evaluation.Condition{{Operator: evaluation.OperatorInSegment, Value: segmentRef("internal")}}},
			{Key: "internal", Name: "Internal", Match: evaluation.MatchAll, Conditions: []evaluation.Condition{{Attribute: "email", Operator: evaluation.OperatorEndsWith, Value: json.RawMessage(`"@example.com"`)}}},
			{Key: "unused", Name: "Unused", Match: evaluation.MatchAll, Conditions: []evaluation.Condition{{Attribute: "plan", Operator: evaluation.OperatorEquals, Value: json.RawMessage(`"enterprise"`)}}},
		},
	}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	created, err := service.CreateCredential(context.Background(), "org-1", "project-1", CreateInput{
		Name: "Browser", Kind: KindClient, EnvironmentID: "env-1",
	})
	if err != nil {
		t.Fatalf("CreateCredential() error = %v", err)
	}

	configuration, err := service.ConfigurationForKey(context.Background(), created.Key)
	if err != nil {
		t.Fatalf("ConfigurationForKey() error = %v", err)
	}
	if len(configuration.Segments) != 2 || configuration.Segments[0].Key != "staff" || configuration.Segments[1].Key != "internal" {
		t.Fatalf("segments = %#v, want staff and internal only", configuration.Segments)
	}
}

func TestCreateCredentialValidatesInput(t *testing.T) {
	service, err := NewService(newFakeRepository())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	for name, input := range map[string]CreateInput{
		"name":        {Kind: KindServer, EnvironmentID: uuid.NewString()},
		"kind":        {Name: "Key", Kind: "mobile", EnvironmentID: uuid.NewString()},
		"environment": {Name: "Key", Kind: KindServer},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.CreateCredential(context.Background(), "org-1", "project-1", input); err == nil {
				t.Fatal("CreateCredential() error = nil, want validation error")
			}
		})
	}
}
