-- Initial schema for Quarterdeck authentication service.
-- NOTE: all primary keys are ULIDs but rather than using the 16 byte blob version of
-- the ULIDs we're using the string representation to make database queries easier and
-- because use of the sqlite3 storage backend isn't considered to be performance
-- intensive. NOTE: the oklog/v2 ulid package provides Scan for both []byte and string.
BEGIN;

-- Primary authentication table that holds email addresses and argon2 derived key
-- passwords that are used to authenticate users. Authorization data is defined by roles
-- that are assigned to users rather than specific permissions.  Metadata is the user's
-- name and last login but no other profile information is stored in this table.
CREATE TABLE IF NOT EXISTS users (
    id              TEXT PRIMARY KEY,
    name            TEXT,
    email           TEXT NOT NULL UNIQUE,
    password        TEXT NOT NULL UNIQUE,
    last_login      DATETIME,
    email_verified  BOOLEAN DEFAULT false NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Secondary authentication table that holds API keys for machine to machine access.
-- API Keys are identified both by a unique client ID (randomly generated string) and
-- by a Key ID (ULID). The client secret is an argon2 derived key that is used to
-- authenticate the client. APIKeys do not have roles but are specifically associated
-- with specific permissions. Metadat includes a description, the last time the key
-- was used for authentication and the user that created the key.
CREATE TABLE IF NOT EXISTS api_keys (
    id              TEXT PRIMARY KEY,
    description     TEXT,
    client_id       TEXT NOT NULL UNIQUE,
    secret          TEXT NOT NULL UNIQUE,
    created_by      TEXT NOT NULL,
    last_seen       DATETIME,
    revoked         DATETIME,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

-- Roles are essentially a set of permissions that can be assigned to users. Users can
-- have multiple roles and their permissions are the union of all roles assigned to them.
CREATE TABLE IF NOT EXISTS roles (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT NOT NULL UNIQUE,
    description     TEXT,
    is_default      BOOLEAN DEFAULT false NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Permissions authorize users and api keys to perform specific actions.
CREATE TABLE IF NOT EXISTS permissions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT NOT NULL UNIQUE,
    description     TEXT,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Maps permissions to roles
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id         INTEGER NOT NULL,
    permission_id   INTEGER NOT NULL,
    created         DATETIME NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

-- Maps roles to users
CREATE TABLE IF NOT EXISTS user_roles (
    user_id         TEXT NOT NULL,
    role_id         INTEGER NOT NULL,
    created         DATETIME NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

-- Maps permissions to api keys
CREATE TABLE IF NOT EXISTS api_key_permissions (
    api_key_id      TEXT NOT NULL,
    permission_id   INTEGER NOT NULL,
    created         DATETIME NOT NULL,
    PRIMARY KEY (api_key_id, permission_id),
    FOREIGN KEY (api_key_id) REFERENCES api_keys (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

-- VeroTokens are used to send a one time authentication link to a user via email. This
-- is used for resetting passwords, verifying email addresses, and to invite new users.
CREATE TABLE IF NOT EXISTS vero_tokens (
    id              TEXT PRIMARY KEY,
    token_type      TEXT NOT NULL,
    resource_id     TEXT DEFAULT NULL,
    email           TEXT NOT NULL,
    expiration      DATETIME NOT NULL,
    signature       BLOB DEFAULT NULL,
    sent_on         DATETIME DEFAULT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Allows selection of all permissions for a user based on their role
DROP VIEW IF EXISTS user_permissions;
CREATE VIEW user_permissions AS
    SELECT DISTINCT u.id AS user_id, p.title AS permission
        FROM users u
        JOIN user_roles ur ON ur.user_id = u.id
        JOIN role_permissions rp ON rp.role_id = ur.role_id
        JOIN permissions p ON p.id = rp.permission_id
;

-- Allows selection of all permissions for an api key by title
DROP VIEW IF EXISTS api_key_permission_list;
CREATE VIEW api_key_permission_list AS
    SELECT k.api_key_id, p.title AS permission
        FROM api_key_permissions k
        JOIN permissions p ON p.id = k.permission_id
;

COMMIT;