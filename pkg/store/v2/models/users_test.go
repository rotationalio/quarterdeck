package models_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// Database Conformance Tests
//=============================================================================

// TestUserCRUDConformance verifies User satisfies tidal CRUD shape expectations against the users table.
func (s *modelSuite) TestUserCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*User]{
		Table: "users",
		Create: func() *User {
			return &User{
				Name:          sql.NullString{Valid: true, String: "Conformance User"},
				Email:         fmt.Sprintf("conformance-%s@example.com", ulid.MakeSecure().String()),
				Password:      fmt.Sprintf("pw-%s", ulid.MakeSecure().String()),
				EmailVerified: true,
			}
		},
		Update: func(u *User) {
			u.Name = sql.NullString{Valid: true, String: "Updated Conformance User"}
		},
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan, tsuite.CRUDRoundTrip},
	})
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestUserScan verifies Scan behavior Shape conformance does not cover: List projection,
// nullable columns, and scanner error propagation.
func TestUserScan(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		// Setup: list projection omits password column.
		data := []any{
			ulid.MakeSecure().String(),
			"First Last",
			"email@example.com",
			time.Now(),
			true,
			time.Now(),
			time.Now().Add(1 * time.Hour),
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan row using List shape.
		model := &User{}
		err := model.Scan(tidal.List, mockScanner)
		require.NoError(t, err)
		mockScanner.AssertScanned(t, len(data))

		// Assert: password stays zero and remaining fields map correctly.
		require.Equal(t, data[0], model.ID.String())
		require.Equal(t, data[1], model.Name.String)
		require.Equal(t, data[2], model.Email)
		require.Zero(t, model.Password)
		require.Equal(t, data[3], model.LastLogin.Time)
		require.Equal(t, data[4], model.EmailVerified)
		require.Equal(t, data[5], model.Created)
		require.Equal(t, data[6], model.Modified)
	})

	t.Run("Nulls", func(t *testing.T) {
		// Setup: nullable columns and zero modified timestamp.
		data := []any{
			ulid.MakeSecure().String(),
			nil,
			"email@example.com",
			"Password",
			nil,
			false,
			time.Now(),
			time.Time{},
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan nullable row.
		model := &User{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.NoError(t, err)
		mockScanner.AssertScanned(t, len(data))

		// Assert: null SQL values produce invalid Null* fields and zero modified.
		require.False(t, model.Name.Valid)
		require.False(t, model.LastLogin.Valid)
		require.True(t, model.Modified.IsZero())
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &User{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.ErrorIs(t, err, ErrModelScan)
	})
}

// TestUserClaims verifies Claims maps user identity, roles, permissions, and subject ID for auth.
func TestUserClaims(t *testing.T) {
	user := &User{
		BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
		Name:      sql.NullString{Valid: true, String: "Carol King"},
		Email:     "cking@example.com",
	}

	user.Roles = []Role{{Title: "Admin"}, {Title: "KeyManager"}}
	user.Permissions = []Permission{
		{Title: "read"},
		{Title: "write"},
		{Title: "delete"},
	}

	claims := user.Claims()

	require.Equal(t, "", claims.ClientID)
	require.Equal(t, user.Name.String, claims.Name)
	require.Equal(t, user.Email, claims.Email)
	require.Equal(t, user.Gravatar(), claims.Gravatar)
	require.Equal(t, []string{"Admin", "KeyManager"}, claims.Roles)
	require.Equal(t, []string{"read", "write", "delete"}, claims.Permissions)

	subject, userID, err := claims.SubjectID()
	require.NoError(t, err)
	require.Equal(t, auth.SubjectUser, subject)
	require.Equal(t, user.ID, userID)
}

// TestUserGravatar verifies Gravatar returns the hashed URL for a non-empty email.
func TestUserGravatar(t *testing.T) {
	user := &User{
		BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
		Name:      sql.NullString{Valid: true, String: "Carol King"},
		Email:     "cking@example.com",
	}

	require.Equal(t, "https://www.gravatar.com/avatar/0af35294c2926497116ff93ab0d139c0?d=identicon&r=pg&s=256", user.Gravatar())
}

// TestEmptyGravatar verifies Gravatar returns an empty string when email is unset.
func TestEmptyGravatar(t *testing.T) {
	user := &User{
		BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
		Name:      sql.NullString{Valid: true, String: "Carol King"},
		Email:     "",
	}

	require.Equal(t, "", user.Gravatar())
}
