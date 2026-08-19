package featureflag

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeRepository struct {
	created CreateInput
}

func (f *fakeRepository) List(context.Context, string, string) ([]FeatureFlag, error) {
	return nil, nil
}

func (f *fakeRepository) Create(_ context.Context, _, _ string, input CreateInput) (FeatureFlag, error) {
	f.created = input
	return FeatureFlag{
		Name:         input.Name,
		Key:          input.Key,
		Description:  input.Description,
		Kind:         input.Kind,
		DefaultValue: input.DefaultValue,
	}, nil
}

func TestCreateValidatesDefaultValueKind(t *testing.T) {
	tests := []struct {
		name         string
		kind         string
		defaultValue string
		wantError    bool
	}{
		{name: "boolean", kind: "boolean", defaultValue: `false`},
		{name: "string", kind: "string", defaultValue: `"enabled"`},
		{name: "number", kind: "number", defaultValue: `42.5`},
		{name: "json object", kind: "json", defaultValue: `{"percentage":50}`},
		{name: "json null", kind: "json", defaultValue: `null`},
		{name: "boolean mismatch", kind: "boolean", defaultValue: `"false"`, wantError: true},
		{name: "string mismatch", kind: "string", defaultValue: `1`, wantError: true},
		{name: "number mismatch", kind: "number", defaultValue: `true`, wantError: true},
		{name: "invalid json", kind: "json", defaultValue: `{`, wantError: true},
		{name: "invalid kind", kind: "variant", defaultValue: `false`, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service, err := NewService(repository)
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}

			_, err = service.Create(context.Background(), "org-1", "project-1", CreateInput{
				Name:         "Checkout flow",
				Key:          "checkout.new-flow",
				Kind:         tt.kind,
				DefaultValue: json.RawMessage(tt.defaultValue),
			})
			if tt.wantError {
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Fatalf("Create() error = %v, want ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if repository.created.Key != "checkout.new-flow" || repository.created.Kind != tt.kind {
				t.Fatalf("normalized input = %#v", repository.created)
			}
		})
	}
}

func TestCreateRejectsInvalidKey(t *testing.T) {
	service, err := NewService(&fakeRepository{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Create(context.Background(), "org-1", "project-1", CreateInput{
		Name:         "Checkout flow",
		Key:          "checkout/new-flow",
		Kind:         "boolean",
		DefaultValue: json.RawMessage(`false`),
	})
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "key" {
		t.Fatalf("Create() error = %v, want key ValidationError", err)
	}
}
