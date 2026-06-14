-- Primary schema for Quarterdeck (SQLite).

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

CREATE TABLE IF NOT EXISTS roles (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT NOT NULL UNIQUE,
    description     TEXT,
    is_default      BOOLEAN DEFAULT false NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT NOT NULL UNIQUE,
    description     TEXT,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id         INTEGER NOT NULL,
    permission_id   INTEGER NOT NULL,
    created         DATETIME NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id         TEXT NOT NULL,
    role_id         INTEGER NOT NULL,
    created         DATETIME NOT NULL,
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS api_key_permissions (
    api_key_id      TEXT NOT NULL,
    permission_id   INTEGER NOT NULL,
    created         DATETIME NOT NULL,
    PRIMARY KEY (api_key_id, permission_id),
    FOREIGN KEY (api_key_id) REFERENCES api_keys (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE CASCADE
);

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

CREATE TABLE IF NOT EXISTS oidc_clients (
    id              TEXT PRIMARY KEY,
    client_name     TEXT NOT NULL,
    client_uri      TEXT,
    logo_uri        TEXT,
    policy_uri      TEXT,
    tos_uri         TEXT,
    redirect_uris   BLOB,
    contacts        BLOB,
    client_id       TEXT NOT NULL UNIQUE,
    secret          TEXT NOT NULL UNIQUE,
    created_by      TEXT NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

DROP VIEW IF EXISTS user_permissions;
CREATE VIEW user_permissions AS
    SELECT DISTINCT u.id AS user_id, p.title AS permission
        FROM users u
        JOIN user_roles ur ON ur.user_id = u.id
        JOIN role_permissions rp ON rp.role_id = ur.role_id
        JOIN permissions p ON p.id = rp.permission_id
;

DROP VIEW IF EXISTS api_key_permission_list;
CREATE VIEW api_key_permission_list AS
    SELECT k.api_key_id, p.title AS permission
        FROM api_key_permissions k
        JOIN permissions p ON p.id = k.permission_id
;
