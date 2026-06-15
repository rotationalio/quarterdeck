package models

import (
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/fields"
	"go.rtnl.ai/ulid"
)

type OIDCClient struct {
	tidal.BaseModel

	CreatedBy ulid.ULID // ID of the user who created the client

	// OIDC spec descriptive fields

	ClientName   string             // Descriptive name
	ClientURI    sql.NullString     // Main page, or "about us" type of page
	LogoURI      sql.NullString     // Logo image
	PolicyURI    sql.NullString     // Privacy policy page
	TOSURI       sql.NullString     // Terms of service page
	RedirectURIs fields.StringArray // Redirect URIs
	Contacts     fields.StringArray // Contact email addresses for the client

	// OIDC spec technical fields

	ClientID string
	Secret   string
}

var _ tidal.Model = (*OIDCClient)(nil)
var _ tidal.Validator = (*OIDCClient)(nil)

func (c *OIDCClient) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.List:
		return []string{
			"id",
			"client_name",
			"client_uri",
			"logo_uri",
			"policy_uri",
			"tos_uri",
			"redirect_uris",
			"contacts",
			"client_id",
			"created_by",
			"created",
			"modified",
		}
	case tidal.Update:
		return []string{
			"id",
			"client_name",
			"client_uri",
			"logo_uri",
			"policy_uri",
			"tos_uri",
			"redirect_uris",
			"contacts",
			"modified",
		}
	default:
		return []string{
			"id",
			"client_name",
			"client_uri",
			"logo_uri",
			"policy_uri",
			"tos_uri",
			"redirect_uris",
			"contacts",
			"client_id",
			"secret",
			"created_by",
			"created",
			"modified",
		}
	}
}

func (c *OIDCClient) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", c.ID),
			sql.Named("client_name", c.ClientName),
			sql.Named("client_uri", c.ClientURI),
			sql.Named("logo_uri", c.LogoURI),
			sql.Named("policy_uri", c.PolicyURI),
			sql.Named("tos_uri", c.TOSURI),
			sql.Named("redirect_uris", c.RedirectURIs),
			sql.Named("contacts", c.Contacts),
			sql.Named("modified", c.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("id", c.ID),
			sql.Named("client_name", c.ClientName),
			sql.Named("client_uri", c.ClientURI),
			sql.Named("logo_uri", c.LogoURI),
			sql.Named("policy_uri", c.PolicyURI),
			sql.Named("tos_uri", c.TOSURI),
			sql.Named("redirect_uris", c.RedirectURIs),
			sql.Named("contacts", c.Contacts),
			sql.Named("client_id", c.ClientID),
			sql.Named("secret", c.Secret),
			sql.Named("created_by", c.CreatedBy),
			sql.Named("created", c.Created),
			sql.Named("modified", c.Modified),
		}
	}
}

func (c *OIDCClient) Scan(op tidal.Operation, s tidal.Scanner) error {
	switch op {
	case tidal.List:
		return s.Scan(
			&c.ID,
			&c.ClientName,
			&c.ClientURI,
			&c.LogoURI,
			&c.PolicyURI,
			&c.TOSURI,
			&c.RedirectURIs,
			&c.Contacts,
			&c.ClientID,
			&c.CreatedBy,
			&c.Created,
			&c.Modified,
		)
	default:
		return s.Scan(
			&c.ID,
			&c.ClientName,
			&c.ClientURI,
			&c.LogoURI,
			&c.PolicyURI,
			&c.TOSURI,
			&c.RedirectURIs,
			&c.Contacts,
			&c.ClientID,
			&c.Secret,
			&c.CreatedBy,
			&c.Created,
			&c.Modified,
		)
	}
}

func (c *OIDCClient) Validate(op tidal.Operation) error {
	if err := c.BaseModel.Validate(op); err != nil {
		return err
	}
	if op == tidal.Create {
		if c.ClientID == "" || c.Secret == "" {
			return qerrors.ErrZeroValuedNotNull
		}
	}
	return nil
}
