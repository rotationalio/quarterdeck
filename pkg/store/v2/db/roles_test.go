package db_test

import (
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
)

//=============================================================================
// Role Store Tests
//=============================================================================

// TestRoleList verifies all fixture roles are returned.
func (s *storeSuite) TestRoleList() {
	cursor, err := s.store.ListRoles(s.Context(), nil)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(cursor.Close()) }()
	roles, err := cursor.List()
	s.Require().NoError(err)
	s.Len(roles, 4)
}

// TestCreateRole verifies validation, permission assignment, and error cases on create.
func (s *storeSuite) TestCreateRole() {
	s.Run("NoIDOnCreate", func() {
		role := &models.Role{
			ID:          128,
			Title:       "Test Role",
			Description: "This is a test role",
			IsDefault:   false,
		}

		_, err := s.store.CreateRole(s.Context(), role)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("CreateEmptyPermissions", func() {
		roleCount := s.count("roles")
		rolePermsCount := s.count("role_permissions")

		require := s.Require()
		role := &models.Role{
			Title:       "testing",
			Description: "This is a test role",
			IsDefault:   false,
		}

		created, err := s.store.CreateRole(s.Context(), role)
		require.NoError(err)
		require.NotZero(created.ID)
		require.Equal(role.Title, created.Title)
		require.Equal(role.Description, created.Description)
		require.Equal(role.IsDefault, created.IsDefault)
		require.WithinDuration(time.Now(), created.Created, 1*time.Second)
		require.WithinDuration(time.Now(), created.Modified, 1*time.Second)
		require.Equal(roleCount+1, s.count("roles"))
		require.Equal(rolePermsCount, s.count("role_permissions"))
	})

	s.Run("CreateWithPermissions", func() {
		roleCount := s.count("roles")
		rolePermsCount := s.count("role_permissions")

		role := &models.Role{
			Title:       "testing:view",
			Description: "This is a test role",
			IsDefault:   false,
		}
		role.Permissions = []models.Permission{
			{ID: 2},
			{Title: "users:view"},
		}

		require := s.Require()
		created, err := s.store.CreateRole(s.Context(), role)
		require.NoError(err)
		require.NotZero(created.ID)
		require.Equal(roleCount+1, s.count("roles"))
		require.Equal(rolePermsCount+2, s.count("role_permissions"))
	})

	s.Run("CreateWithBadPermission", func() {
		role := &models.Role{
			Title:       "testing:error",
			Description: "This is a test role",
			IsDefault:   false,
		}
		role.Permissions = []models.Permission{
			{ID: 4992912, Title: "foo:bar"},
		}

		_, err := s.store.CreateRole(s.Context(), role)
		s.Require().Error(err)
	})
}

// TestRetrieveRole verifies lookup by ID/title loads attached permissions.
func (s *storeSuite) TestRetrieveRole() {
	expected := &models.Role{
		ID:          3,
		Title:       "viewer",
		Description: "Viewer role with permissions to view content only",
		IsDefault:   false,
		Created:     time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
		Modified:    time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
	}

	s.Run("RetrieveByID", func() {
		require := s.Require()
		role, err := s.store.RetrieveRole(s.Context(), expected.ID)
		require.NoError(err)
		require.Equal(expected.ID, role.ID)
		require.Equal(expected.Title, role.Title)
		require.Equal(expected.Description, role.Description)
		require.Equal(expected.IsDefault, role.IsDefault)
		require.Equal(expected.Created, role.Created)
		require.Equal(expected.Modified, role.Modified)

		permissions := role.Permissions
		require.Len(permissions, 2)
		require.Equal("content:view", permissions[0].Title)
		require.Equal("users:view", permissions[1].Title)
	})

	s.Run("RetrieveByTitle", func() {
		require := s.Require()
		role, err := s.store.RetrieveRoleByTitle(s.Context(), expected.Title)
		require.NoError(err)
		require.Equal(expected.ID, role.ID)
		require.Equal(expected.Title, role.Title)
		require.Equal(expected.Description, role.Description)
		require.Equal(expected.IsDefault, role.IsDefault)
		require.Equal(expected.Created, role.Created)
		require.Equal(expected.Modified, role.Modified)

		permissions := role.Permissions
		require.Len(permissions, 2)
		require.Equal("content:view", permissions[0].Title)
		require.Equal("users:view", permissions[1].Title)
	})

	s.Run("NotFound", func() {
		require := s.Require()
		role, err := s.store.RetrieveRoleByTitle(s.Context(), "foo")
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(role)
	})
}

// TestUpdateRole verifies role updates, permission junction changes, and not-found handling.
func (s *storeSuite) TestUpdateRole() {
	require := s.Require()
	role, err := s.store.RetrieveRoleByTitle(s.Context(), "viewer")
	require.NoError(err)
	require.NotNil(role)

	s.Run("HappyPath", func() {
		// Setup: mutate fields that should and should not persist.
		role.Title = "observer"
		role.Description = "Updated description for observer role"
		role.IsDefault = true
		role.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)
		role.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)
		role.Permissions = nil

		// Action: update role metadata only.
		err = s.store.UpdateRole(s.Context(), role)
		require.NoError(err)

		// Assert: writable fields changed; created unchanged; permissions untouched.
		cmpt, err := s.store.RetrieveRole(s.Context(), role.ID)
		require.NoError(err)
		require.Equal(role.ID, cmpt.ID)
		require.Equal(role.Title, cmpt.Title)
		require.Equal(role.Description, cmpt.Description)
		require.Equal(role.IsDefault, cmpt.IsDefault)
		require.NotEqual(role.Created, cmpt.Created)
		require.WithinDuration(time.Now(), cmpt.Modified, 1*time.Second)

		permissions := cmpt.Permissions
		require.Len(permissions, 2)
	})

	s.Run("AddPermissionToRole", func() {
		s.Run("ByTitle", func() {
			err := s.store.AddPermissionToRoleByTitle(s.Context(), role.ID, "keys:view")
			require.NoError(err)

			cmpt, err := s.store.RetrieveRole(s.Context(), role.ID)
			require.NoError(err)

			permissions := cmpt.Permissions
			require.Len(permissions, 3)
			require.Equal("keys:view", permissions[2].Title)
		})

		s.Run("ByID", func() {
			// Reset: viewer role has two fixture permissions and should not yet include keys:view (ID 10).
			s.resetStore()
			viewer, err := s.store.RetrieveRoleByTitle(s.Context(), "viewer")
			require.NoError(err)

			err = s.store.AddPermissionToRole(s.Context(), viewer.ID, 10)
			require.NoError(err)

			cmpt, err := s.store.RetrieveRole(s.Context(), viewer.ID)
			require.NoError(err)
			require.Len(cmpt.Permissions, 3)
			require.Equal("keys:view", cmpt.Permissions[2].Title)
		})
	})

	s.Run("RemovePermissionFromRole", func() {
		cmpt, err := s.store.RetrieveRole(s.Context(), role.ID)
		require.NoError(err)

		permissions := cmpt.Permissions
		require.NotEmpty(permissions)

		target := permissions[0]

		err = s.store.RemovePermissionFromRole(s.Context(), role.ID, target.ID)
		require.NoError(err)

		cmpt, err = s.store.RetrieveRole(s.Context(), role.ID)
		require.NoError(err)

		permissions = cmpt.Permissions
		for _, perm := range permissions {
			require.NotEqual(target.ID, perm.ID)
		}
	})

	s.Run("NotFound", func() {
		role.ID = 999999
		err = s.store.UpdateRole(s.Context(), role)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestDeleteRole verifies cascade to role_permissions and idempotent not-found.
func (s *storeSuite) TestDeleteRole() {
	require := s.Require()
	roleCount := s.count("roles")
	rolePermsCount := s.count("role_permissions")

	roleID := int64(3)

	err := s.store.DeleteRole(s.Context(), roleID)
	require.NoError(err)

	require.Equal(roleCount-1, s.count("roles"))
	require.Equal(rolePermsCount-2, s.count("role_permissions"))

	err = s.store.DeleteRole(s.Context(), roleID)
	require.ErrorIs(err, errors.ErrNotFound)
}
