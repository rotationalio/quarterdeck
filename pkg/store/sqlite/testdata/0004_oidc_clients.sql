-- OIDC clients: full-metadata and minimal (required fields only).
-- created_by references Admin User (x'019545eb8b6e4c28bc6d4c684b20e9fd').
INSERT INTO oidc_clients (id, client_name, client_uri, logo_uri, policy_uri, tos_uri, redirect_uris, contacts, client_id, secret, created_by, created, modified) VALUES
    (x'019a0001000000000000000000000001', 'Full Metadata OIDC Client', 'https://example.com', 'https://example.com/logo.png', 'https://example.com/policy', 'https://example.com/tos', '["https://example.com/callback","https://app.example.com/cb"]', '["admin@example.com","support@example.com"]', 'OidcClient1FullMetadata', '$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=', x'019545eb8b6e4c28bc6d4c684b20e9fd', '2025-02-20T21:34:08Z', '2025-02-20T21:34:08Z'),
    (x'019a0002000000000000000000000002', 'Minimal Metadata OIDC Client', NULL, NULL, NULL, NULL, '["https://example.com/cb"]', NULL, 'OidcClient2Minimal', '$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=', x'019545eb8b6e4c28bc6d4c684b20e9fd', '2025-02-21T10:00:00Z', '2025-02-21T10:00:00Z');
;
