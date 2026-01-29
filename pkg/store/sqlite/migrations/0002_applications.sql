BEGIN;

-- Add applications table to Quarterdeck which will support OIDC login flows and
-- better multi-application support from Quarterdeck.
CREATE TABLE IF NOT EXISTS applications (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    org_display_name TEXT NOT NULL,
    support_email TEXT NOT NULL,
    client_id TEXT NOT NULL UNIQUE,
    client_secret TEXT NOT NULL UNIQUE,
    new_user_email_template_html TEXT NOT NULL,
    new_user_email_template_text TEXT NOT NULL,
    base_url TEXT NOT NULL UNIQUE,
    oidc_redirect_path TEXT NOT NULL,
    created DATETIME NOT NULL,
    modified DATETIME NOT NULL
);

-- Many of the lookups for this table will be by the 'client_id' so an index
-- will be more efficient.
CREATE INDEX idx_client_id ON applications (client_id);

COMMIT;
