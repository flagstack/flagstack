package flagconfig

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrEnvironmentNotFound = errors.New("environment not found")
	ErrFeatureFlagNotFound = errors.New("feature flag not found")
)

type Config struct {
	EnvironmentID string
	FeatureFlagID string
	Enabled       bool
	Revision      int64
}

type Repository interface {
	List(context.Context, string, string) ([]Config, error)
	SetEnabled(context.Context, string, string, string, string, bool) (Config, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("flag config repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) List(ctx context.Context, organisationID, projectID string) ([]Config, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" {
		return nil, errors.New("organisation and project IDs are required")
	}
	return s.repository.List(ctx, organisationID, projectID)
}

func (s *Service) SetEnabled(ctx context.Context, organisationID, projectID, environmentID, featureFlagID string, enabled bool) (Config, error) {
	if strings.TrimSpace(organisationID) == "" || strings.TrimSpace(projectID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(featureFlagID) == "" {
		return Config{}, errors.New("organisation, project, environment and feature flag IDs are required")
	}
	return s.repository.SetEnabled(ctx, organisationID, projectID, environmentID, featureFlagID, enabled)
}
