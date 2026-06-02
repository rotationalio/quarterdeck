package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/ulid"

	"go.rtnl.ai/gimlet/auth"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
)

var (
	modelID  = ulid.MustParse("01JYMS2J4X5XKFWCGKSX5G1JMK")
	created  = time.Date(2025, 4, 7, 12, 21, 33, 00, time.UTC)
	modified = time.Date(2025, 5, 8, 24, 42, 55, 00, time.UTC)
)

func TestUserClaims(t *testing.T) {
	user := &User{
		BaseModel: BaseModel{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Name:  sql.NullString{Valid: true, String: "Carol King"},
		Email: "cking@example.com",
	}

	user.Roles.Load([]string{"Admin", "KeyManager"})
	user.Permissions.Load([]string{"read", "write", "delete"})

	claims, err := user.Claims()
	require.NoError(t, err, "expected no error when getting user claims")

	require.Equal(t, "", claims.ClientID, "expected empty ClientID for user claims")
	require.Equal(t, user.Name.String, claims.Name, "expected Name to match user Name")
	require.Equal(t, user.Email, claims.Email, "expected Email to match user Email")
	require.Equal(t, user.Gravatar(), claims.Gravatar, "expected Gravatar to match user Gravatar")
	require.Equal(t, []string{"Admin", "KeyManager"}, claims.Roles, "expected Roles to match user roles")
	require.Equal(t, []string{"read", "write", "delete"}, claims.Permissions, "expected Permissions to match user permissions")

	subject, userID, err := claims.SubjectID()
	require.NoError(t, err, "expected no error when getting subject ID")
	require.Equal(t, auth.SubjectUser, subject, "expected SubjectType to be User")
	require.Equal(t, user.ID, userID, "expected User ID to match claims subject ID")
}

func TestUserGravatar(t *testing.T) {
	user := &User{
		BaseModel: BaseModel{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Name:  sql.NullString{Valid: true, String: "Carol King"},
		Email: "cking@example.com",
	}

	require.Equal(t, "https://www.gravatar.com/avatar/0af35294c2926497116ff93ab0d139c0?d=identicon&r=pg&s=256", user.Gravatar(), "gravatar did not match expected")
}

func TestEmptyGravatar(t *testing.T) {
	user := &User{
		BaseModel: BaseModel{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Name:  sql.NullString{Valid: true, String: "Carol King"},
		Email: "",
	}

	require.Equal(t, "", user.Gravatar(), "gravatar should be empty when email is not set")
}
