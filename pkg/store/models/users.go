package models

import (
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

type User struct {
	Model
	Name        sql.NullString
	Email       string
	Password    string
	LastLogin   sql.NullTime
	roles       []*Role
	permissions []string
}

type UserList struct {
	Page  *UserPage
	Users []*User
}

// UserPage allows a list of paginated users to be optionally filtered by role.
type UserPage struct {
	Page
	Role string `json:"role,omitempty"`
}

//===========================================================================
// Scanning and Params
//===========================================================================

// Scan the User struct from a database row.
func (u *User) Scan(scanner Scanner) error {
	return scanner.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.LastLogin,
		&u.Created,
		&u.Modified,
	)
}

// ScanSummary scans a User struct from a database row, excluding the Password field.
func (u *User) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.LastLogin,
		&u.Created,
		&u.Modified,
	)
}

// Params returns all user fields as named params to be used in a SQL query.
func (u User) Params() []any {
	return []any{
		sql.Named("id", u.ID),
		sql.Named("name", u.Name),
		sql.Named("email", u.Email),
		sql.Named("password", u.Password),
		sql.Named("lastLogin", u.LastLogin),
		sql.Named("created", u.Created),
		sql.Named("modified", u.Modified),
	}
}

//===========================================================================
// Associations
//===========================================================================

// Role returns the role associated with the user, if set, otherwise returns
// ErrMissingAssociation.
func (u User) Roles() ([]*Role, error) {
	if u.roles == nil {
		return nil, errors.ErrMissingAssociation
	}
	return u.roles, nil
}

// SetRole sets the role for the user and updates the RoleID field.
func (u *User) SetRole(roles ...*Role) {
	u.roles = roles
}

// Permissions returns the permissions associated with the user, if set.
func (u User) Permissions() []string {
	return u.permissions
}

// SetPermissions sets the permissions for the user.
func (u *User) SetPermissions(permissions []string) {
	u.permissions = permissions
}
