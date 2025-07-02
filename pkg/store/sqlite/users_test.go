package sqlite_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestUserList() {
	require := s.Require()
	out, err := s.db.ListUsers(s.Context(), nil)
	require.NoError(err, "should be able to list users")
	require.NotNil(out, "should return a user list")
	require.Len(out.Users, 5, "should return the four fixture users")
}

func (s *storeTestSuite) TestUserListWithRole() {
	require := s.Require()
	out, err := s.db.ListUsers(s.Context(), &models.UserPage{Role: "admin"})
	require.NoError(err, "should be able to list users with role")
	require.NotNil(out, "should return a user list")
	require.Len(out.Users, 2, "should return the two fixture users with the admin role")
}

func (s *storeTestSuite) TestCreateUser() {
	s.Run("NoIDOnCreate", func() {
		user := &models.User{
			Model: models.Model{
				ID:       ulid.Make(),
				Created:  time.Now(),
				Modified: time.Now(),
			},
			Name: sql.NullString{String: "Test User", Valid: true},
		}

		err := s.db.CreateUser(s.Context(), user)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate, "should not allow creating a user with an ID set")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping read-only test in read-write mode")
		}

		user := &models.User{
			Name:     sql.NullString{String: "Test User", Valid: true},
			Email:    "test@example.com",
			Password: "password",
		}
		err := s.db.CreateUser(s.Context(), user)
		s.Require().ErrorIs(err, errors.ErrReadOnly, "should not allow creating a user in read-only mode")
	})

	s.Run("Create", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping write test in read-only mode")
		}

		require := s.Require()

		user := &models.User{
			Name:     sql.NullString{String: "Test User", Valid: true},
			Email:    "test@example.com",
			Password: "password",
		}
		err := s.db.CreateUser(s.Context(), user)
		require.NoError(err, "should be able to create a user")
		require.NotNil(user.ID, "should set the user ID on create")
		require.False(user.Created.IsZero(), "should set the user created time on create")
		require.False(user.Modified.IsZero(), "should set the user modified time on create")
	})
}
