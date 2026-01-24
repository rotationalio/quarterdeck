package models

import (
	"database/sql"
	"net/url"
)

type Application struct {
	Model
	DisplayName          string // Application's display name
	OrgDisplayName       string // Organization's name that owns the application
	SupportEmail         string // Support email for the application
	ClientID             string // ID for the application for OIDC use
	ClientSecret         string // Secret for the application for OIDC use
	NewUserEmailTemplate string // Go template string for new user emails //FIXME: add to db table and funcs
	baseURL              string // Application's base URL; all other paths are appended to this base
	oidcRedirectPath     string // OIDC login redirect path for this application
}

type ApplicationList struct {
	Page         *Page
	Applications []*Application
}

// ===========================================================================
// URL Helpers
// ===========================================================================

// Returns the base URL for the application.
func (a *Application) BaseURL() (url *url.URL, err error) {
	return url.Parse(a.baseURL)
}

// Returns the URL that is approved for OIDC login redirects for the application.
func (a *Application) OIDCRedirectURL() (u *url.URL, err error) {
	if u, err = a.BaseURL(); err != nil {
		return nil, err
	}
	return u.JoinPath(a.oidcRedirectPath), nil
}

//===========================================================================
// Scanning and Params
//===========================================================================

// Scanner is an interface for scanning database rows into the Application
// struct.
func (a *Application) Scan(scanner Scanner) error {
	return scanner.Scan(
		&a.ID,
		&a.DisplayName,
		&a.OrgDisplayName,
		&a.SupportEmail,
		&a.ClientID,
		&a.ClientSecret,
		&a.baseURL,
		&a.oidcRedirectPath,
		&a.Created,
		&a.Modified,
	)
}

// Scanner is an interface for scanning database rows into the Application
// struct, excluding ClientSecret.
func (a *Application) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&a.ID,
		&a.DisplayName,
		&a.OrgDisplayName,
		&a.SupportEmail,
		&a.ClientID,
		// EXCLUDE 'ClientSecret'
		&a.baseURL,
		&a.oidcRedirectPath,
		&a.Created,
		&a.Modified,
	)
}

// Params returns all Application fields as named params to be used in a SQL
// query.
func (a Application) Params() []any {
	return []any{
		sql.Named("id", a.ID),
		sql.Named("display_name", a.DisplayName),
		sql.Named("org_display_name", a.OrgDisplayName),
		sql.Named("support_email", a.SupportEmail),
		sql.Named("client_id", a.ClientID),
		sql.Named("client_secret", a.ClientSecret),
		sql.Named("base_url", a.baseURL),
		sql.Named("oidc_redirect_path", a.oidcRedirectPath),
		sql.Named("created", a.Created),
		sql.Named("modified", a.Modified),
	}
}
