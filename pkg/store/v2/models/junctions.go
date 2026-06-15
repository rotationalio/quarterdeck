package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// UserRole Junction Table
//===========================================================================

// UserRole is a row in the user_roles junction table.
type UserRole struct {
	UserID  ulid.ULID
	RoleID  int64
	Created time.Time
}

var _ tidal.Model = (*UserRole)(nil)
var _ tidal.Preparer = (*UserRole)(nil)

func (ur *UserRole) Fields(op tidal.Operation) []string {
	return []string{
		"user_id",
		"role_id",
		"created",
	}
}

func (ur *UserRole) Params(op tidal.Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("user_id", ur.UserID),
		sql.Named("role_id", ur.RoleID),
		sql.Named("created", ur.Created),
	}
}

func (ur *UserRole) Scan(op tidal.Operation, s tidal.Scanner) error {
	return s.Scan(
		&ur.UserID,
		&ur.RoleID,
		&ur.Created,
	)
}

func (ur *UserRole) Prepare(op tidal.Operation) {
	if op == tidal.Create && ur.Created.IsZero() {
		ur.Created = time.Now().UTC()
	}
}

//===========================================================================
// RolePermission Junction Table
//===========================================================================

// RolePermission is a row in the role_permissions junction table.
type RolePermission struct {
	RoleID       int64
	PermissionID int64
	Created      time.Time
}

var _ tidal.Model = (*RolePermission)(nil)
var _ tidal.Preparer = (*RolePermission)(nil)

func (rp *RolePermission) Fields(op tidal.Operation) []string {
	return []string{
		"role_id",
		"permission_id",
		"created",
	}
}

func (rp *RolePermission) Params(op tidal.Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("role_id", rp.RoleID),
		sql.Named("permission_id", rp.PermissionID),
		sql.Named("created", rp.Created),
	}
}

func (rp *RolePermission) Scan(op tidal.Operation, s tidal.Scanner) error {
	return s.Scan(
		&rp.RoleID,
		&rp.PermissionID,
		&rp.Created,
	)
}

func (rp *RolePermission) Prepare(op tidal.Operation) {
	if op == tidal.Create && rp.Created.IsZero() {
		rp.Created = time.Now().UTC()
	}
}

//===========================================================================
// APIKeyPermission Junction Table
//===========================================================================

// APIKeyPermission is a row in the api_key_permissions junction table.
type APIKeyPermission struct {
	APIKeyID     ulid.ULID
	PermissionID int64
	Created      time.Time
}

var _ tidal.Model = (*APIKeyPermission)(nil)
var _ tidal.Preparer = (*APIKeyPermission)(nil)

func (ap *APIKeyPermission) Fields(op tidal.Operation) []string {
	return []string{
		"api_key_id",
		"permission_id",
		"created",
	}
}

func (ap *APIKeyPermission) Params(op tidal.Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("api_key_id", ap.APIKeyID),
		sql.Named("permission_id", ap.PermissionID),
		sql.Named("created", ap.Created),
	}
}

func (ap *APIKeyPermission) Scan(op tidal.Operation, s tidal.Scanner) error {
	return s.Scan(
		&ap.APIKeyID,
		&ap.PermissionID,
		&ap.Created,
	)
}

func (ap *APIKeyPermission) Prepare(op tidal.Operation) {
	if op == tidal.Create && ap.Created.IsZero() {
		ap.Created = time.Now().UTC()
	}
}
