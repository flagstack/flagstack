package sdkconfig

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flagstack/flagstack/backend/internal/evaluation"
	"github.com/google/uuid"
)

const (
	SchemaVersion = 1
	KindServer    = "server"
	KindClient    = "client"

	serverKeyPrefix = "fs_server_"
	clientKeyPrefix = "fs_client_"
)

var (
	ErrCredentialNotFound  = errors.New("SDK credential not found")
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrFeatureFlagNotFound = errors.New("feature flag not found")
	ErrInvalidCredential   = errors.New("invalid SDK credential")
)

type Credential struct {
	ID             string     `json:"id"`
	OrganisationID string     `json:"organisation_id"`
	ProjectID      string     `json:"project_id"`
	EnvironmentID  string     `json:"environment_id"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	ClientKey      string     `json:"client_key,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type CreatedCredential struct {
	Credential Credential `json:"credential"`
	Key        string     `json:"key"`
}

type CreateInput struct {
	Name          string
	Kind          string
	EnvironmentID string
}

type CredentialRecord struct {
	ID             uuid.UUID
	OrganisationID string
	ProjectID      string
	EnvironmentID  string
	Name           string
	Kind           string
	ClientKey      string
	SecretDigest   []byte
}

type StoredServerCredential struct {
	Credential
	SecretDigest []byte
}

type Environment struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type Flag struct {
	ID           string               `json:"id"`
	Key          string               `json:"key"`
	Kind         string               `json:"kind"`
	DefaultValue json.RawMessage      `json:"default_value"`
	Enabled      bool                 `json:"enabled"`
	Variants     []evaluation.Variant `json:"variants,omitempty"`
	Policy       evaluation.Policy    `json:"policy,omitempty"`
	Revision     int64                `json:"revision"`
}

type Configuration struct {
	SchemaVersion int                  `json:"schema_version"`
	Environment   Environment          `json:"environment"`
	Flags         []Flag               `json:"flags"`
	Segments      []evaluation.Segment `json:"segments,omitempty"`
}

type Repository interface {
	ListCredentials(context.Context, string, string) ([]Credential, error)
	CreateCredential(context.Context, CredentialRecord) (Credential, error)
	RevokeCredential(context.Context, string, string, string, time.Time) (Credential, error)
	FindServerCredential(context.Context, string) (StoredServerCredential, error)
	FindClientCredential(context.Context, string) (Credential, error)
	SetClientVisible(context.Context, string, string, string, bool) (bool, error)
	LoadConfiguration(context.Context, Credential) (Configuration, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("SDK configuration repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) ListCredentials(ctx context.Context, organisationID, projectID string) ([]Credential, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" {
		return nil, errors.New("organisation and project IDs are required")
	}
	return s.repository.ListCredentials(ctx, organisationID, projectID)
}

func (s *Service) CreateCredential(ctx context.Context, organisationID, projectID string, input CreateInput) (CreatedCredential, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreatedCredential{}, &ValidationError{Field: "name", Message: "is required"}
	}
	if len(name) > 160 {
		return CreatedCredential{}, &ValidationError{Field: "name", Message: "is too long"}
	}
	kind := strings.ToLower(strings.TrimSpace(input.Kind))
	if kind != KindServer && kind != KindClient {
		return CreatedCredential{}, &ValidationError{Field: "kind", Message: "must be server or client"}
	}
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return CreatedCredential{}, &ValidationError{Field: "environment_id", Message: "is required"}
	}

	id, err := uuid.NewV7()
	if err != nil {
		return CreatedCredential{}, fmt.Errorf("generate SDK credential ID: %w", err)
	}
	record := CredentialRecord{
		ID:             id,
		OrganisationID: organisationID,
		ProjectID:      projectID,
		EnvironmentID:  input.EnvironmentID,
		Name:           name,
		Kind:           kind,
	}

	var key string
	if kind == KindServer {
		secret, err := randomToken(32)
		if err != nil {
			return CreatedCredential{}, err
		}
		digest := sha256.Sum256([]byte(secret))
		record.SecretDigest = digest[:]
		key = serverKeyPrefix + strings.ReplaceAll(id.String(), "-", "") + "." + secret
	} else {
		public, err := randomToken(24)
		if err != nil {
			return CreatedCredential{}, err
		}
		record.ClientKey = clientKeyPrefix + public
		key = record.ClientKey
	}

	credential, err := s.repository.CreateCredential(ctx, record)
	if err != nil {
		return CreatedCredential{}, err
	}
	return CreatedCredential{Credential: credential, Key: key}, nil
}

func (s *Service) RevokeCredential(ctx context.Context, organisationID, projectID, credentialID string) (Credential, error) {
	if strings.TrimSpace(credentialID) == "" {
		return Credential{}, ErrCredentialNotFound
	}
	return s.repository.RevokeCredential(ctx, organisationID, projectID, credentialID, time.Now().UTC())
}

func (s *Service) SetClientVisible(ctx context.Context, organisationID, projectID, featureFlagID string, visible bool) (bool, error) {
	return s.repository.SetClientVisible(ctx, organisationID, projectID, featureFlagID, visible)
}

func (s *Service) ConfigurationForKey(ctx context.Context, rawKey string) (Configuration, error) {
	credential, err := s.authenticate(ctx, strings.TrimSpace(rawKey))
	if err != nil {
		return Configuration{}, err
	}
	configuration, err := s.repository.LoadConfiguration(ctx, credential)
	if err != nil {
		return Configuration{}, err
	}
	configuration.SchemaVersion = SchemaVersion
	configuration.Segments = referencedSegments(configuration.Flags, configuration.Segments)
	return configuration, nil
}

func (s *Service) authenticate(ctx context.Context, rawKey string) (Credential, error) {
	if strings.HasPrefix(rawKey, serverKeyPrefix) {
		parts := strings.SplitN(strings.TrimPrefix(rawKey, serverKeyPrefix), ".", 2)
		if len(parts) != 2 || len(parts[0]) != 32 || parts[1] == "" {
			return Credential{}, ErrInvalidCredential
		}
		id, err := uuid.Parse(parts[0])
		if err != nil {
			return Credential{}, ErrInvalidCredential
		}
		stored, err := s.repository.FindServerCredential(ctx, id.String())
		if err != nil {
			return Credential{}, ErrInvalidCredential
		}
		digest := sha256.Sum256([]byte(parts[1]))
		if len(stored.SecretDigest) != len(digest) || subtle.ConstantTimeCompare(stored.SecretDigest, digest[:]) != 1 {
			return Credential{}, ErrInvalidCredential
		}
		return stored.Credential, nil
	}

	if strings.HasPrefix(rawKey, clientKeyPrefix) {
		credential, err := s.repository.FindClientCredential(ctx, rawKey)
		if err != nil {
			return Credential{}, ErrInvalidCredential
		}
		return credential, nil
	}
	return Credential{}, ErrInvalidCredential
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func randomToken(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate SDK credential secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func referencedSegments(flags []Flag, segments []evaluation.Segment) []evaluation.Segment {
	index := make(map[string]evaluation.Segment, len(segments))
	for _, segment := range segments {
		index[segment.Key] = segment
	}

	needed := make(map[string]bool)
	queue := make([]string, 0)
	addConditions := func(conditions []evaluation.Condition) {
		for _, condition := range conditions {
			if condition.Operator != evaluation.OperatorInSegment && condition.Operator != evaluation.OperatorNotInSegment {
				continue
			}
			var key string
			if err := json.Unmarshal(condition.Value, &key); err == nil && key != "" && !needed[key] {
				needed[key] = true
				queue = append(queue, key)
			}
		}
	}

	for _, flag := range flags {
		for _, rule := range flag.Policy.Rules {
			addConditions(rule.Conditions)
		}
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		segment, ok := index[key]
		if !ok {
			continue
		}
		addConditions(segment.Conditions)
	}

	result := make([]evaluation.Segment, 0, len(needed))
	for _, segment := range segments {
		if needed[segment.Key] {
			result = append(result, segment)
		}
	}
	return result
}
