package models

import (
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/x/gravatar"
)

var (
	gravatarOpts = &gravatar.Options{
		Size:          256,
		DefaultImage:  "identicon",
		ForceDefault:  false,
		Rating:        "pg",
		FileExtension: "",
	}
)

type User struct {
	Model
	Name          sql.NullString
	Email         string
	Password      string
	LastLogin     sql.NullTime
	EmailVerified bool
	roles         []*Role
	permissions   []string
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

func UserPageFrom(in *UserPage) (out *UserPage) {
	out = &UserPage{
		Page: Page{
			PageSize: DefaultPageSize,
		},
	}

	if in != nil {
		if in.PageSize > 0 {
			out.PageSize = in.PageSize
		}
		out.Role = in.Role
	}

	return out
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
		&u.EmailVerified,
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
		&u.EmailVerified,
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
		sql.Named("emailVerified", u.EmailVerified),
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
func (u *User) SetRoles(roles []*Role) {
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

//===========================================================================
// Helper Methods
//===========================================================================

func (u User) Claims() (claims *auth.Claims, err error) {
	claims = &auth.Claims{
		Name:        u.Name.String,
		Email:       u.Email,
		Gravatar:    u.Gravatar(),
		Permissions: u.Permissions(),
	}

	var roles []*Role
	if roles, err = u.Roles(); err != nil {
		return nil, err
	}

	claims.Roles = make([]string, 0, len(roles))
	for _, role := range roles {
		claims.Roles = append(claims.Roles, role.Title)
	}

	claims.SetSubjectID(auth.SubjectUser, u.ID)
	return claims, nil
}

func (u User) Gravatar() string {
	if u.Email == "" {
		return ""
	}
	return gravatar.New(u.Email, gravatarOpts)
}
