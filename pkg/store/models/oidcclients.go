package models

import (
	"database/sql"
	"encoding/json"

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
