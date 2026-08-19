package flagconfig

import (
	"context"
	"testing"
)

type fakeRepository struct {
	configs []Config
}

func (f *fakeRepository) List(context.Context, string, string) ([]Config, error) {
	return append([]Config(nil), f.configs...), nil
}

func (f *fakeRepository) SetEnabled(_ context.Context, _, _, environmentID, featureFlagID string, enabled bool) (Config, error) {
	config := Config{EnvironmentID: environmentID, FeatureFlagID: featureFlagID, Enabled: enabled, Revision: 1}
	f.configs = append(f.configs, config)
	return config, nil
}

func TestServiceDelegatesFlagConfiguration(t *testing.T) {
	repository := &fakeRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	configured, err := service.SetEnabled(context.Background(), "org-1", "project-1", "environment-1", "flag-1", true)
	if err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}
	if !configured.Enabled || configured.EnvironmentID != "environment-1" || configured.FeatureFlagID != "flag-1" {
		t.Fatalf("SetEnabled() config = %#v", configured)
	}

	configs, err := service.List(context.Background(), "org-1", "project-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("List() length = %d, want 1", len(configs))
	}
}

func TestServiceRejectsMissingScope(t *testing.T) {
	service, err := NewService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err := service.List(context.Background(), "", "project-1"); err == nil {
		t.Fatal("List() error = nil, want error")
	}
	if _, err := service.SetEnabled(context.Background(), "org-1", "project-1", "", "flag-1", true); err == nil {
		t.Fatal("SetEnabled() error = nil, want error")
	}
}
