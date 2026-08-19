-- +goose Up
CREATE TABLE local_credentials (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT local_credentials_password_hash_not_blank CHECK (btrim(password_hash) <> '')
);

CREATE TABLE user_sessions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    csrf_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_sessions_token_hash_length CHECK (octet_length(token_hash) = 32),
    CONSTRAINT user_sessions_csrf_hash_length CHECK (octet_length(csrf_hash) = 32),
    CONSTRAINT user_sessions_expiry_after_creation CHECK (expires_at > created_at)
);

CREATE INDEX user_sessions_user_expiry_idx
    ON user_sessions (user_id, expires_at DESC);

CREATE INDEX user_sessions_expiry_idx
    ON user_sessions (expires_at);

-- +goose Down
DROP TABLE user_sessions;
DROP TABLE local_credentials;
