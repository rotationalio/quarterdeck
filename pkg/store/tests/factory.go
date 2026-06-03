package tests

import (
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/randstr"
)

type Factory[M models.Model] interface {
	ID(M) sql.NamedArg
	Make() M
}

//============================================================================
// API Key Factory
//============================================================================

type APIKeyFactory struct {
	CreatedBy ulid.ULID
}

var (
	_ Factory[*models.APIKey] = (*APIKeyFactory)(nil)
)

func (f *APIKeyFactory) ID(key *models.APIKey) sql.NamedArg {
	return sql.NamedArg{Name: "id", Value: key.ID}
}

func (f *APIKeyFactory) Make() *models.APIKey {
	secret := passwords.ClientSecret()
	derived, _ := passwords.CreateDerivedKey(secret)

	return &models.APIKey{
		Description: sql.NullString{String: lorem.Sentence(3, 7), Valid: true},
		ClientID:    passwords.ClientID(),
		Secret:      derived,
		CreatedBy:   f.CreatedBy,
	}
}

//============================================================================
// User Factory
//============================================================================

type UserFactory struct{}

var (
	_ Factory[*models.User] = (*UserFactory)(nil)
)

func (f *UserFactory) ID(user *models.User) sql.NamedArg {
	return sql.NamedArg{Name: "id", Value: user.ID}
}

func (f *UserFactory) Make() *models.User {
	password := randstr.Password(18)
	derived, _ := passwords.CreateDerivedKey(password)

	return &models.User{
		Name:          sql.NullString{String: lorem.Name(), Valid: true},
		Email:         lorem.Email(),
		Password:      derived,
		EmailVerified: true,
	}
}
