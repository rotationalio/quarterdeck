package models

import (
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/store/fields"
	"go.rtnl.ai/ulid"
)

type OIDCClient struct {
	BaseModel

	// OIDC spec descriptive fields
	ClientName string                 // Descriptive name
	ClientURI  sql.NullString         // Main page, or "about us" type of page
	LogoURI    sql.NullString         // Logo image
	PolicyURI  sql.NullString         // Privacy policy page
	TOSURI     sql.NullString         // Terms of service page
	Contacts   fields.NullStringArray // Email addresses for the client

	// OIDC spec technical fields
	ClientID     string
	Secret       string
	RedirectURIs fields.NullStringArray

	// Associated Fields
	CreatedBy ulid.ULID
}

var (
	_ Model = (*OIDCClient)(nil)
)

var (
	oidcclientFields = [13]string{
		"id",
		"client_name",
		"client_uri",
		"logo_uri",
		"policy_uri",
		"tos_uri",
		"contacts",
		"client_id",
		"secret",
		"redirect_uris",
		"created_by",
		"created",
		"modified",
	}

	oidcclientSummaryFields = [12]string{
		"id",
		"client_name",
		"client_uri",
		"logo_uri",
		"policy_uri",
		"tos_uri",
		"contacts",
		"client_id",
		"redirect_uris",
		"created_by",
		"created",
		"modified",
	}
)

//===========================================================================
// Model Methods
//===========================================================================

// Scanner is an interface for scanning database rows into the OIDCClient struct.
func (k *OIDCClient) Scan(op Operation, scanner Scanner) (err error) {
	switch op {
	case List:
		return scanner.Scan(
			&k.ID,
			&k.ClientName,
			&k.ClientURI,
			&k.LogoURI,
			&k.PolicyURI,
			&k.TOSURI,
			&k.Contacts,
			&k.ClientID,
			&k.RedirectURIs,
			&k.CreatedBy,
			&k.Created,
			&k.Modified,
		)
	default:
		return scanner.Scan(
			&k.ID,
			&k.ClientName,
			&k.ClientURI,
			&k.LogoURI,
			&k.PolicyURI,
			&k.TOSURI,
			&k.Contacts,
			&k.ClientID,
			&k.Secret,
			&k.RedirectURIs,
			&k.CreatedBy,
			&k.Created,
			&k.Modified,
		)
	}
}

func (k *OIDCClient) Fields(op Operation) []string {
	switch op {
	case List:
		return oidcclientSummaryFields[:]
	default:
		return oidcclientFields[:]
	}
}

// Params returns all OIDCClient fields as named params to be used in a SQL query.
func (k *OIDCClient) Params(_ Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("id", k.ID),
		sql.Named("clientName", k.ClientName),
		sql.Named("clientURI", k.ClientURI),
		sql.Named("logoURI", k.LogoURI),
		sql.Named("policyURI", k.PolicyURI),
		sql.Named("tosURI", k.TOSURI),
		sql.Named("contacts", k.Contacts),
		sql.Named("clientID", k.ClientID),
		sql.Named("secret", k.Secret),
		sql.Named("redirectURIs", k.RedirectURIs),
		sql.Named("createdBy", k.CreatedBy),
		sql.Named("created", k.Created),
		sql.Named("modified", k.Modified),
	}
}
