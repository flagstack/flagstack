package environment

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrKeyConflict     = errors.New("environment key already exists")
	ErrProjectNotFound = errors.New("project not found")
)

var keyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,62}[a-z0-9])?$`)

type Environment struct {
	ID             string
	OrganisationID string
	ProjectID      string
	Name           string
	Key            string
	Description    string
	CreatedAt      time.Time
}

type CreateInput struct {
	Name        string
	Key         string
	Description string
}

type Repository interface {
	List(context.Context, string, string) ([]Environment, error)
	Create(context.Context, string, string, CreateInput) (Environment, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("environment repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context, organisationID, projectID string) ([]Environment, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" {
		return nil, errors.New("organisation and project IDs are required")
	}
	return s.repository.List(ctx, organisationID, projectID)
}

func (s *Service) Create(ctx context.Context, organisationID, projectID string, input CreateInput) (Environment, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" {
		return Environment{}, errors.New("organisation and project IDs are required")
	}
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return Environment{}, err
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
		return CreateInput{}, &ValidationError{Field: "key", Message: "must use lowercase letters, numbers, hyphens or underscores and be at most 64 characters"}
	}

	description := strings.TrimSpace(input.Description)
	if len(description) > 2000 {
		return CreateInput{}, &ValidationError{Field: "description", Message: "is too long"}
	}

	return CreateInput{Name: name, Key: key, Description: description}, nil
}
