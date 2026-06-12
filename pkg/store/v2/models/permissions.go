package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

type Role struct {
	ID          int64
	Title       string
	Description string
	IsDefault   bool
	Created     time.Time
	Modified    time.Time
	permissions []*Permission
}

type RoleList struct {
	Page  *Page
	Roles []*Role
}

type Permission struct {
	ID          int64
	Title       string
	Description string
	Created     time.Time
	Modified    time.Time
}

type PermissionList struct {
	Page        *Page
	Permissions []*Permission
}

//===========================================================================
// Scanning and Params
//===========================================================================

// Scanner is an interface for scanning database rows into the Role structs.
func (r *Role) Scan(scanner Scanner) error {
	return scanner.Scan(
		&r.ID,
		&r.Title,
		&r.Description,
		&r.IsDefault,
		&r.Created,
		&r.Modified,
	)
}

// Params returns all Role fields as named params to be used in a SQL query.
func (r *Role) Params() []any {
	return []any{
		sql.Named("id", r.ID),
		sql.Named("title", r.Title),
		sql.Named("description", r.Description),
		sql.Named("isDefault", r.IsDefault),
		sql.Named("created", r.Created),
		sql.Named("modified", r.Modified),
	}
}

// Scan the Permission struct from a database row.
func (p *Permission) Scan(scanner Scanner) error {
	return scanner.Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.Created,
		&p.Modified,
	)
}

// Params returns all Permission fields as named params to be used in a SQL query.
func (p *Permission) Params() []any {
	return []any{
		sql.Named("id", p.ID),
		sql.Named("title", p.Title),
		sql.Named("description", p.Description),
		sql.Named("created", p.Created),
		sql.Named("modified", p.Modified),
	}
}

//===========================================================================
// Associations
//===========================================================================

// Permissions returns the permissions associated with the role, if set, otherwise
// returns ErrMissingAssociation.
func (r Role) Permissions() ([]*Permission, error) {
	if r.permissions == nil {
		return nil, errors.ErrMissingAssociation
	}
	return r.permissions, nil
}

// SetPermissions sets the permissions for the role.
func (r *Role) SetPermissions(permissions []*Permission) {
	r.permissions = permissions
}
