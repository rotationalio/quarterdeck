package db_test

import (
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
)

//=============================================================================
// Permission Store Tests
//=============================================================================

// TestPermissionList verifies all fixture permissions are returned.
func (s *storeSuite) TestPermissionList() {
	cursor, err := s.store.ListPermissions(s.Context(), nil)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(cursor.Close()) }()
	permissions, err := cursor.List()
	s.Require().NoError(err)
	s.Len(permissions, 10)
}

// TestCreatePermission verifies validation and successful permission creation.
func (s *storeSuite) TestCreatePermission() {
	s.Run("NoIDOnCreate", func() {
		perm := &models.Permission{
			ID:          128,
			Title:       "Test Permission",
			Description: "This is a test permission",
		}

		_, err := s.store.CreatePermission(s.Context(), perm)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("HappyPath", func() {
		perm := &models.Permission{
			Title:       "Test Permission",
			Description: "This is a test permission",
		}

		require := s.Require()
		count := s.count("permissions")

		created, err := s.store.CreatePermission(s.Context(), perm)
		require.NoError(err)
		require.NotZero(created.ID)
		require.Equal(perm.Title, created.Title)
		require.Equal(perm.Description, created.Description)
		require.WithinDuration(time.Now(), created.Created, 1*time.Second)
		require.WithinDuration(time.Now(), created.Modified, 1*time.Second)
		require.Equal(count+1, s.count("permissions"))
	})

	s.Run("UniqueTitle", func() {
		perm := &models.Permission{
			Title: "content:view",
		}
		_, err := s.store.CreatePermission(s.Context(), perm)
		s.Require().ErrorIs(err, errors.ErrAlreadyExists)
	})
}

// TestRetrievePermission verifies lookup by ID, title, and not-found cases.
func (s *storeSuite) TestRetrievePermission() {
	expected := &models.Permission{
		ID:          2,
		Title:       "content:view",
		Description: "Permission to view content",
		Created:     time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
		Modified:    time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
	}

	s.Run("RetrieveByID", func() {
		perm, err := s.store.RetrievePermission(s.Context(), int64(2))
		s.Require().NoError(err)
		s.Equal(expected, perm)
	})

	s.Run("RetrieveByTitle", func() {
		perm, err := s.store.RetrievePermissionByTitle(s.Context(), "content:view")
		s.Require().NoError(err)
		s.Equal(expected, perm)
	})

	s.Run("NotFound", func() {
		perm, err := s.store.RetrievePermissionByTitle(s.Context(), "non-existent:permission")
		s.Require().ErrorIs(err, errors.ErrNotFound)
		s.Nil(perm)
	})
}

// TestUpdatePermission verifies field updates preserve created timestamp and reject missing rows.
func (s *storeSuite) TestUpdatePermission() {
	require := s.Require()
	perm, err := s.store.RetrievePermissionByTitle(s.Context(), "content:view")
	require.NoError(err)
	require.NotNil(perm)

	s.Run("HappyPath", func() {
		perm.Title = "content:edit"
		perm.Description = "Updated description for content edit permission"
		perm.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)
		perm.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)

		err = s.store.UpdatePermission(s.Context(), perm)
		require.NoError(err)

		cmpt, err := s.store.RetrievePermission(s.Context(), perm.ID)
		require.NoError(err)
		require.Equal(perm.ID, cmpt.ID)
		require.Equal(perm.Title, cmpt.Title)
		require.Equal(perm.Description, cmpt.Description)
		require.NotEqual(perm.Created, cmpt.Created)
		require.WithinDuration(time.Now(), cmpt.Modified, 1*time.Second)
	})

	s.Run("NotFound", func() {
		perm.ID = 999999
		err = s.store.UpdatePermission(s.Context(), perm)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestDeletePermission verifies cascade to role_permissions and idempotent not-found.
func (s *storeSuite) TestDeletePermission() {
	require := s.Require()
	permCount := s.count("permissions")
	rolePermCount := s.count("role_permissions")

	permID := int64(2)

	err := s.store.DeletePermission(s.Context(), permID)
	require.NoError(err)

	require.Equal(permCount-1, s.count("permissions"))
	require.Equal(rolePermCount-3, s.count("role_permissions"))

	err = s.store.DeletePermission(s.Context(), permID)
	require.ErrorIs(err, errors.ErrNotFound)
}
