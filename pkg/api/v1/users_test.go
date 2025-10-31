package api_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func TestNewUser(t *testing.T) {
	id := ulid.MakeSecure()
	now := time.Now()

	modelUser := &models.User{
		Model: models.Model{
			ID:       id,
			Created:  now.Add(-2 * time.Hour),
			Modified: now.Add(-1 * time.Hour),
		},
		Name:          sql.NullString{Valid: true, String: "Some User"},
		Email:         "user@example.com",
		Password:      "not_a_valid_derived_key",
		LastLogin:     sql.NullTime{Valid: true, Time: now},
		EmailVerified: true,
	}
	modelUser.SetRoles([]*models.Role{
		{ID: 123, Title: "role", Description: "description is not used"},
	})
	modelUser.SetPermissions([]string{"one", "two"})

	apiUser, err := api.NewUser(modelUser)
	require.NoError(t, err)
	require.NotNil(t, apiUser)
	require.Equal(t, modelUser.ID, apiUser.ID)
	require.Equal(t, modelUser.Created, apiUser.Created)
	require.Equal(t, modelUser.Modified, apiUser.Modified)
	require.Equal(t, modelUser.Name.String, apiUser.Name)
	require.Equal(t, modelUser.Email, apiUser.Email)
	require.Equal(t, modelUser.LastLogin.Time, apiUser.LastLogin)
	require.Equal(t, []*api.Role{{ID: 123, Title: "role"}}, apiUser.Roles)
	require.Equal(t, apiUser.Permissions, modelUser.Permissions())
}

func TestValidateUser(t *testing.T) {
	t.Run("ValidEmails", func(t *testing.T) {
		emails := []string{
			"Some User <email@example.com>",
			"\"Some User\" <email@example.com>",
			"email@example.com",
			"firstname.lastname@example.com",
			"firstname-lastname@example.com",
			"firstname+lastname@example.com",
			"email@subdomain.example.com",
			"email@123.123.123.123",
			"email@[123.123.123.123]",
			"\"email\"@example.com",
			"1234567890@example.com",
			"email@example-one.com",
			"_______@example.com",
			"email@example.tld",
			"危ない@example.co.jp",
			"email@localhost",
			"email@localhost.local",
		}

		for _, email := range emails {
			user := &api.User{
				Email: email,
			}

			require.NoErrorf(t, user.Validate(), "should be valid: %s", email)
		}
	})

	t.Run("ValidNameEmail", func(t *testing.T) {
		user := &api.User{
			Name:  "Some User",
			Email: "user@example.com",
		}

		require.NoError(t, user.Validate())
	})

	t.Run("ValidNilRoles", func(t *testing.T) {
		user := &api.User{
			Name:  "Some User",
			Email: "user@example.com",
			Roles: nil,
		}

		require.NoError(t, user.Validate())
	})

	t.Run("ValidEmptyRoles", func(t *testing.T) {
		user := &api.User{
			Name:  "Some User",
			Email: "user@example.com",
			Roles: make([]*api.Role, 0),
		}

		require.NoError(t, user.Validate())
	})

	t.Run("ValidSingleRole", func(t *testing.T) {
		user := &api.User{
			Name:  "Some User",
			Email: "user@example.com",
			Roles: []*api.Role{
				{
					ID:    123,
					Title: "a_role",
				},
			},
		}

		require.NoError(t, user.Validate())
	})

	t.Run("ValidMultipleRoles", func(t *testing.T) {
		user := &api.User{
			Name:  "Some User",
			Email: "user@example.com",
			Roles: []*api.Role{
				{
					ID:    123,
					Title: "a_role",
				},
				{
					ID:    321,
					Title: "b_roll",
				},
				{
					ID:    808,
					Title: "kings_hawaiian_roll",
				},
			},
		}

		require.NoError(t, user.Validate())
	})

	t.Run("ValidMultipleZeroedRoles", func(t *testing.T) {
		user := &api.User{
			Name:  "Some User",
			Email: "user@example.com",
			Roles: make([]*api.Role, 5),
		}

		require.NoError(t, user.Validate())
	})

	t.Run("ValidZeroLengthPermissions", func(t *testing.T) {
		user := &api.User{
			Name:        "Some User",
			Email:       "user@example.com",
			Permissions: make([]string, 0),
		}

		require.NoError(t, user.Validate())
	})

	t.Run("InvalidNonZeroID", func(t *testing.T) {
		user := &api.User{
			Email: "user@example.com",
			ID:    ulid.MakeSecure(),
		}

		require.EqualError(t, user.Validate(), api.ReadOnlyField("id").Error())
	})

	t.Run("InvalidNonZeroLastLogin", func(t *testing.T) {
		user := &api.User{
			Email:     "user@example.com",
			LastLogin: time.Now(),
		}

		require.EqualError(t, user.Validate(), api.ReadOnlyField("last_login").Error())
	})

	t.Run("InvalidNonZeroPermissions", func(t *testing.T) {
		user := &api.User{
			Email:       "user@example.com",
			Permissions: []string{"a_permission", "b_permission"},
		}

		require.EqualError(t, user.Validate(), api.ReadOnlyField("permissions").Error())
	})

	t.Run("InvalidNonZeroCreated", func(t *testing.T) {
		user := &api.User{
			Email:   "user@example.com",
			Created: time.Now(),
		}

		require.EqualError(t, user.Validate(), api.ReadOnlyField("created").Error())
	})

	t.Run("InvalidNonZeroModified", func(t *testing.T) {
		user := &api.User{
			Email:    "user@example.com",
			Modified: time.Now(),
		}

		require.EqualError(t, user.Validate(), api.ReadOnlyField("modified").Error())
	})

	t.Run("InvalidEmails", func(t *testing.T) {
		emails := []string{
			"plainaddress",
			"#@%^%#$@#$@#.com",
			"@example.com",
			"email.example.com",
			"email@example@example.com",
			".email@example.com",
			"email.@example.com",
			"email..email@example.com",
			"email@example..com",
			"Abc..123@example.com",
		}

		for _, email := range emails {
			user := &api.User{
				Email: email,
			}

			require.Errorf(t, user.Validate(), "should be invalid: %s", email)
		}
	})
}
