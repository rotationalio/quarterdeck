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
			s.T().Skip("skipping create read-only error test in read-write mode")
		}

		user := &models.User{
			Name:     sql.NullString{String: "Test User", Valid: true},
			Email:    "test@example.com",
			Password: "password",
		}
		err := s.db.CreateUser(s.Context(), user)
		s.Require().ErrorIs(err, errors.ErrReadOnly, "should not allow creating a user in read-only mode")
	})

	s.Run("CreateDefaultRole", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		require := s.Require()
		userCount := s.Count("users")
		userRolesCount := s.Count("user_roles")

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
		require.Equal(userCount+1, s.Count("users"), "should increase the user count by one")
		require.Equal(userRolesCount+1, s.Count("user_roles"), "should increase the user roles count by one")
	})

	s.Run("CreateMultipleRoles", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		require := s.Require()
		userCount := s.Count("users")
		userRolesCount := s.Count("user_roles")

		user := &models.User{
			Name:     sql.NullString{String: "Fran Shepherd", Valid: true},
			Email:    "fran@example.com",
			Password: "billygoatcurses",
		}

		user.SetRoles([]*models.Role{
			{ID: int64(2), Title: "Editor", IsDefault: false},
			{ID: int64(4), Title: "Keyholder", IsDefault: false},
		})

		err := s.db.CreateUser(s.Context(), user)
		require.NoError(err, "should be able to create a user")
		require.NotNil(user.ID, "should set the user ID on create")
		require.False(user.Created.IsZero(), "should set the user created time on create")
		require.False(user.Modified.IsZero(), "should set the user modified time on create")
		require.Equal(userCount+1, s.Count("users"), "should increase the user count by one")
		require.Equal(userRolesCount+2, s.Count("user_roles"), "should increase the user roles count by one")
	})
}

func (s *storeTestSuite) TestRetrieveUser() {
	s.Run("ByEmail", func() {
		require := s.Require()
		user, err := s.db.RetrieveUser(s.Context(), "editor@example.com")
		require.NoError(err, "should be able to retrieve user by email")
		require.NotNil(user, "should return a user")

		require.Equal("01JQNPQ1CHG36SV7NRQKTZB20R", user.ID.String(), "should return the correct user ID")
		require.Equal("Editor User", user.Name.String, "should return the correct user name")
		require.Equal("editor@example.com", user.Email, "should return the correct user email")
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$oPREW7ztC12IG7EVldbneA==$K/4cNUUt661D30ufLmTTN/bZD0WSig/FrbqOmkOoX9I=", user.Password, "should return the correct user password")
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.LastLogin.Time, "should return the correct user last login time")
		require.Equal(time.Date(2025, time.March, 31, 8, 57, 27, 0, time.UTC), user.Created, "should return the correct user created time")
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.Modified, "should return the correct user modified time")

		roles, err := user.Roles()
		require.NoError(err, "should be able to retrieve user roles")
		require.Len(roles, 1, "should return one role for the user")
		require.Equal("editor", roles[0].Title, "should return the correct role for the user")

		permissions := user.Permissions()
		require.Len(permissions, 6, "should return six permission for the user")
	})

	s.Run("ByUserID", func() {
		require := s.Require()
		userID := ulid.MustParse("01JQNPQ1CHG36SV7NRQKTZB20R")
		user, err := s.db.RetrieveUser(s.Context(), userID)
		require.NoError(err, "should be able to retrieve user by email")
		require.NotNil(user, "should return a user")

		require.Equal(userID, user.ID, "should return the correct user ID")
		require.Equal("Editor User", user.Name.String, "should return the correct user name")
		require.Equal("editor@example.com", user.Email, "should return the correct user email")
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$oPREW7ztC12IG7EVldbneA==$K/4cNUUt661D30ufLmTTN/bZD0WSig/FrbqOmkOoX9I=", user.Password, "should return the correct user password")
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.LastLogin.Time, "should return the correct user last login time")
		require.Equal(time.Date(2025, time.March, 31, 8, 57, 27, 0, time.UTC), user.Created, "should return the correct user created time")
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.Modified, "should return the correct user modified time")

		roles, err := user.Roles()
		require.NoError(err, "should be able to retrieve user roles")
		require.Len(roles, 1, "should return one role for the user")
		require.Equal("editor", roles[0].Title, "should return the correct role for the user")

		permissions := user.Permissions()
		require.Len(permissions, 6, "should return six permission for the user")
	})

	s.Run("NotFound", func() {
		require := s.Require()
		userID := ulid.Make()
		user, err := s.db.RetrieveUser(s.Context(), userID)
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error for non-existent user")
		require.Nil(user, "should not return a user for non-existent user ID")
	})

	s.Run("IDTypeError", func() {
		require := s.Require()
		user, err := s.db.RetrieveUser(s.Context(), 42)
		require.EqualError(err, "invalid type int for emailOrUserID")
		require.Nil(user, "should not return a user for non-existent user ID")
	})
}

func (s *storeTestSuite) TestUpdateUser() {
	if s.ReadOnly() {
		s.T().Skip("skipping update test in read-only mode")
	}

	require := s.Require()
	userID := ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A")
	user, err := s.db.RetrieveUser(s.Context(), userID)
	require.NoError(err, "should be able to retrieve user by email")
	require.NotNil(user, "should return a user")

	s.Run("HappyPath", func() {
		user.Name = sql.NullString{String: "Gary Franklin Redfield", Valid: true}
		user.Email = "gfredfield@example.com"
		user.Password = ""
		user.LastLogin = sql.NullTime{Valid: false}
		user.EmailVerified = true                                                  // Should not change
		user.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)  // Should not change
		user.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC) // Should be set to now

		err = s.db.UpdateUser(s.Context(), user)
		require.NoError(err, "should be able to update user")

		// Fetch updated user for comparison
		cmpt, err := s.db.RetrieveUser(s.Context(), user.ID)
		require.NoError(err, "should be able to retrieve updated user")

		require.Equal(user.ID, cmpt.ID, "should keep the same user ID")
		require.Equal(user.Name, cmpt.Name, "should update the user name")
		require.Equal(user.Email, cmpt.Email, "should update the user email")
		require.NotEqual(user.Password, cmpt.Password, "should not change/update the user password")
		require.NotEqual(user.LastLogin, cmpt.LastLogin, "should not clear the user last login time")
		require.NotEqual(user.EmailVerified, cmpt.EmailVerified, "should not change the user email verified status")
		require.NotEqual(user.Created, cmpt.Created, "should not change the user created time")
		require.NotEqual(user.Modified, cmpt.Modified, "should update the user modified time to now, not what was set")
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("UpdatePassword", func() {
		password := "$argon2id$v=19$m=65536,t=1,p=2$DT/LSMZjHhVlprmPaBSCcg==$UKT1g5gqWvKhiBC8gywVU6zepCEew0x3IW9vTWnlVlg="
		err := s.db.UpdatePassword(s.Context(), userID, password)
		require.NoError(err, "should be able to update user password")

		// Fetch updated user for comparison
		cmpt, err := s.db.RetrieveUser(s.Context(), userID)
		require.NoError(err, "should be able to retrieve updated user after password change")
		require.Equal(password, cmpt.Password, "should update the user password")
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("UpdateLastLogin", func() {
		lastLogin := time.Now().UTC()
		err = s.db.UpdateLastLogin(s.Context(), userID, lastLogin)
		require.NoError(err, "should be able to update user last login time")

		// Fetch updated user for comparison
		cmpt, err := s.db.RetrieveUser(s.Context(), userID)
		require.NoError(err, "should be able to retrieve updated user after last login change")
		require.True(cmpt.LastLogin.Valid, "should have a valid last login time")
		require.Equal(lastLogin, cmpt.LastLogin.Time, "should update the user last login time")
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("VerifyEmail", func() {
		err = s.db.VerifyEmail(s.Context(), userID)
		require.NoError(err, "should be able to verify user email")

		// Fetch updated user for comparison
		cmpt, err := s.db.RetrieveUser(s.Context(), userID)
		require.NoError(err, "should be able to retrieve updated user after email verification")
		require.True(cmpt.EmailVerified, "should set the user email verified status to true")
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("NotFound", func() {
		user.ID = ulid.Make() // Set a new ID that does not exist
		err = s.db.UpdateUser(s.Context(), user)
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error for non-existent user")
	})
}

func (s *storeTestSuite) TestDeleteUser() {
	if s.ReadOnly() {
		s.T().Skip("skipping delete test in read-only mode")
	}

	require := s.Require()
	userCount := s.Count("users")
	userRolesCount := s.Count("user_roles")

	userID := ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A")

	err := s.db.DeleteUser(s.Context(), userID)
	require.NoError(err, "should be able to delete user")

	require.Equal(userCount-1, s.Count("users"), "should decrease the user count by one")
	require.Equal(userRolesCount-1, s.Count("user_roles"), "should decrease the user roles count by one")

	err = s.db.DeleteUser(s.Context(), userID)
	require.ErrorIs(err, errors.ErrNotFound)
}
