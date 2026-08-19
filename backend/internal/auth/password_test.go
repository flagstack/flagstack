package auth

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	valid, err := VerifyPassword(hash, "correct horse battery staple")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !valid {
		t.Fatal("VerifyPassword() = false, want true")
	}

	valid, err = VerifyPassword(hash, "incorrect password")
	if err != nil {
		t.Fatalf("VerifyPassword(wrong password) error = %v", err)
	}
	if valid {
		t.Fatal("VerifyPassword(wrong password) = true, want false")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	if _, err := VerifyPassword("not-a-password-hash", "password"); err == nil {
		t.Fatal("VerifyPassword() error = nil, want error")
	}
}

func TestBootstrapValidation(t *testing.T) {
	_, err := normalizeBootstrapInput(BootstrapInput{
		Email:            "admin@example.com",
		DisplayName:      "Admin",
		Password:         "short",
		OrganisationName: "Example",
		OrganisationSlug: "example",
	})
	if err == nil {
		t.Fatal("normalizeBootstrapInput() error = nil, want validation error")
	}

	validationErr, ok := err.(*ValidationError)
	if !ok || validationErr.Field != "password" {
		t.Fatalf("error = %#v, want password ValidationError", err)
	}
}
