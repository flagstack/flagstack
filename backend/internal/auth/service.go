package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
)

const sessionSecretLength = 32

type Service struct {
	repository Repository
	sessionTTL time.Duration
	dummyHash  string
}

func NewService(repository Repository, sessionTTL time.Duration) (*Service, error) {
	if repository == nil {
		return nil, errors.New("auth repository is required")
	}
	if sessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}

	return &Service{
		repository: repository,
		sessionTTL: sessionTTL,
		dummyHash:  dummyPasswordHash(),
	}, nil
}

func (s *Service) BootstrapRequired(ctx context.Context) (bool, error) {
	return s.repository.BootstrapRequired(ctx)
}

func (s *Service) Bootstrap(ctx context.Context, input BootstrapInput) (IssuedSession, error) {
	normalized, err := normalizeBootstrapInput(input)
	if err != nil {
		return IssuedSession{}, err
	}

	passwordHash, err := HashPassword(normalized.Password)
	if err != nil {
		return IssuedSession{}, err
	}

	issued, tokenHash, csrfHash, expiresAt, err := s.newSessionSecrets()
	if err != nil {
		return IssuedSession{}, err
	}

	principal, err := s.repository.Bootstrap(ctx, BootstrapRecord{
		Email:            normalized.Email,
		DisplayName:      normalized.DisplayName,
		PasswordHash:     passwordHash,
		OrganisationName: normalized.OrganisationName,
		OrganisationSlug: normalized.OrganisationSlug,
		SessionTokenHash: tokenHash,
		CSRFHash:         csrfHash,
		SessionExpiresAt: expiresAt,
	})
	if err != nil {
		return IssuedSession{}, err
	}

	issued.Principal = principal
	return issued, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (IssuedSession, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return IssuedSession{}, ErrInvalidCredentials
	}
	if strings.TrimSpace(input.Password) == "" || len(input.Password) > 1024 {
		return IssuedSession{}, ErrInvalidCredentials
	}

	credential, err := s.repository.CredentialByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, ErrCredentialNotFound) {
			return IssuedSession{}, err
		}
		_, _ = VerifyPassword(s.dummyHash, input.Password)
		return IssuedSession{}, ErrInvalidCredentials
	}

	valid, err := VerifyPassword(credential.PasswordHash, input.Password)
	if err != nil {
		return IssuedSession{}, fmt.Errorf("verify stored password hash: %w", err)
	}
	if !valid {
		return IssuedSession{}, ErrInvalidCredentials
	}

	issued, tokenHash, csrfHash, expiresAt, err := s.newSessionSecrets()
	if err != nil {
		return IssuedSession{}, err
	}

	principal, err := s.repository.CreateSession(ctx, SessionRecord{
		UserID:    credential.UserID,
		TokenHash: tokenHash,
		CSRFHash:  csrfHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return IssuedSession{}, err
	}

	issued.Principal = principal
	return issued, nil
}

func (s *Service) AuthenticateSession(ctx context.Context, token string) (Session, error) {
	if !validSecret(token) {
		return Session{}, ErrSessionNotFound
	}
	return s.repository.SessionByTokenHash(ctx, sha256.Sum256([]byte(token)))
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if !validSecret(token) {
		return nil
	}
	return s.repository.DeleteSession(ctx, sha256.Sum256([]byte(token)))
}

func ValidCSRF(session Session, token string) bool {
	if !validSecret(token) {
		return false
	}
	hash := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(hash[:], session.CSRFHash[:]) == 1
}

func (s *Service) newSessionSecrets() (IssuedSession, [32]byte, [32]byte, time.Time, error) {
	sessionToken, err := randomSecret()
	if err != nil {
		return IssuedSession{}, [32]byte{}, [32]byte{}, time.Time{}, err
	}
	csrfToken, err := randomSecret()
	if err != nil {
		return IssuedSession{}, [32]byte{}, [32]byte{}, time.Time{}, err
	}

	expiresAt := time.Now().UTC().Add(s.sessionTTL)
	return IssuedSession{
		SessionToken: sessionToken,
		CSRFToken:    csrfToken,
		ExpiresAt:    expiresAt,
	}, sha256.Sum256([]byte(sessionToken)), sha256.Sum256([]byte(csrfToken)), expiresAt, nil
}

func randomSecret() (string, error) {
	secret := make([]byte, sessionSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate session secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(secret), nil
}

func validSecret(secret string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(secret)
	return err == nil && len(decoded) == sessionSecretLength
}
