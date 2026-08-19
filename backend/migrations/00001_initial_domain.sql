-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    email text,
    display_name text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_not_blank CHECK (email IS NULL OR btrim(email) <> '')
);

CREATE UNIQUE INDEX users_email_lower_unique
    ON users (lower(email))
    WHERE email IS NOT NULL;

CREATE TABLE organisations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organisations_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT organisations_slug_format CHECK (
        slug ~ '^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$'
    )
);

CREATE TABLE organisation_memberships (
    organisation_id uuid NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, user_id),
    CONSTRAINT organisation_memberships_role CHECK (
        role IN ('owner', 'admin', 'developer', 'viewer')
    )
);

CREATE INDEX organisation_memberships_user_idx
    ON organisation_memberships (user_id, organisation_id);

CREATE TABLE projects (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    organisation_id uuid NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    name text NOT NULL,
    key text NOT NULL,
    description text NOT NULL DEFAULT '',
    archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT projects_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT projects_key_not_blank CHECK (btrim(key) <> ''),
    CONSTRAINT projects_key_length CHECK (char_length(key) <= 64),
    UNIQUE (organisation_id, id),
    UNIQUE (organisation_id, key)
);

CREATE INDEX projects_organisation_active_idx
    ON projects (organisation_id, created_at DESC)
    WHERE archived_at IS NULL;

CREATE TABLE environments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    organisation_id uuid NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    key text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT environments_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT environments_key_not_blank CHECK (btrim(key) <> ''),
    CONSTRAINT environments_key_length CHECK (char_length(key) <= 64),
    UNIQUE (organisation_id, project_id, id),
    UNIQUE (project_id, key),
    FOREIGN KEY (organisation_id, project_id)
        REFERENCES projects (organisation_id, id)
        ON DELETE CASCADE
);

CREATE TABLE feature_flags (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    organisation_id uuid NOT NULL,
    project_id uuid NOT NULL,
    name text NOT NULL,
    key text NOT NULL,
    description text NOT NULL DEFAULT '',
    kind text NOT NULL,
    default_value jsonb NOT NULL,
    archived_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT feature_flags_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT feature_flags_key_not_blank CHECK (btrim(key) <> ''),
    CONSTRAINT feature_flags_key_length CHECK (char_length(key) <= 128),
    CONSTRAINT feature_flags_kind CHECK (kind IN ('boolean', 'string', 'number', 'json')),
    CONSTRAINT feature_flags_default_value_type CHECK (
        kind = 'json'
        OR (kind = 'boolean' AND jsonb_typeof(default_value) = 'boolean')
        OR (kind = 'string' AND jsonb_typeof(default_value) = 'string')
        OR (kind = 'number' AND jsonb_typeof(default_value) = 'number')
    ),
    UNIQUE (organisation_id, project_id, id),
    UNIQUE (project_id, key),
    FOREIGN KEY (organisation_id, project_id)
        REFERENCES projects (organisation_id, id)
        ON DELETE CASCADE
);

CREATE INDEX feature_flags_project_active_idx
    ON feature_flags (project_id, created_at DESC)
    WHERE archived_at IS NULL;

CREATE TABLE environment_flag_configs (
    organisation_id uuid NOT NULL,
    project_id uuid NOT NULL,
    environment_id uuid NOT NULL,
    feature_flag_id uuid NOT NULL,
    enabled boolean NOT NULL DEFAULT false,
    value jsonb,
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (environment_id, feature_flag_id),
    CONSTRAINT environment_flag_configs_revision_positive CHECK (revision > 0),
    FOREIGN KEY (organisation_id, project_id, environment_id)
        REFERENCES environments (organisation_id, project_id, id)
        ON DELETE CASCADE,
    FOREIGN KEY (organisation_id, project_id, feature_flag_id)
        REFERENCES feature_flags (organisation_id, project_id, id)
        ON DELETE CASCADE
);

CREATE INDEX environment_flag_configs_flag_idx
    ON environment_flag_configs (feature_flag_id, environment_id);

-- +goose Down
DROP TABLE environment_flag_configs;
DROP TABLE feature_flags;
DROP TABLE environments;
DROP TABLE projects;
DROP TABLE organisation_memberships;
DROP TABLE organisations;
DROP TABLE users;
