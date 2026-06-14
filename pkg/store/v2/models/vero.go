package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
)

type VeroToken struct {
	tidal.BaseModel
	TokenType  enum.TokenType
	ResourceID ulid.NullULID
	Email      string
	Expiration time.Time
	Signature  *vero.SignedToken
	SentOn     sql.NullTime
}

var _ tidal.Model = (*VeroToken)(nil)

func (v *VeroToken) Fields(op tidal.Operation) []string {
	return []string{
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
}

func (v *VeroToken) Params(op tidal.Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("id", v.ID),
		sql.Named("token_type", v.TokenType),
		sql.Named("resource_id", v.ResourceID),
		sql.Named("email", v.Email),
		sql.Named("expiration", v.Expiration),
		sql.Named("signature", v.Signature),
		sql.Named("sent_on", v.SentOn),
		sql.Named("created", v.Created),
		sql.Named("modified", v.Modified),
	}
}

func (v *VeroToken) Scan(op tidal.Operation, s tidal.Scanner) error {
	return s.Scan(
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

func (v *VeroToken) IsExpired() bool {
	return v.Expiration.IsZero() || time.Now().After(v.Expiration)
}
