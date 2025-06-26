package models

import "database/sql"

type APIKey struct {
	Model
	Description sql.NullString
	ClientID    string
	Secret      string
	LastSeen    sql.NullTime
	permissions []string
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
		&k.LastSeen,
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
		&k.LastSeen,
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
		sql.Named("lastSeen", k.LastSeen),
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
