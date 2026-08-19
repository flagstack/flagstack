package project

import (
	"context"
	"testing"
)

type fakeRepository struct {
	createdOrganisationID string
	createdInput          CreateInput
}

func (f *fakeRepository) List(context.Context, string) ([]Project, error) {
	return []Project{}, nil
}

func (f *fakeRepository) Create(_ context.Context, organisationID string, input CreateInput) (Project, error) {
	f.createdOrganisationID = organisationID
	f.createdInput = input
	return Project{ID: "project-1", OrganisationID: organisationID, Name: input.Name, Key: input.Key, Description: input.Description}, nil
}

func TestCreateNormalisesProjectInput(t *testing.T) {
	repository := &fakeRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	project, err := service.Create(context.Background(), "org-1", CreateInput{
		Name:        "  API Service  ",
		Key:         "  API_Service  ",
		Description: "  Backend feature flags  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if project.Key != "api_service" {
		t.Fatalf("project key = %q, want %q", project.Key, "api_service")
	}
	if repository.createdOrganisationID != "org-1" {
		t.Fatalf("organisation ID = %q", repository.createdOrganisationID)
	}
	if repository.createdInput.Name != "API Service" || repository.createdInput.Description != "Backend feature flags" {
		t.Fatalf("normalized input = %#v", repository.createdInput)
	}
}

func TestCreateRejectsInvalidKey(t *testing.T) {
	service, err := NewService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Create(context.Background(), "org-1", CreateInput{Name: "API", Key: "API service"})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	validationErr, ok := err.(*ValidationError)
	if !ok || validationErr.Field != "key" {
		t.Fatalf("error = %#v, want key ValidationError", err)
	}
}
