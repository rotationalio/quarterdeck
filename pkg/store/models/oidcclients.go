package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/ulid"
)

type OIDCClient struct {
	Model
	CreatedBy ulid.ULID
	Revoked   sql.NullTime

	// OIDC spec descriptive fields

	ClientName string           // Descriptive name
	ClientURI  sql.NullString   // Main page, or "about us" type of page
	LogoURI    sql.NullString   // Logo image
	PolicyURI  sql.NullString   // Privacy policy page
	TOSURI     sql.NullString   // Terms of service page
	Contacts   []sql.NullString // Email addresses for the client

	// OIDC spec technical fields

	ClientID     string
	Secret       string
	RedirectURIs []string
}

type OIDCClientList struct {
	Page        *Page
	OIDCClients []*OIDCClient
}

//===========================================================================
// Scanning and Params
//===========================================================================

// Scanner is an interface for scanning database rows into the OIDCClient struct.
func (k *OIDCClient) Scan(scanner Scanner) (err error) {
	var redirectURIsJSON, contactsJSON sql.NullString

	if err = scanner.Scan(
		&k.ID,
		&k.ClientName,
		&k.ClientURI,
		&k.LogoURI,
		&k.PolicyURI,
		&k.TOSURI,
		&redirectURIsJSON,
		&contactsJSON,
		&k.ClientID,
		&k.Secret,
		&k.CreatedBy,
		&k.Revoked,
		&k.Created,
		&k.Modified,
	); err != nil {
		return err
	}

	if redirectURIsJSON.Valid && redirectURIsJSON.String != "" {
		_ = json.Unmarshal([]byte(redirectURIsJSON.String), &k.RedirectURIs)
	} else {
		k.RedirectURIs = nil
	}

	if contactsJSON.Valid && contactsJSON.String != "" {
		var strs []string
		_ = json.Unmarshal([]byte(contactsJSON.String), &strs)
		k.Contacts = make([]sql.NullString, len(strs))
		for i, s := range strs {
			k.Contacts[i] = sql.NullString{Valid: true, String: s}
		}
	} else {
		k.Contacts = nil
	}

	return nil
}

// ScanSummary scans an OIDCClient struct from a database row, excluding the Secret field.
func (k *OIDCClient) ScanSummary(scanner Scanner) (err error) {
	var redirectURIsJSON, contactsJSON sql.NullString

	if err = scanner.Scan(
		&k.ID,
		&k.ClientName,
		&k.ClientURI,
		&k.LogoURI,
		&k.PolicyURI,
		&k.TOSURI,
		&redirectURIsJSON,
		&contactsJSON,
		&k.ClientID,
		&k.CreatedBy,
		&k.Revoked,
		&k.Created,
		&k.Modified,
	); err != nil {
		return err
	}

	if redirectURIsJSON.Valid && redirectURIsJSON.String != "" {
		_ = json.Unmarshal([]byte(redirectURIsJSON.String), &k.RedirectURIs)
	} else {
		k.RedirectURIs = nil
	}

	if contactsJSON.Valid && contactsJSON.String != "" {
		var strs []string
		_ = json.Unmarshal([]byte(contactsJSON.String), &strs)
		k.Contacts = make([]sql.NullString, len(strs))
		for i, s := range strs {
			k.Contacts[i] = sql.NullString{Valid: true, String: s}
		}
	} else {
		k.Contacts = nil
	}

	k.Secret = ""

	return nil
}

// Params returns all OIDCClient fields as named params to be used in a SQL query.
func (k *OIDCClient) Params() []any {
	redirectURIs := []string{}
	if k.RedirectURIs != nil {
		redirectURIs = k.RedirectURIs
	}
	redirectURIsJSON, _ := json.Marshal(redirectURIs)

	contactsStrs := make([]string, 0, len(k.Contacts))
	for _, c := range k.Contacts {
		if c.Valid {
			contactsStrs = append(contactsStrs, c.String)
		}
	}
	contactsJSON, _ := json.Marshal(contactsStrs)

	return []any{
		sql.Named("id", k.ID),
		sql.Named("clientName", k.ClientName),
		sql.Named("clientURI", k.ClientURI),
		sql.Named("logoURI", k.LogoURI),
		sql.Named("policyURI", k.PolicyURI),
		sql.Named("tosURI", k.TOSURI),
		sql.Named("redirectURIs", string(redirectURIsJSON)),
		sql.Named("contacts", string(contactsJSON)),
		sql.Named("clientID", k.ClientID),
		sql.Named("secret", k.Secret),
		sql.Named("createdBy", k.CreatedBy),
		sql.Named("revoked", k.Revoked),
		sql.Named("created", k.Created),
		sql.Named("modified", k.Modified),
	}
}

//===========================================================================
// Validation
//===========================================================================

// Validate checks the OIDCClient against OpenID Connect Dynamic Client Registration
// Client Metadata requirements: URIs are valid URLs, contacts are valid emails,
// redirect_uris is required and for web clients must use https and not localhost.
// When gin.Mode() is DebugMode, http scheme and localhost/127.0.0.1/::1 are allowed.
func (k *OIDCClient) Validate() error {
	var errs []error

	// Required persistence fields
	if k.ClientID == "" {
		errs = append(errs, fmt.Errorf("client_id: required"))
	}
	if k.Secret == "" {
		errs = append(errs, fmt.Errorf("secret: required"))
	}
	if k.CreatedBy.IsZero() {
		errs = append(errs, fmt.Errorf("created_by: required"))
	}

	// redirect_uris: REQUIRED; at least one; each must be valid URL; web: https, no localhost (unless debug)
	if len(k.RedirectURIs) == 0 {
		errs = append(errs, fmt.Errorf("redirect_uris: at least one redirect URI is required"))
	} else {
		for i, u := range k.RedirectURIs {
			if u == "" {
				errs = append(errs, fmt.Errorf("redirect_uris[%d]: redirect URI cannot be empty", i))
				continue
			}
			parsed, err := url.Parse(u)
			if err != nil {
				errs = append(errs, fmt.Errorf("redirect_uris[%d]: %w", i, err))
				continue
			}
			if !parsed.IsAbs() || parsed.Scheme == "" || parsed.Host == "" {
				errs = append(errs, fmt.Errorf("redirect_uris[%d]: must be an absolute URL with scheme and host", i))
				continue
			}
			// application_type default is web: https only, no localhost (bypass when gin is in debug mode)
			if !(gin.Mode() == gin.DebugMode) {
				if parsed.Scheme != "https" {
					errs = append(errs, fmt.Errorf("redirect_uris[%d]: web clients must use https scheme", i))
				}
				if parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1" {
					errs = append(errs, fmt.Errorf("redirect_uris[%d]: web clients must not use localhost", i))
				}
			}
		}
	}

	// Optional URI metadata: when present, must be valid absolute URLs
	if k.ClientURI.Valid && k.ClientURI.String != "" {
		if err := validateURI("client_uri", k.ClientURI.String); err != nil {
			errs = append(errs, err)
		}
	}
	if k.LogoURI.Valid && k.LogoURI.String != "" {
		if err := validateURI("logo_uri", k.LogoURI.String); err != nil {
			errs = append(errs, err)
		}
	}
	if k.PolicyURI.Valid && k.PolicyURI.String != "" {
		if err := validateURI("policy_uri", k.PolicyURI.String); err != nil {
			errs = append(errs, err)
		}
	}
	if k.TOSURI.Valid && k.TOSURI.String != "" {
		if err := validateURI("tos_uri", k.TOSURI.String); err != nil {
			errs = append(errs, err)
		}
	}

	// contacts: optional; when present, each must be valid email
	for i, c := range k.Contacts {
		if !c.Valid || c.String == "" {
			continue
		}
		if _, err := mail.ParseAddress(c.String); err != nil {
			errs = append(errs, fmt.Errorf("contacts[%d]: invalid email: %w", i, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func validateURI(field, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	if !parsed.IsAbs() || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s: must be an absolute URL with scheme and host", field)
	}
	return nil
}
