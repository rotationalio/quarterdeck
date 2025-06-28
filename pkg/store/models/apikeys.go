package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/ulid"
)

type APIKey struct {
	Model
	Description sql.NullString
	ClientID    string
	Secret      string
	CreatedBy   ulid.ULID
	LastSeen    sql.NullTime
	Revoked     sql.NullTime
	permissions []string
}

type APIKeyList struct {
	Page    *Page
	APIKeys []*APIKey
}

//===========================================================================
// Scanning and Params
//===========================================================================

// Scanner is an interface for scanning database rows into the APIKey struct.
func (k *APIKey) Scan(scanner Scanner) error {
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

// ScanSummary scans an APIKey struct from a database row, excluding the Secret field.
func (k *APIKey) ScanSummary(scanner Scanner) error {
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
}

// Params returns all APIKey fields as named params to be used in a SQL query.
func (k *APIKey) Params() []any {
	return []any{
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
// Associations
//===========================================================================

// Permissions returns the permissions associated with the APIKey, if set.
func (k APIKey) Permissions() []string {
	return k.permissions
}

// SetPermissions sets the permissions for the APIKey.
func (k *APIKey) SetPermissions(permissions []string) {
	k.permissions = permissions
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
