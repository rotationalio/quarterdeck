-- OIDC clients table for OpenID Connect Dynamic Client Registration.
-- Stores client metadata and credentials; created_by references the user who registered the client.
BEGIN;

CREATE TABLE IF NOT EXISTS oidc_clients (
    id TEXT PRIMARY KEY,
    client_name TEXT NOT NULL,
    client_uri TEXT,
    logo_uri TEXT,
    policy_uri TEXT,
    tos_uri TEXT,
    redirect_uris TEXT,
    contacts TEXT,
    client_id TEXT NOT NULL UNIQUE,
    secret TEXT NOT NULL UNIQUE,
    created_by TEXT NOT NULL,
    created DATETIME NOT NULL,
    modified DATETIME NOT NULL,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

COMMIT;
