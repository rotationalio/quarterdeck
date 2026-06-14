-- Primary schema for Quarterdeck (Postgres).
CREATE TABLE IF NOT EXISTS users (
    id BYTEA PRIMARY KEY,
    name TEXT,
    email TEXT NOT NULL UNIQUE,
    PASSWORD TEXT NOT NULL UNIQUE,
    last_login TIMESTAMPTZ,
    email_verified BOOLEAN DEFAULT false NOT NULL,
    created TIMESTAMPTZ NOT NULL,
    modified TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS api_keys (
    id BYTEA PRIMARY KEY,
    description TEXT,
    client_id TEXT NOT NULL UNIQUE,
    secret TEXT NOT NULL UNIQUE,
    created_by BYTEA NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_seen TIMESTAMPTZ,
    revoked TIMESTAMPTZ,
    created TIMESTAMPTZ NOT NULL,
    modified TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    description TEXT,
    is_default BOOLEAN DEFAULT false NOT NULL,
    created TIMESTAMPTZ NOT NULL,
    modified TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    description TEXT,
    created TIMESTAMPTZ NOT NULL,
    modified TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    created TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BYTEA NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    created TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS api_key_permissions (
    api_key_id BYTEA NOT NULL REFERENCES api_keys (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    created TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (api_key_id, permission_id)
);

CREATE TABLE IF NOT EXISTS vero_tokens (
    id BYTEA PRIMARY KEY,
    token_type TEXT NOT NULL,
    resource_id BYTEA DEFAULT NULL,
    email TEXT NOT NULL,
    expiration TIMESTAMPTZ NOT NULL,
    signature BYTEA DEFAULT NULL,
    sent_on TIMESTAMPTZ DEFAULT NULL,
    created TIMESTAMPTZ NOT NULL,
    modified TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS oidc_clients (
    id BYTEA PRIMARY KEY,
    client_name TEXT NOT NULL,
    client_uri TEXT,
    logo_uri TEXT,
    policy_uri TEXT,
    tos_uri TEXT,
    redirect_uris JSONB,
    contacts JSONB,
    client_id TEXT NOT NULL UNIQUE,
    secret TEXT NOT NULL UNIQUE,
    created_by BYTEA NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created TIMESTAMPTZ NOT NULL,
    modified TIMESTAMPTZ NOT NULL
);

CREATE OR REPLACE VIEW user_permissions AS
SELECT DISTINCT u.id AS user_id,
    p.title AS permission
FROM users u
    JOIN user_roles ur ON ur.user_id = u.id
    JOIN role_permissions rp ON rp.role_id = ur.role_id
    JOIN permissions p ON p.id = rp.permission_id;

CREATE OR REPLACE VIEW api_key_permission_list AS
SELECT k.api_key_id,
    p.title AS permission
FROM api_key_permissions k
    JOIN permissions p ON p.id = k.permission_id;
