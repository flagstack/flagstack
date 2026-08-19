package featureflag

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var (
	ErrKeyConflict     = errors.New("feature flag key already exists")
	ErrProjectNotFound = errors.New("project not found")
)

var keyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._-]{0,126}[a-z0-9])?$`)

type FeatureFlag struct {
	ID             string
	OrganisationID string
	ProjectID      string
	Name           string
	Key            string
	Description    string
	Kind           string
	DefaultValue   json.RawMessage
	ClientVisible  bool
	CreatedAt      time.Time
}

type CreateInput struct {
	Name         string
	Key          string
	Description  string
	Kind         string
	DefaultValue json.RawMessage
}

type Repository interface {
	List(context.Context, string, string) ([]FeatureFlag, error)
	Create(context.Context, string, string, CreateInput) (FeatureFlag, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("feature flag repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context, organisationID, projectID string) ([]FeatureFlag, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" {
		return nil, errors.New("organisation and project IDs are required")
	}
	return s.repository.List(ctx, organisationID, projectID)
}

func (s *Service) Create(ctx context.Context, organisationID, projectID string, input CreateInput) (FeatureFlag, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" {
		return FeatureFlag{}, errors.New("organisation and project IDs are required")
	}

	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return FeatureFlag{}, err
	}
	return s.repository.Create(ctx, organisationID, projectID, normalized)
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return CreateInput{}, &ValidationError{Field: "name", Message: "is required"}
	}
	if len(name) > 160 {
		return CreateInput{}, &ValidationError{Field: "name", Message: "is too long"}
	}

	key := strings.ToLower(strings.TrimSpace(input.Key))
	if !keyPattern.MatchString(key) {
		return CreateInput{}, &ValidationError{Field: "key", Message: "must use lowercase letters, numbers, dots, hyphens or underscores and be at most 128 characters"}
	}

	description := strings.TrimSpace(input.Description)
	if len(description) > 2000 {
		return CreateInput{}, &ValidationError{Field: "description", Message: "is too long"}
	}

	kind := strings.ToLower(strings.TrimSpace(input.Kind))
	if kind != "boolean" && kind != "string" && kind != "number" && kind != "json" {
		return CreateInput{}, &ValidationError{Field: "kind", Message: "must be boolean, string, number or json"}
	}

	defaultValue, value, err := normalizeJSON(input.DefaultValue)
	if err != nil {
		return CreateInput{}, &ValidationError{Field: "default_value", Message: "must be valid JSON"}
	}
	if !matchesKind(kind, value) {
		return CreateInput{}, &ValidationError{Field: "default_value", Message: "must match the selected flag kind"}
	}

	return CreateInput{
		Name:         name,
		Key:          key,
		Description:  description,
		Kind:         kind,
		DefaultValue: defaultValue,
	}, nil
}

func normalizeJSON(raw json.RawMessage) (json.RawMessage, any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil, errors.New("empty JSON value")
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("multiple JSON values")
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return nil, nil, err
	}
	return json.RawMessage(compact.Bytes()), value, nil
}

func matchesKind(kind string, value any) bool {
	switch kind {
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(json.Number)
		return ok
	case "json":
		return true
	default:
		return false
	}
}
