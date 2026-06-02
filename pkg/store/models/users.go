package models

import (
	"database/sql"

	"go.rtnl.ai/gimlet/auth"
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
	BaseModel
	Name          sql.NullString
	Email         string
	Password      string
	LastLogin     sql.NullTime
	EmailVerified bool

	// Associated Fields
	Roles       Roles
	Permissions Permissions
}

var (
	_ Model = (*User)(nil)
)

var (
	userFields = [8]string{
		"id",
		"name",
		"email",
		"password",
		"last_login",
		"email_verified",
		"created",
		"modified",
	}

	userSummaryFields = [6]string{
		"id",
		"name",
		"email",
		"last_login",
		"created",
		"modified",
	}
)

//===========================================================================
// Scanning and Params
//===========================================================================

// Scan the User struct from a database row.
func (u *User) Scan(op Operation, scanner Scanner) error {
	switch op {
	case List:
		return scanner.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.LastLogin,
			&u.Created,
			&u.Modified,
		)
	default:
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
}

func (u *User) Fields(op Operation) []string {
	switch op {
	case List:
		return userSummaryFields[:]
	default:
		return userFields[:]
	}
}

// Params returns all user fields as named params to be used in a SQL query.
func (u User) Params(_ Operation) []sql.NamedArg {
	return []sql.NamedArg{
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
// Helper Methods
//===========================================================================

func (u User) Claims() (claims *auth.Claims, err error) {
	claims = &auth.Claims{
		Name:        u.Name.String,
		Email:       u.Email,
		Gravatar:    u.Gravatar(),
		Roles:       make([]string, 0, len(u.Roles)),
		Permissions: make([]string, 0, len(u.Permissions)),
	}

	for _, role := range u.Roles {
		claims.Roles = append(claims.Roles, role.Title)
	}

	for _, permission := range u.Permissions {
		claims.Permissions = append(claims.Permissions, permission.Title)
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
