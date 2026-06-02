package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
)

// VeroTokens are sent via email to a user to allow them to securely authenticate to
// Quarterdeck for a one-time task such as resetting a password, verifying an email
// address, or accepting an invitation to a team.
type VeroToken struct {
	BaseModel
	TokenType  enum.TokenType
	ResourceID ulid.NullULID
	Email      string
	Expiration time.Time
	Signature  *vero.SignedToken
	SentOn     sql.NullTime
}

var (
	_ Model = (*VeroToken)(nil)
)

var (
	veroTokenFields = [9]string{
		"id",
		"token_type",
		"resource_id",
		"email",
		"expiration",
		"signature",
		"sent_on",
		"created",
		"modified",
	}
)

//===========================================================================
// Scanning and Params
//===========================================================================

// Scan is an interface for scanning database rows into the VeroToken struct.
func (v *VeroToken) Scan(op Operation, scanner Scanner) error {
	return scanner.Scan(
		&v.ID,
		&v.TokenType,
		&v.ResourceID,
		&v.Email,
		&v.Expiration,
		&v.Signature,
		&v.SentOn,
		&v.Created,
		&v.Modified,
	)
}

func (v *VeroToken) Fields(op Operation) []string {
	return veroTokenFields[:]
}

// Params returns all VeroToken fields as named params to be used in a SQL query.
func (v *VeroToken) Params(_ Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("id", v.ID),
		sql.Named("tokenType", v.TokenType),
		sql.Named("resourceID", v.ResourceID),
		sql.Named("email", v.Email),
		sql.Named("expiration", v.Expiration),
		sql.Named("signature", v.Signature),
		sql.Named("sentOn", v.SentOn),
		sql.Named("created", v.Created),
		sql.Named("modified", v.Modified),
	}
}

//===========================================================================
// Helpers
//===========================================================================

func (v *VeroToken) IsExpired() bool {
	return v.Expiration.IsZero() || time.Now().After(v.Expiration)
}
