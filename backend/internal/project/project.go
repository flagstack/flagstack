package project

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrKeyConflict = errors.New("project key already exists")
)

var keyPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9_-]{0,62}[a-z0-9])?$`)

type Project struct {
	ID               string
	OrganisationID   string
	Name             string
	Key              string
	Description      string
	EnvironmentCount int
	FeatureFlagCount int
	CreatedAt        time.Time
}

type CreateInput struct {
	Name        string
	Key         string
	Description string
}

type Repository interface {
	List(context.Context, string) ([]Project, error)
	Create(context.Context, string, CreateInput) (Project, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("project repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context, organisationID string) ([]Project, error) {
	if strings.TrimSpace(organisationID) == "" {
		return nil, errors.New("organisation ID is required")
	}
	return s.repository.List(ctx, organisationID)
}

func (s *Service) Create(ctx context.Context, organisationID string, input CreateInput) (Project, error) {
	if strings.TrimSpace(organisationID) == "" {
		return Project{}, errors.New("organisation ID is required")
	}

	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return Project{}, err
	}
	return s.repository.Create(ctx, organisationID, normalized)
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
