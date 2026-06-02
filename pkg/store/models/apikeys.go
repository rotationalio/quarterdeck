package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/ulid"
)

type APIKey struct {
	BaseModel
	Description sql.NullString
	ClientID    string
	Secret      string
	CreatedBy   ulid.ULID
	LastSeen    sql.NullTime
	Revoked     sql.NullTime

	// Associated fields
	Permissions Permissions
}

var (
	_ Model = (*APIKey)(nil)
)

var (
	apikeyFields = [9]string{
		"id",
		"description",
		"client_id",
		"secret",
		"created_by",
		"last_seen",
		"revoked",
		"created",
		"modified",
	}

	apikeySummaryFields = [8]string{
		"id",
		"description",
		"client_id",
		"created_by",
		"last_seen",
		"revoked",
		"created",
		"modified",
	}
)

//===========================================================================
// Scanning and Params
//===========================================================================

// Scanner is an interface for scanning database rows into the APIKey struct.
func (k *APIKey) Scan(op Operation, scanner Scanner) error {
	switch op {
	case List:
		return scanner.Scan(
			&k.ID,
			&k.Description,
			&k.ClientID,
			&k.CreatedBy,
			&k.LastSeen,
			&k.Revoked,
			&k.Created,
			&k.Modified,
		)
	default:
		return scanner.Scan(
			&k.ID,
			&k.Description,
			&k.ClientID,
			&k.Secret,
			&k.CreatedBy,
			&k.LastSeen,
			&k.Revoked,
			&k.Created,
			&k.Modified,
		)
	}
}

func (k *APIKey) Fields(op Operation) []string {
	switch op {
	case List:
		return apikeySummaryFields[:]
	default:
		return apikeyFields[:]
	}
}

// Params returns all APIKey fields as named params to be used in a SQL query.
func (k *APIKey) Params(_ Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("id", k.ID),
		sql.Named("description", k.Description),
		sql.Named("clientID", k.ClientID),
		sql.Named("secret", k.Secret),
		sql.Named("createdBy", k.CreatedBy),
		sql.Named("lastSeen", k.LastSeen),
		sql.Named("revoked", k.Revoked),
		sql.Named("created", k.Created),
		sql.Named("modified", k.Modified),
	}
}

//===========================================================================
// APIKey Status
//===========================================================================

// API Keys are considered stale if they have not been used in the last 3 months or so.
const APIKeyStalenessThreshold = 90 * 24 * time.Hour

// Status of the APIKey based on the LastUsed timestamp if the api keys have not been
// revoked. If the keys have never been used the unused status is returned; if they have
// not been used in 90 days then the stale status is returned; otherwise the apikey is
// considered active unless it has been revoked.
func (k *APIKey) Status() enum.APIKeyStatus {
	if k.Revoked.Valid || !k.Revoked.Time.IsZero() {
		return enum.APIKeyStatusRevoked
	}

	if !k.LastSeen.Valid || k.LastSeen.Time.IsZero() {
		return enum.APIKeyStatusUnused
	}

	if time.Since(k.LastSeen.Time) > APIKeyStalenessThreshold {
		return enum.APIKeyStatusStale
	}

	return enum.APIKeyStatusActive
}

//===========================================================================
// Helper Methods
//===========================================================================

func (k APIKey) Claims() *auth.Claims {
	claims := &auth.Claims{
		ClientID:    k.ClientID,
		Permissions: k.Permissions.List(),
	}

	claims.SetSubjectID(auth.SubjectAPIKey, k.ID)
	return claims
}
