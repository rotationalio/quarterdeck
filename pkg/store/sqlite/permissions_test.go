package sqlite_test

import (
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

func (s *storeTestSuite) TestRoleList() {
	require := s.Require()
	out, err := s.db.ListRoles(s.Context(), nil)
	require.NoError(err, "listing roles should not error")
	require.NotNil(out.Roles, "should return a list of roles")
	require.NotNil(out.Page, "should return a page object")
	require.Len(out.Roles, 4, "expected 4 fixture roles")
}

func (s *storeTestSuite) TestCreateRole() {

	s.Run("NoIDOnCreate", func() {
		require := s.Require()
		role := &models.Role{
			ID:          128,
			Title:       "Test Role",
			Description: "This is a test role",
			IsDefault:   false,
		}

		err := s.db.CreateRole(s.Context(), role)
		require.ErrorIs(err, errors.ErrNoIDOnCreate, "creating a role with an ID set")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping create read-only error test in read-write mode")
		}

		role := &models.Role{
			Title:       "Test Role",
			Description: "This is a test role",
		}

		err := s.db.CreateRole(s.Context(), role)
		s.Require().ErrorIs(err, errors.ErrReadOnly, "creating a role in read-only mode should error")
	})

	s.Run("CreateEmptyPermissions", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		roleCount := s.Count("roles")
		rolePermsCount := s.Count("role_permissions")

		require := s.Require()
		role := &models.Role{
			Title:       "testing",
			Description: "This is a test role",
			IsDefault:   false,
		}

		err := s.db.CreateRole(s.Context(), role)
		require.NoError(err, "creating a role should not error")
		require.NotZero(role.ID, "role ID should be set after creation")
		require.WithinDuration(time.Now(), role.Created, 1*time.Second)
		require.WithinDuration(time.Now(), role.Modified, 1*time.Second)

		require.Equal(roleCount+1, s.Count("roles"), "should have one more role after creation")
		require.Equal(rolePermsCount, s.Count("role_permissions"), "should not have created any role permissions")
	})

	s.Run("CreateWithPermissions", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		roleCount := s.Count("roles")
		rolePermsCount := s.Count("role_permissions")

		role := &models.Role{
			Title:       "testing:view",
			Description: "This is a test role",
			IsDefault:   false,
		}

		role.SetPermissions([]*models.Permission{
			{ID: 2},
			{Title: "users:view"},
		})

		require := s.Require()
		err := s.db.CreateRole(s.Context(), role)
		require.NoError(err, "creating a role should not error")
		require.NotZero(role.ID, "role ID should be set after creation")

		require.Equal(roleCount+1, s.Count("roles"), "should have one more role after creation")
		require.Equal(rolePermsCount+2, s.Count("role_permissions"), "should not have created any role permissions")
	})

	s.Run("CreateWithBadPermission", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		role := &models.Role{
			Title:       "testing:error",
			Description: "This is a test role",
			IsDefault:   false,
		}

		role.SetPermissions([]*models.Permission{
			{ID: 4992912, Title: "foo:bar"},
		})

		require := s.Require()
		err := s.db.CreateRole(s.Context(), role)
		require.EqualError(err, "invalid permission \"foo:bar\" (ID: 4992912): role not created: record not found")
	})
}

func (s *storeTestSuite) TestRetrieveRole() {
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
		role, err := s.db.RetrieveRole(s.Context(), expected.ID)
		require.NoError(err, "retrieving role by ID should not error")
		require.Equal(expected.ID, role.ID, "retrieved role should match expected")
		require.Equal(expected.Title, role.Title, "retrieved role should match expected")
		require.Equal(expected.Description, role.Description, "retrieved role should match expected")
		require.Equal(expected.IsDefault, role.IsDefault, "retrieved role should match expected")
		require.Equal(expected.Created, role.Created, "retrieved role should match expected")
		require.Equal(expected.Modified, role.Modified, "retrieved role should match expected")

		permissions, err := role.Permissions()
		require.NoError(err, "retrieving permissions for role should not error")
		require.Len(permissions, 2, "role should have 2 permissions")
		require.Equal("content:view", permissions[0].Title, "first permission should be content:view")
		require.Equal("users:view", permissions[1].Title, "second permission should be users:view")
	})

	s.Run("RetrieveByTitle", func() {
		require := s.Require()
		role, err := s.db.RetrieveRole(s.Context(), expected.Title)
		require.NoError(err, "retrieving role by title should not error")
		require.Equal(expected.ID, role.ID, "retrieved role should match expected")
		require.Equal(expected.Title, role.Title, "retrieved role should match expected")
		require.Equal(expected.Description, role.Description, "retrieved role should match expected")
		require.Equal(expected.IsDefault, role.IsDefault, "retrieved role should match expected")
		require.Equal(expected.Created, role.Created, "retrieved role should match expected")
		require.Equal(expected.Modified, role.Modified, "retrieved role should match expected")

		permissions, err := role.Permissions()
		require.NoError(err, "retrieving permissions for role should not error")
		require.Len(permissions, 2, "role should have 2 permissions")
		require.Equal("content:view", permissions[0].Title, "first permission should be content:view")
		require.Equal("users:view", permissions[1].Title, "second permission should be users:view")
	})

	s.Run("RetrieveByInvalidType", func() {
		require := s.Require()
		role, err := s.db.RetrieveRole(s.Context(), true)
		require.EqualError(err, "invalid type bool for titleOrName")
		require.Nil(role, "retrieved role should be nil")
	})

	s.Run("NotFound", func() {
		require := s.Require()
		role, err := s.db.RetrieveRole(s.Context(), "foo")
		require.ErrorIs(err, errors.ErrNotFound, "retrieving non-existent role should return not found error")
		require.Nil(role, "retrieved role should be nil")
	})

}

func (s *storeTestSuite) TestUpdateRole() {
	if s.ReadOnly() {
		s.T().Skip("skipping update test in read-only mode")
	}

	require := s.Require()
	role, err := s.db.RetrieveRole(s.Context(), "viewer")
	require.NoError(err, "retrieving role for update should not error")
	require.NotNil(role, "retrieved role should not be nil")

	s.Run("HappyPath", func() {
		role.Title = "observer"
		role.Description = "Updated description for observer role"
		role.IsDefault = true
		role.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)  // Should not change
		role.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC) // Should be set to now
		role.SetPermissions(nil)

		err = s.db.UpdateRole(s.Context(), role)
		require.NoError(err, "updating role should not error")

		// Fetch updated role for comparison
		cmpt, err := s.db.RetrieveRole(s.Context(), role.ID)
		require.NoError(err, "retrieving updated role should not error")
		require.Equal(role.ID, cmpt.ID, "updated role ID should match")
		require.Equal(role.Title, cmpt.Title, "updated role title should match")
		require.Equal(role.Description, cmpt.Description, "updated role description should match")
		require.Equal(role.IsDefault, cmpt.IsDefault, "updated role isDefault should match")
		require.NotEqual(role.Created, cmpt.Created, "created should not be updated")
		require.WithinDuration(time.Now(), cmpt.Modified, 1*time.Second, "updated role modified time should be close to now")

		permissions, err := cmpt.Permissions()
		require.NoError(err, "retrieving permissions for updated role should not error")
		require.Len(permissions, 2, "permissions should not have changed")
	})

	s.Run("AddPermissionToRole", func() {
		require := s.Require()
		err := s.db.AddPermissionToRole(s.Context(), role.ID, "keys:view")
		require.NoError(err, "adding permission to role should not error")

		cmpt, err := s.db.RetrieveRole(s.Context(), role.ID)
		require.NoError(err, "retrieving updated role should not error")

		permissions, err := cmpt.Permissions()
		require.NoError(err, "retrieving permissions for updated role should not error")
		require.Len(permissions, 3, "role should now have 3 permissions")
		require.Equal("keys:view", permissions[2].Title, "new permission should be keys:view")
	})

	s.Run("RemovePermissionFromRole", func() {
		require := s.Require()
		cmpt, err := s.db.RetrieveRole(s.Context(), role.ID)
		require.NoError(err, "retrieving role for permission removal should not error")

		permissions, err := cmpt.Permissions()
		require.NoError(err, "retrieving permissions for role should not error")
		require.NotEmpty(permissions, "role should have permissions before removal")

		// Remove the first permission from the list
		target := permissions[0]

		err = s.db.RemovePermissionFromRole(s.Context(), role.ID, target.ID)
		require.NoError(err, "removing permission from role should not error")

		cmpt, err = s.db.RetrieveRole(s.Context(), role.ID)
		require.NoError(err, "retrieving updated role after permission removal should not error")

		permissions, err = cmpt.Permissions()
		require.NoError(err, "retrieving permissions for updated role should not error")

		for _, perm := range permissions {
			require.NotEqual(target.ID, perm.ID, "removed permission should not be present in updated role permissions")
		}
	})

	s.Run("NotFound", func() {
		role.ID = 999999 // Non-existent role ID
		err = s.db.UpdateRole(s.Context(), role)
		require.ErrorIs(err, errors.ErrNotFound, "updating non-existent role should return not found error")
	})
}

func (s *storeTestSuite) TestDeleteRole() {
	if s.ReadOnly() {
		s.T().Skip("skipping delete test in read-only mode")
	}

	require := s.Require()
	roleCount := s.Count("roles")
	rolePermsCount := s.Count("role_permissions")

	roleID := int64(3)

	err := s.db.DeleteRole(s.Context(), roleID)
	require.NoError(err, "deleting role should not error")

	require.Equal(roleCount-1, s.Count("roles"), "role count should decrease by 1")
	require.Equal(rolePermsCount-2, s.Count("role_permissions"), "role permissions count should change")

	err = s.db.DeleteRole(s.Context(), roleID)
	require.ErrorIs(err, errors.ErrNotFound, "deleting non-existent role should return not found error")
}

func (s *storeTestSuite) TestPermissionList() {
	require := s.Require()
	out, err := s.db.ListPermissions(s.Context(), nil)
	require.NoError(err, "listing permissions should not error")
	require.NotNil(out.Permissions, "should return a list of permissions")
	require.NotNil(out.Page, "should return a page object")
	require.Len(out.Permissions, 10, "expected 10 fixture permissions")
}

func (s *storeTestSuite) TestCreatePermission() {
	s.Run("NoIDOnCreate", func() {
		require := s.Require()
		perm := &models.Permission{
			ID:          128,
			Title:       "Test Permission",
			Description: "This is a test permission",
		}

		err := s.db.CreatePermission(s.Context(), perm)
		require.ErrorIs(err, errors.ErrNoIDOnCreate, "creating a permission with an ID set")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping create read-only error test in read-write mode")
		}

		perm := &models.Permission{
			Title:       "Test Permission",
			Description: "This is a test permission",
		}

		err := s.db.CreatePermission(s.Context(), perm)
		s.Require().ErrorIs(err, errors.ErrReadOnly, "creating a permission in read-only mode should error")
	})

	s.Run("HappyPath", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		perm := &models.Permission{
			Title:       "Test Permission",
			Description: "This is a test permission",
		}

		require := s.Require()
		count := s.Count("permissions")

		err := s.db.CreatePermission(s.Context(), perm)
		require.NoError(err, "creating a permission should not error")
		require.NotZero(perm.ID, "permission ID should be set after creation")
		require.WithinDuration(time.Now(), perm.Created, 1*time.Second)
		require.WithinDuration(time.Now(), perm.Modified, 1*time.Second)
		require.Equal(count+1, s.Count("permissions"), "should have one more permission after creation")
	})

	s.Run("UniqueTitle", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		perm := &models.Permission{
			Title: "content:view",
		}
		require := s.Require()
		err := s.db.CreatePermission(s.Context(), perm)
		require.ErrorIs(err, errors.ErrAlreadyExists)
	})

}

func (s *storeTestSuite) TestRetrievePermission() {
	require := s.Require()
	expected := &models.Permission{
		ID:          2,
		Title:       "content:view",
		Description: "Permission to view content",
		Created:     time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
		Modified:    time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
	}

	s.Run("RetrieveByID", func() {
		perm, err := s.db.RetrievePermission(s.Context(), int64(2))
		require.NoError(err, "retrieving permission by ID should not error")
		require.Equal(expected, perm, "retrieved permission should match expected")
	})

	s.Run("RetrieveByIntID", func() {
		perm, err := s.db.RetrievePermission(s.Context(), int(2))
		require.NoError(err, "retrieving permission by int instead of int64 should not error")
		require.Equal(expected, perm, "retrieved permission should match expected")
	})

	s.Run("RetrieveByTitle", func() {
		perm, err := s.db.RetrievePermission(s.Context(), "content:view")
		require.NoError(err, "retrieving permission by title should not error")
		require.Equal(expected, perm, "retrieved permission should match expected")
	})

	s.Run("RetrieveByModelID", func() {
		perm, err := s.db.RetrievePermission(s.Context(), &models.Permission{ID: 2})
		require.NoError(err, "retrieving permission by model ID should not error")
		require.Equal(expected, perm, "retrieved permission should match expected")
	})

	s.Run("RetrieveByModelTitle", func() {
		perm, err := s.db.RetrievePermission(s.Context(), &models.Permission{Title: "content:view"})
		require.NoError(err, "retrieving permission by model title should not error")
		require.Equal(expected, perm, "retrieved permission should match expected")
	})

	s.Run("RetrieveByModelMissingID", func() {
		badIDs := []any{
			int64(0),
			int(0),
			"",
			&models.Permission{ID: 0, Title: ""},
		}

		for _, bid := range badIDs {
			perm, err := s.db.RetrievePermission(s.Context(), bid)
			require.ErrorIs(err, errors.ErrMissingID, "retrieving permission with bad ID should return missing ID error")
			require.Nil(perm, "retrieved permission should be nil for bad ID")
		}
	})

	s.Run("RetrieveByInvalidType", func() {
		perm, err := s.db.RetrievePermission(s.Context(), true)
		require.EqualError(err, "invalid type bool for titleOrID")
		require.Nil(perm, "retrieved permission should be nil")
	})

	s.Run("NotFound", func() {
		perm, err := s.db.RetrievePermission(s.Context(), "non-existent:permission")
		require.ErrorIs(err, errors.ErrNotFound, "retrieving non-existent permission should return not found error")
		require.Nil(perm, "retrieved permission should be nil")
	})
}

func (s *storeTestSuite) TestUpdatePermission() {
	if s.ReadOnly() {
		s.T().Skip("skipping update test in read-only mode")
	}

	require := s.Require()
	perm, err := s.db.RetrievePermission(s.Context(), "content:view")
	require.NoError(err, "retrieving permission for update should not error")
	require.NotNil(perm, "retrieved permission should not be nil")

	s.Run("HappyPath", func() {
		perm.Title = "content:edit"
		perm.Description = "Updated description for content edit permission"
		perm.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)  // Should not change
		perm.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC) // Should be set to now

		err = s.db.UpdatePermission(s.Context(), perm)
		require.NoError(err, "updating permission should not error")

		cmpt, err := s.db.RetrievePermission(s.Context(), perm.ID)
		require.NoError(err, "retrieving updated permission should not error")
		require.Equal(perm.ID, cmpt.ID, "updated permission ID should match")
		require.Equal(perm.Title, cmpt.Title, "updated permission title should match")
		require.Equal(perm.Description, cmpt.Description, "updated permission description should match")
		require.NotEqual(perm.Created, cmpt.Created, "created should not be updated")
		require.WithinDuration(time.Now(), cmpt.Modified, 1*time.Second, "updated permission modified time should be close to now")
	})

	s.Run("NotFound", func() {
		perm.ID = 999999 // Non-existent permission ID
		err = s.db.UpdatePermission(s.Context(), perm)
		require.ErrorIs(err, errors.ErrNotFound, "updating non-existent permission should return not found error")
	})
}

func (s *storeTestSuite) TestDeletePermission() {
	if s.ReadOnly() {
		s.T().Skip("skipping delete test in read-only mode")
	}

	require := s.Require()
	permCount := s.Count("permissions")
	rolePermCount := s.Count("role_permissions")

	permID := int64(2)

	err := s.db.DeletePermission(s.Context(), permID)
	require.NoError(err, "deleting permission should not error")

	require.Equal(permCount-1, s.Count("permissions"), "permission count should decrease by 1")
	require.Equal(rolePermCount-3, s.Count("role_permissions"), "role permission count should decrease by 1")

	err = s.db.DeletePermission(s.Context(), permID)
	require.ErrorIs(err, errors.ErrNotFound, "deleting non-existent permission should return not found error")
}
