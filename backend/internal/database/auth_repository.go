package database

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	flagstackent "github.com/flagstack/flagstack/backend/ent"
	"github.com/flagstack/flagstack/backend/ent/localcredential"
	"github.com/flagstack/flagstack/backend/ent/organisationmembership"
	"github.com/flagstack/flagstack/backend/ent/user"
	"github.com/flagstack/flagstack/backend/ent/usersession"
	"github.com/flagstack/flagstack/backend/internal/auth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const bootstrapAdvisoryLock int64 = 0x466c616753746163

type AuthRepository struct {
	pool   *pgxpool.Pool
	client *flagstackent.Client
}

func NewAuthRepository(pool *pgxpool.Pool, client *flagstackent.Client) *AuthRepository {
	return &AuthRepository{pool: pool, client: client}
}

func (r *AuthRepository) BootstrapRequired(ctx context.Context) (bool, error) {
	exists, err := r.client.User.Query().Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("check bootstrap status: %w", err)
	}
	return !exists, nil
}

func (r *AuthRepository) Bootstrap(ctx context.Context, record auth.BootstrapRecord) (auth.Principal, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("acquire bootstrap lock connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, bootstrapAdvisoryLock); err != nil {
		return auth.Principal{}, fmt.Errorf("lock bootstrap: %w", err)
	}
	defer func() { _, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, bootstrapAdvisoryLock) }()

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	exists, err := tx.User.Query().Exist(ctx)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("recheck bootstrap status: %w", err)
	}
	if exists {
		return auth.Principal{}, auth.ErrBootstrapComplete
	}

	userEntity, err := tx.User.Create().
		SetEmail(record.Email).
		SetDisplayName(record.DisplayName).
		Save(ctx)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap user: %w", err)
	}

	if _, err := tx.LocalCredential.Create().
		SetUserID(userEntity.ID).
		SetPasswordHash(record.PasswordHash).
		Save(ctx); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap credential: %w", err)
	}

	organisationEntity, err := tx.Organisation.Create().
		SetName(record.OrganisationName).
		SetSlug(record.OrganisationSlug).
		Save(ctx)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap organisation: %w", err)
	}

	if _, err := tx.OrganisationMembership.Create().
		SetOrganisationID(organisationEntity.ID).
		SetUserID(userEntity.ID).
		SetRole("owner").
		Save(ctx); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap membership: %w", err)
	}

	if _, err := tx.UserSession.Create().
		SetUserID(userEntity.ID).
		SetTokenHash(record.SessionTokenHash[:]).
		SetCsrfHash(record.CSRFHash[:]).
		SetExpiresAt(record.SessionExpiresAt).
		Save(ctx); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return auth.Principal{}, fmt.Errorf("commit bootstrap transaction: %w", err)
	}

	return auth.Principal{
		User: auth.User{
			ID:          userEntity.ID.String(),
			Email:       record.Email,
			DisplayName: record.DisplayName,
		},
		Organisations: []auth.OrganisationMembership{{
			ID:   organisationEntity.ID.String(),
			Name: organisationEntity.Name,
			Slug: organisationEntity.Slug,
			Role: "owner",
		}},
	}, nil
}

func (r *AuthRepository) CredentialByEmail(ctx context.Context, email string) (auth.Credential, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	userEntity, err := r.client.User.Query().Where(user.Email(normalized)).Only(ctx)
	if flagstackent.IsNotFound(err) {
		return auth.Credential{}, auth.ErrCredentialNotFound
	}
	if err != nil {
		return auth.Credential{}, fmt.Errorf("find local credential user: %w", err)
	}

	credentialEntity, err := r.client.LocalCredential.Query().
		Where(localcredential.UserID(userEntity.ID)).
		Only(ctx)
	if flagstackent.IsNotFound(err) {
		return auth.Credential{}, auth.ErrCredentialNotFound
	}
	if err != nil {
		return auth.Credential{}, fmt.Errorf("find local credential: %w", err)
	}

	return auth.Credential{
		UserID:       userEntity.ID.String(),
		PasswordHash: credentialEntity.PasswordHash,
	}, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, record auth.SessionRecord) (auth.Principal, error) {
	userID, err := uuid.Parse(record.UserID)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("parse session user ID: %w", err)
	}

	if _, err := r.client.UserSession.Create().
		SetUserID(userID).
		SetTokenHash(record.TokenHash[:]).
		SetCsrfHash(record.CSRFHash[:]).
		SetExpiresAt(record.ExpiresAt).
		Save(ctx); err != nil {
		return auth.Principal{}, fmt.Errorf("create session: %w", err)
	}
	return r.principalByUserID(ctx, userID)
}

func (r *AuthRepository) SessionByTokenHash(ctx context.Context, tokenHash [32]byte) (auth.Session, error) {
	sessionEntity, err := r.client.UserSession.Query().
		Where(
			usersession.TokenHash(tokenHash[:]),
			usersession.ExpiresAtGT(time.Now()),
		).
		Only(ctx)
	if flagstackent.IsNotFound(err) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("find session: %w", err)
	}
	if len(sessionEntity.CsrfHash) != 32 {
		return auth.Session{}, errors.New("stored CSRF hash has invalid length")
	}

	principal, err := r.principalByUserID(ctx, sessionEntity.UserID)
	if err != nil {
		return auth.Session{}, err
	}

	var csrfHash [32]byte
	copy(csrfHash[:], sessionEntity.CsrfHash)
	return auth.Session{
		Principal: principal,
		CSRFHash:  csrfHash,
		ExpiresAt: sessionEntity.ExpiresAt,
	}, nil
}

func (r *AuthRepository) DeleteSession(ctx context.Context, tokenHash [32]byte) error {
	if _, err := r.client.UserSession.Delete().Where(usersession.TokenHash(tokenHash[:])).Exec(ctx); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *AuthRepository) principalByUserID(ctx context.Context, userID uuid.UUID) (auth.Principal, error) {
	userEntity, err := r.client.User.Get(ctx, userID)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("load session user: %w", err)
	}

	email := ""
	if userEntity.Email != nil {
		email = *userEntity.Email
	}
	memberships, err := r.membershipsByUserID(ctx, userID)
	if err != nil {
		return auth.Principal{}, err
	}

	return auth.Principal{
		User: auth.User{
			ID:          userEntity.ID.String(),
			Email:       email,
			DisplayName: userEntity.DisplayName,
		},
		Organisations: memberships,
	}, nil
}

func (r *AuthRepository) membershipsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.OrganisationMembership, error) {
	entities, err := r.client.OrganisationMembership.Query().
		Where(organisationmembership.UserID(userID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("load organisation memberships: %w", err)
	}

	memberships := make([]auth.OrganisationMembership, 0, len(entities))
	for _, entity := range entities {
		organisationEntity, err := r.client.Organisation.Get(ctx, entity.OrganisationID)
		if err != nil {
			return nil, fmt.Errorf("load membership organisation: %w", err)
		}
		memberships = append(memberships, auth.OrganisationMembership{
			ID:   organisationEntity.ID.String(),
			Name: organisationEntity.Name,
			Slug: organisationEntity.Slug,
			Role: entity.Role,
		})
	}

	sort.Slice(memberships, func(i, j int) bool {
		if memberships[i].Name == memberships[j].Name {
			return memberships[i].ID < memberships[j].ID
		}
		return memberships[i].Name < memberships[j].Name
	})
	return memberships, nil
}
