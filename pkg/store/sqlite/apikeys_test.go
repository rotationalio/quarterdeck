package sqlite_test

import (
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestAPIKeyList() {
	require := s.Require()
	out, err := s.db.ListAPIKeys(s.Context(), nil)
	require.NoError(err, "should be able to list api keys")
	require.NotNil(out, "should return an api key list")
	require.Len(out.APIKeys, 3, "api key list should return 3 keys and none that are revoked")

	// Ensure no keys returned are revoked
	for _, key := range out.APIKeys {
		require.False(key.Revoked.Valid, "api key should not be revoked")
	}
}

func (s *storeTestSuite) TestCreateAPIKey() {
	s.Run("NoIDOnCreate", func() {
		key := &models.APIKey{
			Model: models.Model{
				ID:       ulid.Make(),
				Created:  time.Now(),
				Modified: time.Now(),
			},
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "DtptIgWgzkwaibktjczVwr",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$IoXhViMsvBCFkA6NqU38vw==$zmsktbWyYxOuX55yB1o6KC8I3hZltxtU53rOWggs2IM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		err := s.db.CreateAPIKey(s.Context(), key)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate, "should not allow creating API key with ID set")
	})

	s.Run("ReadOnly", func() {
		if !s.ReadOnly() {
			s.T().Skip("skipping create read-only error test in read-write mode")
		}

		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "DtptIgWgzkwaibktjczVwr",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$IoXhViMsvBCFkA6NqU38vw==$zmsktbWyYxOuX55yB1o6KC8I3hZltxtU53rOWggs2IM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		err := s.db.CreateAPIKey(s.Context(), key)
		s.Require().ErrorIs(err, errors.ErrReadOnly, "should not allow creating API key in read-only mode")
	})

	s.Run("NoPermissions", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		require := s.Require()
		keyCount := s.Count("api_keys")
		keyPermsCount := s.Count("api_key_permissions")

		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "DtptIgWgzkwaibktjczVwr",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$IoXhViMsvBCFkA6NqU38vw==$zmsktbWyYxOuX55yB1o6KC8I3hZltxtU53rOWggs2IM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		err := s.db.CreateAPIKey(s.Context(), key)
		require.NoError(err, "should be able to create API key without permissions")
		require.NotNil(key.ID, "should set the api key ID on create")
		require.WithinDuration(time.Now(), key.Created, 3*time.Second, "created timestamp should be set")
		require.WithinDuration(time.Now(), key.Modified, 3*time.Second, "modified timestamp should be set")
		require.Equal(keyCount+1, s.Count("api_keys"), "should increase the API key count by one")
		require.Equal(keyPermsCount, s.Count("api_key_permissions"), "should not create any API key permissions")
	})

	s.Run("WithPermissions", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		require := s.Require()
		keyCount := s.Count("api_keys")
		keyPermsCount := s.Count("api_key_permissions")

		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "hsuAdrWAmPJnzCgDsyTQiN",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$/pvm23rK6tGeC+rh91UiJA==$ZAax7OxfvSr7yr3sxC2ViOhtxCDSWr70RFHAV2F76FM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		key.SetPermissions([]string{
			"content:modify", "content:view", "content:delete", "users:view",
		})

		err := s.db.CreateAPIKey(s.Context(), key)
		require.NoError(err, "should be able to create API key without permissions")
		require.NotNil(key.ID, "should set the api key ID on create")
		require.WithinDuration(time.Now(), key.Created, 3*time.Second, "created timestamp should be set")
		require.WithinDuration(time.Now(), key.Modified, 3*time.Second, "modified timestamp should be set")
		require.Equal(keyCount+1, s.Count("api_keys"), "should increase the API key count by one")
		require.Equal(keyPermsCount+4, s.Count("api_key_permissions"), "should not create any API key permissions")
	})

	s.Run("NoDuplicateClientID", func() {
		if s.ReadOnly() {
			s.T().Skip("skipping create test in read-only mode")
		}

		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "TPAkoalHEorqAENISHvxYY",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$nXCe+4HPx0YfO/BMRTtePQ==$vRxaszj/Y4NtfqL7DYDKp3zILXuAnEpzxCtCAc1fdTk=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		err := s.db.CreateAPIKey(s.Context(), key)
		s.Require().ErrorIs(err, errors.ErrAlreadyExists, "should not allow creating API key with duplicate client ID")
	})
}

func (s *storeTestSuite) TestRetrieveAPIKey() {
	s.Run("ByID", func() {
		require := s.Require()
		key, err := s.db.RetrieveAPIKey(s.Context(), ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT"))
		require.NoError(err, "should be able to retrieve API key by ID")
		require.NotNil(key, "should return an API key")

		require.Equal("01JNH8ZKWFJ2Z8E3GJTQTFPQCT", key.ID.String(), "should return the correct API key ID")
		require.Equal("Read/view only keys", key.Description.String, "should return the correct description")
		require.Equal("TPAkoalHEorqAENISHvxYY", key.ClientID, "should return the correct client ID")
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$8J11ntVv8i3YBGA74QCS/w==$mOINU411zwT0lNO03UBkMI7l9Mz7rA3XAiQpDIXVVh0=", key.Secret, "should return the correct derived key secret")
		require.Equal("01JMJMGHQSA2SHQ8S1T4JXABFJ", key.CreatedBy.String(), "should return the correct created by user ID")
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.LastSeen.Time, "should return the correct last seen time")
		require.False(key.Revoked.Valid, "should return the correct revoked time")
		require.Equal(time.Date(2025, time.March, 4, 19, 9, 6, 0, time.UTC), key.Created, "should return the correct created time")
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.Modified, "should return the correct modified time")

		permissions := key.Permissions()
		require.Len(permissions, 3, "should return the correct number of permissions")
		require.Contains(permissions, "keys:view", "should return the keys:view permission")
		require.Contains(permissions, "content:view", "should return the content:view permission")
		require.Contains(permissions, "users:view", "should return the users:view permission")
	})

	s.Run("ByClientID", func() {
		require := s.Require()
		key, err := s.db.RetrieveAPIKey(s.Context(), "TPAkoalHEorqAENISHvxYY")
		require.NoError(err, "should be able to retrieve API key by client ID")
		require.NotNil(key, "should return an API key")

		require.Equal("01JNH8ZKWFJ2Z8E3GJTQTFPQCT", key.ID.String(), "should return the correct API key ID")
		require.Equal("Read/view only keys", key.Description.String, "should return the correct description")
		require.Equal("TPAkoalHEorqAENISHvxYY", key.ClientID, "should return the correct client ID")
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$8J11ntVv8i3YBGA74QCS/w==$mOINU411zwT0lNO03UBkMI7l9Mz7rA3XAiQpDIXVVh0=", key.Secret, "should return the correct derived key secret")
		require.Equal("01JMJMGHQSA2SHQ8S1T4JXABFJ", key.CreatedBy.String(), "should return the correct created by user ID")
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.LastSeen.Time, "should return the correct last seen time")
		require.False(key.Revoked.Valid, "should return the correct revoked time")
		require.Equal(time.Date(2025, time.March, 4, 19, 9, 6, 0, time.UTC), key.Created, "should return the correct created time")
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.Modified, "should return the correct modified time")

		permissions := key.Permissions()
		require.Len(permissions, 3, "should return the correct number of permissions")
		require.Contains(permissions, "keys:view", "should return the keys:view permission")
		require.Contains(permissions, "content:view", "should return the content:view permission")
		require.Contains(permissions, "users:view", "should return the users:view permission")
	})

	s.Run("NotFound", func() {
		require := s.Require()
		key, err := s.db.RetrieveAPIKey(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error for non-existent API key")
		require.Nil(key, "should not return an API key")
	})

	s.Run("IDTypeError", func() {
		require := s.Require()
		key, err := s.db.RetrieveAPIKey(s.Context(), 42)
		require.EqualError(err, "invalid type int for API key ID")
		require.Nil(key, "should not return a key for invalid ID type")
	})
}

func (s *storeTestSuite) TestUpdateAPIKey() {
	if s.ReadOnly() {
		s.T().Skip("skipping update test in read-only mode")
	}

	require := s.Require()
	keyID := ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT")
	key, err := s.db.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err, "should be able to retrieve API key by ID")
	require.NotNil(key, "should return an API key")

	s.Run("HappyPath", func() {
		key.Description = sql.NullString{String: "Updated API Key Description", Valid: true}
		key.ClientID = "UpdatedClientID12345"                                     // Should not change
		key.Secret = ""                                                           // Should not change
		key.CreatedBy = ulid.Make()                                               // Should not change
		key.LastSeen = sql.NullTime{Time: time.Now(), Valid: true}                // Should not change
		key.Revoked = sql.NullTime{Time: time.Now(), Valid: true}                 // Should not change
		key.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)  // Should not change
		key.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC) // Should be set to now

		err := s.db.UpdateAPIKey(s.Context(), key)
		require.NoError(err, "should be able to update API key")

		// Fetch the updated key and verify changes
		cmpt, err := s.db.RetrieveAPIKey(s.Context(), keyID)
		require.NoError(err, "should be able to retrieve updated API key")

		require.Equal(key.ID, cmpt.ID, "should retain the same API key ID")
		require.Equal("Updated API Key Description", cmpt.Description.String, "should update the description")
		require.NotEqual(key.ClientID, cmpt.ClientID, "should not update the client ID")
		require.NotEqual(key.Secret, cmpt.Secret, "should not change the secret")
		require.NotEqual(key.CreatedBy, cmpt.CreatedBy, "should not change the created by user ID")
		require.NotEqual(key.LastSeen.Time, cmpt.LastSeen.Time, "should not change the last seen time")
		require.False(cmpt.Revoked.Valid, "should not change the revoked time")
		require.NotEqual(key.Created, cmpt.Created, "should not change the created time")
		require.WithinDuration(time.Now(), cmpt.Modified, 3*time.Second, "should update the modified time to now")
	})

	s.Run("UpdateLastSeen", func() {
		err := s.db.UpdateLastSeen(s.Context(), keyID, time.Now())
		require.NoError(err, "should be able to update last seen time")

		cmpt, err := s.db.RetrieveAPIKey(s.Context(), keyID)
		require.NoError(err, "should be able to retrieve updated API key after last seen update")
		require.WithinDuration(time.Now(), cmpt.LastSeen.Time, 3*time.Second, "should update the last seen time to now")
	})

	s.Run("AddPermission", func() {
		// Ensure the key does not have the keys:revoke permission before running tests
		permissions := key.Permissions()
		require.NotContains(permissions, "keys:revoke", "API key fixture should not have keys:revoke permission for this test")

		s.Run("Title", func() {
			permsCount := s.Count("api_key_permissions")

			err := s.db.AddPermissionToAPIKey(s.Context(), key.ID, "keys:revoke")
			require.NoError(err, "should be able to add permission to API key")

			require.Equal(permsCount+1, s.Count("api_key_permissions"), "should increase the API key permissions count by one")

			cmpt, err := s.db.RetrieveAPIKey(s.Context(), keyID)
			require.NoError(err, "should be able to retrieve updated API key after adding permission")
			permissions = cmpt.Permissions()
			require.Contains(permissions, "keys:revoke", "API key should have keys:revoke permission after being added")

			s.ResetDB()
		})

		s.Run("ID", func() {
			permsCount := s.Count("api_key_permissions")

			err := s.db.AddPermissionToAPIKey(s.Context(), key.ID, 9)
			require.NoError(err, "should be able to add permission to API key")

			require.Equal(permsCount+1, s.Count("api_key_permissions"), "should increase the API key permissions count by one")

			cmpt, err := s.db.RetrieveAPIKey(s.Context(), keyID)
			require.NoError(err, "should be able to retrieve updated API key after adding permission")
			permissions = cmpt.Permissions()
			require.Contains(permissions, "keys:revoke", "API key should have keys:revoke permission after being added")

			s.ResetDB()
		})

		s.Run("Model", func() {
			permsCount := s.Count("api_key_permissions")

			err := s.db.AddPermissionToAPIKey(s.Context(), key.ID, &models.Permission{ID: 9, Title: "keys:revoke"})
			require.NoError(err, "should be able to add permission to API key")

			require.Equal(permsCount+1, s.Count("api_key_permissions"), "should increase the API key permissions count by one")

			cmpt, err := s.db.RetrieveAPIKey(s.Context(), keyID)
			require.NoError(err, "should be able to retrieve updated API key after adding permission")
			permissions = cmpt.Permissions()
			require.Contains(permissions, "keys:revoke", "API key should have keys:revoke permission after being added")

			s.ResetDB()
		})
	})

	s.Run("RemovePermission", func() {
		// Ensure the key does has the content:view permission before running tests
		permissions := key.Permissions()
		require.Contains(permissions, "content:view", "API key fixture should have content:view permission for this test")

		permsCount := s.Count("api_key_permissions")

		err := s.db.RemovePermissionFromAPIKey(s.Context(), key.ID, 2)
		require.NoError(err, "should be able to remove permission from API key")

		require.Equal(permsCount-1, s.Count("api_key_permissions"), "should decrease the API key permissions count by one")

		cmpt, err := s.db.RetrieveAPIKey(s.Context(), keyID)
		require.NoError(err, "should be able to retrieve updated API key after removing permission")
		permissions = cmpt.Permissions()
		require.NotContains(permissions, "content:view", "API key should not have content:view permission after being removed")

		s.ResetDB()
	})

	s.Run("NotFound", func() {
		key.ID = ulid.Make() // Use a new ID that does not exist
		key.Description = sql.NullString{String: "Updated API Key Description", Valid: true}
		key.ClientID = "UpdatedClientID12345"                                     // Should not change
		key.Secret = ""                                                           // Should not change
		key.CreatedBy = ulid.Make()                                               // Should not change
		key.LastSeen = sql.NullTime{Time: time.Now(), Valid: true}                // Should not change
		key.Revoked = sql.NullTime{Time: time.Now(), Valid: true}                 // Should not change
		key.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)  // Should not change
		key.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC) // Should be set to now

		err := s.db.UpdateAPIKey(s.Context(), key)
		require.ErrorIs(err, errors.ErrNotFound, "should return not found error for non-existent API key")
	})
}

func (s *storeTestSuite) TestRevokeAPIKey() {
	if s.ReadOnly() {
		s.T().Skip("skipping delete test in read-only mode")
	}

	require := s.Require()
	keyCount := s.Count("api_keys")
	keyPermissionsCount := s.Count("api_key_permissions")

	keyID := ulid.MustParse("01JX2EX9XHAR5XHRWVZFCGAYK1")

	err := s.db.RevokeAPIKey(s.Context(), keyID)
	require.NoError(err, "should be able to delete API key")

	// Counts should not have changed after revoking the key
	require.Equal(keyCount, s.Count("api_keys"), "key count should not change after revoking")
	require.Equal(keyPermissionsCount, s.Count("api_key_permissions"), "permissions count should not change after revoking")

	// When we fetch the key again, we should be able to see the revoked timestamp
	key, err := s.db.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err, "could not retrieve revoked key")
	require.WithinDuration(time.Now(), key.Revoked.Time, 3*time.Second, "revoked timestamp should be set")
}

func (s *storeTestSuite) TestDeleteAPIKey() {
	if s.ReadOnly() {
		s.T().Skip("skipping delete test in read-only mode")
	}

	require := s.Require()
	keyCount := s.Count("api_keys")
	keyPermissionsCount := s.Count("api_key_permissions")

	keyID := ulid.MustParse("01JX2EX9XHAR5XHRWVZFCGAYK1")

	err := s.db.DeleteAPIKey(s.Context(), keyID)
	require.NoError(err, "should be able to delete API key")

	require.Equal(keyCount-1, s.Count("api_keys"), "should decrease the API key count by one")
	require.Equal(keyPermissionsCount-2, s.Count("api_key_permissions"), "should decrease the API key permissions count by one")

	err = s.db.DeleteAPIKey(s.Context(), keyID)
	require.ErrorIs(err, errors.ErrNotFound)
}
