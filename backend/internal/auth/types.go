package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var (
	ErrBootstrapComplete  = errors.New("initial bootstrap is already complete")
	ErrCredentialNotFound = errors.New("local credential not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
)

var organisationSlugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

type User struct {
	ID          string
	Email       string
	DisplayName string
}

type OrganisationMembership struct {
	ID   string
	Name string
	Slug string
	Role string
}

type Principal struct {
	User          User
	Organisations []OrganisationMembership
}

type Session struct {
	Principal Principal
	CSRFHash  [32]byte
	ExpiresAt time.Time
}

type IssuedSession struct {
	SessionToken string
	CSRFToken    string
	ExpiresAt    time.Time
	Principal    Principal
}

type BootstrapInput struct {
	Email            string
	DisplayName      string
	Password         string
	OrganisationName string
	OrganisationSlug string
}

type LoginInput struct {
	Email    string
	Password string
}

type BootstrapRecord struct {
	Email            string
	DisplayName      string
	PasswordHash     string
	OrganisationName string
	OrganisationSlug string
	SessionTokenHash [32]byte
	CSRFHash         [32]byte
	SessionExpiresAt time.Time
}

type Credential struct {
	UserID       string
	PasswordHash string
}

type SessionRecord struct {
	UserID    string
	TokenHash [32]byte
	CSRFHash  [32]byte
	ExpiresAt time.Time
}

type Repository interface {
	BootstrapRequired(context.Context) (bool, error)
	Bootstrap(context.Context, BootstrapRecord) (Principal, error)
	CredentialByEmail(context.Context, string) (Credential, error)
	CreateSession(context.Context, SessionRecord) (Principal, error)
	SessionByTokenHash(context.Context, [32]byte) (Session, error)
	DeleteSession(context.Context, [32]byte) error
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", &ValidationError{Field: "email", Message: "is required"}
	}
	if len(email) > 320 {
		return "", &ValidationError{Field: "email", Message: "is too long"}
	}

	address, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(address.Address, email) {
		return "", &ValidationError{Field: "email", Message: "must be a valid email address"}
	}

	return email, nil
}

func validatePassword(password string) error {
	if len(password) < 12 {
		return &ValidationError{Field: "password", Message: "must be at least 12 characters"}
	}
	if len(password) > 1024 {
		return &ValidationError{Field: "password", Message: "is too long"}
	}
	return nil
}

func normalizeBootstrapInput(input BootstrapInput) (BootstrapInput, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return BootstrapInput{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return BootstrapInput{}, err
	}

	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return BootstrapInput{}, &ValidationError{Field: "display_name", Message: "is required"}
	}
	if len(displayName) > 120 {
		return BootstrapInput{}, &ValidationError{Field: "display_name", Message: "is too long"}
	}

	organisationName := strings.TrimSpace(input.OrganisationName)
	if organisationName == "" {
		return BootstrapInput{}, &ValidationError{Field: "organisation_name", Message: "is required"}
	}
	if len(organisationName) > 160 {
		return BootstrapInput{}, &ValidationError{Field: "organisation_name", Message: "is too long"}
	}

	organisationSlug := strings.ToLower(strings.TrimSpace(input.OrganisationSlug))
	if !organisationSlugPattern.MatchString(organisationSlug) {
		return BootstrapInput{}, &ValidationError{Field: "organisation_slug", Message: "must contain only lowercase letters, numbers and single hyphen-separated segments"}
	}

	return BootstrapInput{
		Email:            email,
		DisplayName:      displayName,
		Password:         input.Password,
		OrganisationName: organisationName,
		OrganisationSlug: organisationSlug,
	}, nil
}
