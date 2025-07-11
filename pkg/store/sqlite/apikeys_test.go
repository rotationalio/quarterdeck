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
		s.Require().EqualError(err, "sqlite3 error: UNIQUE constraint failed: api_keys.client_id", "should not allow creating API key with duplicate client ID")
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

	s.Run("HappyPath", func() {})

	s.Run("UpdateLastSeen", func() {})

	s.Run("AddPermission", func() {})

	s.Run("RemovePermission", func() {})

	s.Run("NotFound", func() {})
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
