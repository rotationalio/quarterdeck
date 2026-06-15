package models

import (
	"database/sql"

	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/gravatar"
)

var gravatarOpts = &gravatar.Options{
	Size:          256,
	DefaultImage:  "identicon",
	ForceDefault:  false,
	Rating:        "pg",
	FileExtension: "",
}

type User struct {
	tidal.BaseModel
	Name          sql.NullString
	Email         string
	Password      string
	LastLogin     sql.NullTime
	EmailVerified bool
	Roles         []Role
	Permissions   []Permission
}

var _ tidal.Model = (*User)(nil)

func (u *User) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.List:
		return []string{
			"id",
			"name",
			"email",
			"last_login",
			"email_verified",
			"created",
			"modified",
		}
	case tidal.Update:
		return []string{
			"id",
			"name",
			"email",
			"modified",
		}
	default:
		return []string{
			"id",
			"name",
			"email",
			"password",
			"last_login",
			"email_verified",
			"created",
			"modified",
		}
	}
}

func (u *User) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", u.ID),
			sql.Named("name", u.Name),
			sql.Named("email", u.Email),
			sql.Named("modified", u.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("id", u.ID),
			sql.Named("name", u.Name),
			sql.Named("email", u.Email),
			sql.Named("password", u.Password),
			sql.Named("last_login", u.LastLogin),
			sql.Named("email_verified", u.EmailVerified),
			sql.Named("created", u.Created),
			sql.Named("modified", u.Modified),
		}
	}
}

func (u *User) Scan(op tidal.Operation, s tidal.Scanner) error {
	switch op {
	case tidal.List:
		return s.Scan(
			&u.ID,
			&u.Name,
			&u.Email,
			&u.LastLogin,
			&u.EmailVerified,
			&u.Created,
			&u.Modified,
		)
	default:
		return s.Scan(
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

func (u User) Claims() *auth.Claims {
	claims := &auth.Claims{
		Name:        u.Name.String,
		Email:       u.Email,
		Gravatar:    u.Gravatar(),
		Permissions: PermissionTitles(u.Permissions),
	}

	claims.Roles = make([]string, 0, len(u.Roles))
	for _, role := range u.Roles {
		claims.Roles = append(claims.Roles, role.Title)
	}

	claims.SetSubjectID(auth.SubjectUser, u.ID)
	return claims
}

func (u User) Gravatar() string {
	if u.Email == "" {
		return ""
	}
	return gravatar.New(u.Email, gravatarOpts)
}
