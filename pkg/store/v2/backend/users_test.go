package backend_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// User Store Tests
//=============================================================================

// TestUserList verifies all fixture users are returned.
func (s *storeSuite) TestUserList() {
	cursor, err := s.store.ListUsers(s.Context(), nil)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(cursor.Close()) }()
	users, err := cursor.List()
	s.Require().NoError(err)
	s.Len(users, 5)
}

// TestUserListWithRoleFilter verifies role-based filtering on user list.
func (s *storeSuite) TestUserListWithRoleFilter() {
	filter := &tidal.Clause{
		SQL: "WHERE id IN (SELECT ur.user_id FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE LOWER(r.title) = LOWER(:role))",
		Args: []sql.NamedArg{
			sql.Named("role", "admin"),
		},
	}
	cursor, err := s.store.ListUsers(s.Context(), filter)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(cursor.Close()) }()
	users, err := cursor.List()
	s.Require().NoError(err)
	s.Len(users, 2)
}

// TestCreateUser verifies validation, default role assignment, and duplicate email rejection.
func (s *storeSuite) TestCreateUser() {
	s.Run("NoIDOnCreate", func() {
		user := &models.User{
			Name: sql.NullString{String: "Test User", Valid: true},
		}
		user.ID = ulid.Make()

		_, err := s.store.CreateUser(s.Context(), user)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("CreateDefaultRole", func() {
		require := s.Require()
		userCount := s.count("users")
		userRolesCount := s.count("user_roles")

		user := &models.User{
			Name:     sql.NullString{String: "Test User", Valid: true},
			Email:    "test@example.com",
			Password: "password",
		}
		created, err := s.store.CreateUser(s.Context(), user)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.False(created.Created.IsZero())
		require.False(created.Modified.IsZero())
		require.Equal(userCount+1, s.count("users"))
		require.Equal(userRolesCount+1, s.count("user_roles"))
	})

	s.Run("CreateMultipleRoles", func() {
		require := s.Require()
		userCount := s.count("users")
		userRolesCount := s.count("user_roles")

		user := &models.User{
			Name:     sql.NullString{String: "Fran Shepherd", Valid: true},
			Email:    "fran@example.com",
			Password: "billygoatcurses",
		}
		user.Roles = []models.Role{
			{ID: int64(2), Title: "Editor", IsDefault: false},
			{ID: int64(4), Title: "Keyholder", IsDefault: false},
		}

		created, err := s.store.CreateUser(s.Context(), user)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.False(created.Created.IsZero())
		require.False(created.Modified.IsZero())
		require.Equal(userCount+1, s.count("users"))
		require.Equal(userRolesCount+2, s.count("user_roles"))
	})

	s.Run("CreateRolesByTitle", func() {
		require := s.Require()
		userCount := s.count("users")
		userRolesCount := s.count("user_roles")

		user := &models.User{
			Name:     sql.NullString{String: "Title Roles User", Valid: true},
			Email:    "titleroles@example.com",
			Password: "titleroles-password",
		}
		user.Roles = []models.Role{
			{Title: "editor"},
			{Title: "keyholder"},
		}

		created, err := s.store.CreateUser(s.Context(), user)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.Equal(userCount+1, s.count("users"))
		require.Equal(userRolesCount+2, s.count("user_roles"))

		roles := created.Roles
		require.Len(roles, 2)
		require.Equal("editor", roles[0].Title)
		require.Equal("keyholder", roles[1].Title)
	})

	s.Run("NoDuplicateEmail", func() {
		user := &models.User{
			Name:     sql.NullString{String: "Test User", Valid: true},
			Email:    "gary@example.com",
			Password: "$argon2id$v=19$m=65536,t=1,p=2$nXCe+4HPx0YfO/BMRTtePQ==$vRxaszj/Y4NtfqL7DYDKp3zILXuAnEpzxCtCAc1fdTk=",
		}
		_, err := s.store.CreateUser(s.Context(), user)
		s.Require().ErrorIs(err, errors.ErrAlreadyExists)
	})
}

// TestRetrieveUser verifies lookup by email, ID, and not-found cases.
func (s *storeSuite) TestRetrieveUser() {
	s.Run("ByEmail", func() {
		require := s.Require()
		user, err := s.store.RetrieveUserByEmail(s.Context(), "editor@example.com")
		require.NoError(err)
		require.NotNil(user)

		require.Equal("01JQNPQ1CHG36SV7NRQKTZB20R", user.ID.String())
		require.Equal("Editor User", user.Name.String)
		require.Equal("editor@example.com", user.Email)
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$oPREW7ztC12IG7EVldbneA==$K/4cNUUt661D30ufLmTTN/bZD0WSig/FrbqOmkOoX9I=", user.Password)
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.LastLogin.Time)
		require.Equal(time.Date(2025, time.March, 31, 8, 57, 27, 0, time.UTC), user.Created)
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.Modified)

		roles := user.Roles
		require.Len(roles, 1)
		require.Equal("editor", roles[0].Title)

		require.Len(user.Permissions, 6)
	})

	s.Run("ByUserID", func() {
		require := s.Require()
		userID := ulid.MustParse("01JQNPQ1CHG36SV7NRQKTZB20R")
		user, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)
		require.NotNil(user)

		require.Equal(userID, user.ID)
		require.Equal("Editor User", user.Name.String)
		require.Equal("editor@example.com", user.Email)
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$oPREW7ztC12IG7EVldbneA==$K/4cNUUt661D30ufLmTTN/bZD0WSig/FrbqOmkOoX9I=", user.Password)
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.LastLogin.Time)
		require.Equal(time.Date(2025, time.March, 31, 8, 57, 27, 0, time.UTC), user.Created)
		require.Equal(time.Date(2025, time.April, 29, 15, 2, 51, 0, time.UTC), user.Modified)

		roles := user.Roles
		require.Len(roles, 1)
		require.Equal("editor", roles[0].Title)

		require.Len(user.Permissions, 6)
	})

	s.Run("NotFound", func() {
		require := s.Require()
		userID := ulid.Make()
		user, err := s.store.RetrieveUser(s.Context(), userID)
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(user)
	})
}

// TestUpdateUser verifies user updates, password/login/email changes, role management, and not-found handling.
func (s *storeSuite) TestUpdateUser() {
	require := s.Require()
	// Setup: load fixture user shared by subtests.
	userID := ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A")
	user, err := s.store.RetrieveUser(s.Context(), userID)
	require.NoError(err)
	require.NotNil(user)

	s.Run("HappyPath", func() {
		// Setup: mutate fields that should and should not persist.
		user.Name = sql.NullString{String: "Gary Franklin Redfield", Valid: true}
		user.Email = "gfredfield@example.com"
		user.Password = ""
		user.LastLogin = sql.NullTime{Valid: false}
		user.EmailVerified = true
		staleCreated := user.Created
		staleModified := user.Modified
		user.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)
		user.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)

		// Action: update user metadata.
		err = s.store.UpdateUser(s.Context(), user)
		require.NoError(err)

		// Assert: writable fields changed; immutable fields preserved.
		cmpt, err := s.store.RetrieveUser(s.Context(), user.ID)
		require.NoError(err)

		require.Equal(user.ID, cmpt.ID)
		require.Equal(user.Name, cmpt.Name)
		require.Equal(user.Email, cmpt.Email)
		require.NotEqual(user.Password, cmpt.Password)
		require.NotEqual(user.LastLogin, cmpt.LastLogin)
		require.NotEqual(user.EmailVerified, cmpt.EmailVerified)
		require.Equal(staleCreated, cmpt.Created)
		require.NotEqual(staleModified, cmpt.Modified)
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("UpdatePassword", func() {
		password := "$argon2id$v=19$m=65536,t=1,p=2$DT/LSMZjHhVlprmPaBSCcg==$UKT1g5gqWvKhiBC8gywVU6zepCEew0x3IW9vTWnlVlg="
		// Action: set new password hash.
		err := s.store.UpdatePassword(s.Context(), userID, password)
		require.NoError(err)

		// Assert: password updated and modified bumped.
		cmpt, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)
		require.Equal(password, cmpt.Password)
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("UpdateLastLogin", func() {
		lastLogin := time.Now().UTC()
		// Action: record last login timestamp.
		err = s.store.UpdateLastLogin(s.Context(), userID, lastLogin)
		require.NoError(err)

		// Assert: last login persisted and modified bumped.
		cmpt, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)
		require.True(cmpt.LastLogin.Valid)
		require.Equal(lastLogin, cmpt.LastLogin.Time)
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("VerifyEmail", func() {
		// Action: mark email as verified.
		err = s.store.VerifyEmail(s.Context(), userID)
		require.NoError(err)

		// Assert: email verified flag set and modified bumped.
		cmpt, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)
		require.True(cmpt.EmailVerified)
		require.WithinDuration(cmpt.Modified, time.Now(), time.Minute)
	})

	s.Run("AddRole", func() {
		// Setup: confirm keyholder role is not yet assigned.
		user, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)

		for _, role := range user.Roles {
			require.NotEqual(role.Title, "keyholder")
			require.NotEqual(role.ID, int64(4))
		}

		s.Run("Title", func() {
			// Action: add role by title.
			err := s.store.AddRoleToUserByTitle(s.Context(), userID, "keyholder")
			require.NoError(err)

			// Assert: keyholder role present after reload.
			cmpt, err := s.store.RetrieveUser(s.Context(), userID)
			require.NoError(err)

			found := false
			for _, role := range cmpt.Roles {
				if role.Title == "keyholder" {
					found = true
					break
				}
			}
			require.True(found)
			s.resetStore()
		})

		s.Run("ID", func() {
			// Action: add role by ID.
			err := s.store.AddRoleToUser(s.Context(), userID, 4)
			require.NoError(err)

			// Assert: role ID 4 present after reload.
			cmpt, err := s.store.RetrieveUser(s.Context(), userID)
			require.NoError(err)

			found := false
			for _, role := range cmpt.Roles {
				if role.ID == 4 {
					found = true
					break
				}
			}
			require.True(found)
			s.resetStore()
		})
	})

	s.Run("RemoveRole", func() {
		// Setup: confirm admin role is currently assigned.
		user, err := s.store.RetrieveUser(s.Context(), userID)
		require.NoError(err)

		found := false
		for _, role := range user.Roles {
			if role.Title == "admin" && role.ID == int64(1) {
				found = true
				break
			}
		}
		require.True(found)

		s.Run("Title", func() {
			// Action: remove role by title.
			err := s.store.RemoveRoleFromUserByTitle(s.Context(), userID, "admin")
			require.NoError(err)

			// Assert: user has no roles after reload.
			cmpt, err := s.store.RetrieveUser(s.Context(), userID)
			require.NoError(err)
			require.Empty(cmpt.Roles)
			s.resetStore()
		})

		s.Run("ID", func() {
			// Action: remove role by ID.
			err := s.store.RemoveRoleFromUser(s.Context(), userID, 1)
			require.NoError(err)

			// Assert: user has no roles after reload.
			cmpt, err := s.store.RetrieveUser(s.Context(), userID)
			require.NoError(err)
			require.Empty(cmpt.Roles)
			s.resetStore()
		})
	})

	s.Run("NotFound", func() {
		user.ID = ulid.Make()
		err = s.store.UpdateUser(s.Context(), user)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestReplaceUserRoles verifies atomically replacing a user's role set.
func (s *storeSuite) TestReplaceUserRoles() {
	require := s.Require()
	userID := ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A")

	// Setup: gary starts with a single admin role.
	user, err := s.store.RetrieveUser(s.Context(), userID)
	require.NoError(err)
	require.Len(user.Roles, 1)
	require.Equal(int64(1), user.Roles[0].ID)

	// Action: replace admin with editor + viewer.
	err = s.store.ReplaceUserRoles(s.Context(), userID, []int64{2, 3})
	require.NoError(err)

	// Assert: role set matches replacement.
	user, err = s.store.RetrieveUser(s.Context(), userID)
	require.NoError(err)
	require.Len(user.Roles, 2)

	roleIDs := make([]int64, len(user.Roles))
	for i, role := range user.Roles {
		roleIDs[i] = role.ID
	}
	require.ElementsMatch([]int64{2, 3}, roleIDs)

	// Action: clear all roles.
	err = s.store.ReplaceUserRoles(s.Context(), userID, nil)
	require.NoError(err)

	// Assert: user has no roles.
	user, err = s.store.RetrieveUser(s.Context(), userID)
	require.NoError(err)
	require.Empty(user.Roles)
}

// TestDeleteUser verifies cascade to user_roles and idempotent not-found.
func (s *storeSuite) TestDeleteUser() {
	require := s.Require()
	userCount := s.count("users")
	userRolesCount := s.count("user_roles")

	userID := ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A")

	err := s.store.DeleteUser(s.Context(), userID)
	require.NoError(err)

	require.Equal(userCount-1, s.count("users"))
	require.Equal(userRolesCount-1, s.count("user_roles"))

	err = s.store.DeleteUser(s.Context(), userID)
	require.ErrorIs(err, errors.ErrNotFound)
}
