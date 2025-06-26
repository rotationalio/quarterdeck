package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
)

func TestRoleParams(t *testing.T) {
	role := &Role{
		ID:          int64(19),
		Title:       "Observer",
		Description: "Read only access",
		IsDefault:   false,
		Created:     created,
		Modified:    modified,
	}

	CheckParams(t, role.Params(),
		[]string{"id", "title", "description", "isDefault", "created", "modified"},
		[]any{role.ID, role.Title, role.Description, role.IsDefault, role.Created, role.Modified},
	)
}

func TestPermissionParams(t *testing.T) {
	perm := &Permission{
		ID:          int64(120),
		Title:       "dashboard:read",
		Description: "Read access to the dashboard",
		Created:     created,
		Modified:    modified,
	}

	CheckParams(t, perm.Params(),
		[]string{"id", "title", "description", "created", "modified"},
		[]any{perm.ID, perm.Title, perm.Description, perm.Created, perm.Modified},
	)
}

func TestRoleScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		data := []any{
			int64(19),          // ID
			"Observer",         // Title
			"Read only access", // Description
			false,              // IsDefault
			created,            // Created
			modified,           // Modified
		}

		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		role := &Role{}
		err := role.Scan(mockScanner)
		require.NoError(t, err, "expected no error when scanning with mock scanner")

		require.Equal(t, int64(19), role.ID, "expected ID to match")
		require.Equal(t, "Observer", role.Title, "expected Title to match")
		require.Equal(t, "Read only access", role.Description, "expected Description to match")
		require.False(t, role.IsDefault, "expected IsDefault to be false")
		require.Equal(t, created, role.Created, "expected Created to match")
		require.Equal(t, modified, role.Modified, "expected Modified to match")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		role := &Role{}
		err := role.Scan(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")

	})
}

func TestPermissionScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		data := []any{
			int64(120),                     // ID
			"dashboard:read",               // Title
			"Read access to the dashboard", // Description
			created,                        // Created
			modified,                       // Modified
		}

		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)
		permission := &Permission{}
		err := permission.Scan(mockScanner)
		require.NoError(t, err, "expected no error when scanning with mock scanner")

		require.Equal(t, int64(120), permission.ID, "expected ID to match")
		require.Equal(t, "dashboard:read", permission.Title, "expected Title to match")
		require.Equal(t, "Read access to the dashboard", permission.Description, "expected Description to match")
		require.Equal(t, created, permission.Created, "expected Created to match")
		require.Equal(t, modified, permission.Modified, "expected Modified to match")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		role := &Permission{}
		err := role.Scan(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")

	})
}

func TestRolePermissions(t *testing.T) {
	role := &Role{}

	_, err := role.Permissions()
	require.ErrorIs(t, err, errors.ErrMissingAssociation, "expected error when accessing permissions without association")

	perms := []*Permission{
		{
			ID:          int64(1),
			Title:       "dashboard:read",
			Description: "Read access to the dashboard",
			Created:     created,
			Modified:    modified,
		},
		{
			ID:          int64(2),
			Title:       "dashboard:edit",
			Description: "Edit access to the dashboard",
			Created:     created,
			Modified:    modified,
		},
		{
			ID:          int64(3),
			Title:       "dashboard:delete",
			Description: "Can delete dashboard elements",
			Created:     created,
			Modified:    modified,
		},
	}

	role.SetPermissions(perms)
	permissions, err := role.Permissions()
	require.NoError(t, err, "expected no error when accessing permissions after setting them")
	require.Equal(t, perms, permissions, "expected permissions to match set permissions")
}
