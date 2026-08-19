package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/flagstack/flagstack/backend/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const bootstrapAdvisoryLock int64 = 0x466c616753746163

type AuthRepository struct {
	pool *pgxpool.Pool
}

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) BootstrapRequired(ctx context.Context) (bool, error) {
	var required bool
	if err := r.pool.QueryRow(ctx, `SELECT NOT EXISTS (SELECT 1 FROM users LIMIT 1)`).Scan(&required); err != nil {
		return false, fmt.Errorf("check bootstrap status: %w", err)
	}
	return required, nil
}

func (r *AuthRepository) Bootstrap(ctx context.Context, record auth.BootstrapRecord) (auth.Principal, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return auth.Principal{}, fmt.Errorf("begin bootstrap transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, bootstrapAdvisoryLock); err != nil {
		return auth.Principal{}, fmt.Errorf("lock bootstrap transaction: %w", err)
	}

	var userExists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users LIMIT 1)`).Scan(&userExists); err != nil {
		return auth.Principal{}, fmt.Errorf("recheck bootstrap status: %w", err)
	}
	if userExists {
		return auth.Principal{}, auth.ErrBootstrapComplete
	}

	var userID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO users (email, display_name) VALUES ($1, $2) RETURNING id::text`,
		record.Email,
		record.DisplayName,
	).Scan(&userID); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap user: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO local_credentials (user_id, password_hash) VALUES ($1, $2)`,
		userID,
		record.PasswordHash,
	); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap credential: %w", err)
	}

	var organisationID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO organisations (name, slug) VALUES ($1, $2) RETURNING id::text`,
		record.OrganisationName,
		record.OrganisationSlug,
	).Scan(&organisationID); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap organisation: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO organisation_memberships (organisation_id, user_id, role) VALUES ($1, $2, 'owner')`,
		organisationID,
		userID,
	); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap membership: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO user_sessions (user_id, token_hash, csrf_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		userID,
		record.SessionTokenHash[:],
		record.CSRFHash[:],
		record.SessionExpiresAt,
	); err != nil {
		return auth.Principal{}, fmt.Errorf("create bootstrap session: %w", err)
	}

	principal := auth.Principal{
		User: auth.User{ID: userID, Email: record.Email, DisplayName: record.DisplayName},
		Organisations: []auth.OrganisationMembership{{
			ID:   organisationID,
			Name: record.OrganisationName,
			Slug: record.OrganisationSlug,
			Role: "owner",
		}},
	}

	if err := tx.Commit(ctx); err != nil {
		return auth.Principal{}, fmt.Errorf("commit bootstrap transaction: %w", err)
	}
	return principal, nil
}

func (r *AuthRepository) CredentialByEmail(ctx context.Context, email string) (auth.Credential, error) {
	var credential auth.Credential
	err := r.pool.QueryRow(ctx, `
		SELECT u.id::text, c.password_hash
		FROM users u
		JOIN local_credentials c ON c.user_id = u.id
		WHERE lower(u.email) = lower($1)
	`, email).Scan(&credential.UserID, &credential.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Credential{}, auth.ErrCredentialNotFound
	}
	if err != nil {
		return auth.Credential{}, fmt.Errorf("find local credential: %w", err)
	}
	return credential, nil
}

func (r *AuthRepository) CreateSession(ctx context.Context, record auth.SessionRecord) (auth.Principal, error) {
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO user_sessions (user_id, token_hash, csrf_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		record.UserID,
		record.TokenHash[:],
		record.CSRFHash[:],
		record.ExpiresAt,
	); err != nil {
		return auth.Principal{}, fmt.Errorf("create session: %w", err)
	}
	return r.principalByUserID(ctx, record.UserID)
}

func (r *AuthRepository) SessionByTokenHash(ctx context.Context, tokenHash [32]byte) (auth.Session, error) {
	var session auth.Session
	var csrfHash []byte
	var userID string

	err := r.pool.QueryRow(ctx, `
		SELECT s.user_id::text, s.csrf_hash, s.expires_at, u.email, u.display_name
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()
	`, tokenHash[:]).Scan(
		&userID,
		&csrfHash,
		&session.ExpiresAt,
		&session.Principal.User.Email,
		&session.Principal.User.DisplayName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("find session: %w", err)
	}
	if len(csrfHash) != len(session.CSRFHash) {
		return auth.Session{}, errors.New("stored CSRF hash has invalid length")
	}
	copy(session.CSRFHash[:], csrfHash)
	session.Principal.User.ID = userID

	memberships, err := r.membershipsByUserID(ctx, userID)
	if err != nil {
		return auth.Session{}, err
	}
	session.Principal.Organisations = memberships
	return session, nil
}

func (r *AuthRepository) DeleteSession(ctx context.Context, tokenHash [32]byte) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, tokenHash[:]); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *AuthRepository) principalByUserID(ctx context.Context, userID string) (auth.Principal, error) {
	var principal auth.Principal
	principal.User.ID = userID
	if err := r.pool.QueryRow(ctx,
		`SELECT email, display_name FROM users WHERE id = $1`,
		userID,
	).Scan(&principal.User.Email, &principal.User.DisplayName); err != nil {
		return auth.Principal{}, fmt.Errorf("load session user: %w", err)
	}

	memberships, err := r.membershipsByUserID(ctx, userID)
	if err != nil {
		return auth.Principal{}, err
	}
	principal.Organisations = memberships
	return principal, nil
}

func (r *AuthRepository) membershipsByUserID(ctx context.Context, userID string) ([]auth.OrganisationMembership, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id::text, o.name, o.slug, m.role
		FROM organisation_memberships m
		JOIN organisations o ON o.id = m.organisation_id
		WHERE m.user_id = $1
		ORDER BY o.name, o.id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("load organisation memberships: %w", err)
	}
	defer rows.Close()

	memberships := make([]auth.OrganisationMembership, 0)
	for rows.Next() {
		var membership auth.OrganisationMembership
		if err := rows.Scan(&membership.ID, &membership.Name, &membership.Slug, &membership.Role); err != nil {
			return nil, fmt.Errorf("scan organisation membership: %w", err)
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organisation memberships: %w", err)
	}
	return memberships, nil
}
