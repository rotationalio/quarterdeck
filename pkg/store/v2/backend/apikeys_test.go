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
// API Key Store Tests
//=============================================================================

// TestAPIKeyList verifies all non-revoked fixture API keys are returned.
func (s *storeSuite) TestAPIKeyList() {
	cursor, err := s.store.ListAPIKeys(s.Context(), nil)
	s.Require().NoError(err)
	defer func() { s.Require().NoError(cursor.Close()) }()
	keys, err := cursor.List()
	s.Require().NoError(err)
	s.Len(keys, 3)

	for _, key := range keys {
		s.False(key.Revoked.Valid)
	}
}

// TestAPIKeyListRevokedFilter verifies ListAPIKeys revoked-key filtering for nil,
// tidal.Filter, and tidal.Clause filters as documented on ListAPIKeys.
func (s *storeSuite) TestAPIKeyListRevokedFilter() {
	const (
		activeKeys  = 3
		totalKeys   = 5
		revokedKeys = 2
	)

	listKeys := func(filter tidal.ListFilter) []*models.APIKey {
		cursor, err := s.store.ListAPIKeys(s.Context(), filter)
		s.Require().NoError(err)
		defer func() { s.Require().NoError(cursor.Close()) }()
		keys, err := cursor.List()
		s.Require().NoError(err)
		return keys
	}

	countRevoked := func(keys []*models.APIKey) int {
		n := 0
		for _, key := range keys {
			if key.Revoked.Valid {
				n++
			}
		}
		return n
	}

	s.Run("NilFilter", func() {
		keys := listKeys(nil)
		s.Len(keys, activeKeys)
		s.Equal(0, countRevoked(keys), "should return only active keys")
	})

	s.Run("Filter", func() {
		filter := (&tidal.Filter{}).OrderBy("-created").Limit(10)
		keys := listKeys(filter)
		s.Len(keys, activeKeys)
		s.Equal(0, countRevoked(keys), "should return only active keys")
	})

	s.Run("FilterWithWhere", func() {
		filter := (&tidal.Filter{}).
			Where("client_id", tidal.Eq, "TPAkoalHEorqAENISHvxYY").
			OrderBy("-created")
		keys := listKeys(filter)
		s.Len(keys, 1)
		s.Equal("TPAkoalHEorqAENISHvxYY", keys[0].ClientID)
		s.Equal(0, countRevoked(keys), "should return only active keys")
	})

	s.Run("ClauseWithoutRevokedFilter", func() {
		filter := &tidal.Clause{SQL: "ORDER BY created"}
		keys := listKeys(filter)
		s.Len(keys, totalKeys)
		s.Equal(revokedKeys, countRevoked(keys), "should return all keys")
	})

	s.Run("ClauseWithRevokedFilter", func() {
		filter := &tidal.Clause{SQL: "WHERE revoked IS NULL ORDER BY created"}
		keys := listKeys(filter)
		s.Len(keys, activeKeys)
		s.Equal(0, countRevoked(keys), "should return only active keys")
	})
}

// TestCreateAPIKey verifies validation, permission assignment, and duplicate client ID rejection.
func (s *storeSuite) TestCreateAPIKey() {
	s.Run("NoIDOnCreate", func() {
		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "DtptIgWgzkwaibktjczVwr",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$IoXhViMsvBCFkA6NqU38vw==$zmsktbWyYxOuX55yB1o6KC8I3hZltxtU53rOWggs2IM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}
		key.ID = ulid.Make()

		_, err := s.store.CreateAPIKey(s.Context(), key)
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("NoPermissions", func() {
		require := s.Require()
		keyCount := s.count("api_keys")
		keyPermsCount := s.count("api_key_permissions")

		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "DtptIgWgzkwaibktjczVwr",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$IoXhViMsvBCFkA6NqU38vw==$zmsktbWyYxOuX55yB1o6KC8I3hZltxtU53rOWggs2IM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		created, err := s.store.CreateAPIKey(s.Context(), key)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.WithinDuration(time.Now(), created.Created, 3*time.Second)
		require.WithinDuration(time.Now(), created.Modified, 3*time.Second)
		require.Equal(keyCount+1, s.count("api_keys"))
		require.Equal(keyPermsCount, s.count("api_key_permissions"))
	})

	s.Run("WithPermissions", func() {
		require := s.Require()
		keyCount := s.count("api_keys")
		keyPermsCount := s.count("api_key_permissions")

		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "hsuAdrWAmPJnzCgDsyTQiN",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$/pvm23rK6tGeC+rh91UiJA==$ZAax7OxfvSr7yr3sxC2ViOhtxCDSWr70RFHAV2F76FM=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}
		key.Permissions = []models.Permission{
			{Title: "content:modify"},
			{Title: "content:view"},
			{Title: "content:delete"},
			{Title: "users:view"},
		}

		created, err := s.store.CreateAPIKey(s.Context(), key)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.WithinDuration(time.Now(), created.Created, 3*time.Second)
		require.WithinDuration(time.Now(), created.Modified, 3*time.Second)
		require.Equal(keyCount+1, s.count("api_keys"))
		require.Equal(keyPermsCount+4, s.count("api_key_permissions"))
	})

	s.Run("NoDuplicateClientID", func() {
		key := &models.APIKey{
			Description: sql.NullString{String: "Test API Key", Valid: true},
			ClientID:    "TPAkoalHEorqAENISHvxYY",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$nXCe+4HPx0YfO/BMRTtePQ==$vRxaszj/Y4NtfqL7DYDKp3zILXuAnEpzxCtCAc1fdTk=",
			CreatedBy:   ulid.MustParse("01JPYRNYMEHNEZCS0JYX1CP57A"),
		}

		_, err := s.store.CreateAPIKey(s.Context(), key)
		s.Require().ErrorIs(err, errors.ErrAlreadyExists)
	})
}

// TestRetrieveAPIKey verifies lookup by ID, client ID, and not-found cases.
func (s *storeSuite) TestRetrieveAPIKey() {
	s.Run("ByID", func() {
		require := s.Require()
		key, err := s.store.RetrieveAPIKey(s.Context(), ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT"))
		require.NoError(err)
		require.NotNil(key)

		require.Equal("01JNH8ZKWFJ2Z8E3GJTQTFPQCT", key.ID.String())
		require.Equal("Read/view only keys", key.Description.String)
		require.Equal("TPAkoalHEorqAENISHvxYY", key.ClientID)
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$8J11ntVv8i3YBGA74QCS/w==$mOINU411zwT0lNO03UBkMI7l9Mz7rA3XAiQpDIXVVh0=", key.Secret)
		require.Equal("01JMJMGHQSA2SHQ8S1T4JXABFJ", key.CreatedBy.String())
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.LastSeen.Time)
		require.False(key.Revoked.Valid)
		require.Equal(time.Date(2025, time.March, 4, 19, 9, 6, 0, time.UTC), key.Created)
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.Modified)

		permissions := models.PermissionTitles(key.Permissions)
		require.Len(permissions, 3)
		require.Contains(permissions, "keys:view")
		require.Contains(permissions, "content:view")
		require.Contains(permissions, "users:view")
	})

	s.Run("ByClientID", func() {
		require := s.Require()
		key, err := s.store.RetrieveAPIKeyByClientID(s.Context(), "TPAkoalHEorqAENISHvxYY")
		require.NoError(err)
		require.NotNil(key)

		require.Equal("01JNH8ZKWFJ2Z8E3GJTQTFPQCT", key.ID.String())
		require.Equal("Read/view only keys", key.Description.String)
		require.Equal("TPAkoalHEorqAENISHvxYY", key.ClientID)
		require.Equal("$argon2id$v=19$m=65536,t=1,p=2$8J11ntVv8i3YBGA74QCS/w==$mOINU411zwT0lNO03UBkMI7l9Mz7rA3XAiQpDIXVVh0=", key.Secret)
		require.Equal("01JMJMGHQSA2SHQ8S1T4JXABFJ", key.CreatedBy.String())
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.LastSeen.Time)
		require.False(key.Revoked.Valid)
		require.Equal(time.Date(2025, time.March, 4, 19, 9, 6, 0, time.UTC), key.Created)
		require.Equal(time.Date(2025, time.May, 24, 18, 41, 58, 0, time.UTC), key.Modified)

		permissions := models.PermissionTitles(key.Permissions)
		require.Len(permissions, 3)
		require.Contains(permissions, "keys:view")
		require.Contains(permissions, "content:view")
		require.Contains(permissions, "users:view")
	})

	s.Run("NotFound", func() {
		require := s.Require()
		key, err := s.store.RetrieveAPIKey(s.Context(), ulid.Make())
		require.ErrorIs(err, errors.ErrNotFound)
		require.Nil(key)
	})
}

// TestUpdateAPIKey verifies key updates, last-seen tracking, permission changes, and not-found handling.
func (s *storeSuite) TestUpdateAPIKey() {
	require := s.Require()
	// Setup: load fixture key shared by subtests.
	keyID := ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT")
	key, err := s.store.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err)
	require.NotNil(key)

	s.Run("HappyPath", func() {
		// Setup: mutate fields that should and should not persist.
		key.Description = sql.NullString{String: "Updated API Key Description", Valid: true}
		key.ClientID = "UpdatedClientID12345"
		key.Secret = ""
		key.CreatedBy = ulid.Make()
		key.LastSeen = sql.NullTime{Time: time.Now(), Valid: true}
		key.Revoked = sql.NullTime{Time: time.Now(), Valid: true}
		key.Created = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)
		key.Modified = time.Date(2025, time.January, 26, 14, 13, 12, 0, time.UTC)

		// Action: update key metadata.
		err := s.store.UpdateAPIKey(s.Context(), key)
		require.NoError(err)

		// Assert: writable fields changed; immutable fields preserved.
		cmpt, err := s.store.RetrieveAPIKey(s.Context(), keyID)
		require.NoError(err)

		require.Equal(key.ID, cmpt.ID)
		require.Equal("Updated API Key Description", cmpt.Description.String)
		require.NotEqual(key.ClientID, cmpt.ClientID)
		require.NotEqual(key.Secret, cmpt.Secret)
		require.NotEqual(key.CreatedBy, cmpt.CreatedBy)
		require.NotEqual(key.LastSeen.Time, cmpt.LastSeen.Time)
		require.False(cmpt.Revoked.Valid)
		require.NotEqual(key.Created, cmpt.Created)
		require.WithinDuration(time.Now(), cmpt.Modified, 3*time.Second)
	})

	s.Run("UpdateLastSeen", func() {
		// Action: record last seen timestamp.
		err := s.store.UpdateLastSeen(s.Context(), keyID, time.Now())
		require.NoError(err)

		// Assert: last seen updated on reload.
		cmpt, err := s.store.RetrieveAPIKey(s.Context(), keyID)
		require.NoError(err)
		require.WithinDuration(time.Now(), cmpt.LastSeen.Time, 3*time.Second)
	})

	s.Run("AddPermission", func() {
		// Setup: confirm keys:revoke permission is not yet assigned.
		permissions := models.PermissionTitles(key.Permissions)
		require.NotContains(permissions, "keys:revoke")

		s.Run("Title", func() {
			permsCount := s.count("api_key_permissions")

			// Action: add permission by title.
			err := s.store.AddPermissionToAPIKeyByTitle(s.Context(), key.ID, "keys:revoke")
			require.NoError(err)
			require.Equal(permsCount+1, s.count("api_key_permissions"))

			// Assert: permission present after reload.
			cmpt, err := s.store.RetrieveAPIKey(s.Context(), keyID)
			require.NoError(err)
			require.Contains(models.PermissionTitles(cmpt.Permissions), "keys:revoke")
			s.resetStore()
		})

		s.Run("ID", func() {
			permsCount := s.count("api_key_permissions")

			// Action: add permission by ID.
			err := s.store.AddPermissionToAPIKey(s.Context(), key.ID, 9)
			require.NoError(err)
			require.Equal(permsCount+1, s.count("api_key_permissions"))

			// Assert: permission present after reload.
			cmpt, err := s.store.RetrieveAPIKey(s.Context(), keyID)
			require.NoError(err)
			require.Contains(models.PermissionTitles(cmpt.Permissions), "keys:revoke")
			s.resetStore()
		})
	})

	s.Run("RemovePermission", func() {
		// Setup: confirm content:view permission is currently assigned.
		permissions := models.PermissionTitles(key.Permissions)
		require.Contains(permissions, "content:view")

		permsCount := s.count("api_key_permissions")

		// Action: remove permission by ID.
		err := s.store.RemovePermissionFromAPIKey(s.Context(), key.ID, 2)
		require.NoError(err)
		require.Equal(permsCount-1, s.count("api_key_permissions"))

		// Assert: permission absent after reload.
		cmpt, err := s.store.RetrieveAPIKey(s.Context(), keyID)
		require.NoError(err)
		require.NotContains(models.PermissionTitles(cmpt.Permissions), "content:view")
		s.resetStore()
	})

	s.Run("NotFound", func() {
		key.ID = ulid.Make()
		key.Description = sql.NullString{String: "Updated API Key Description", Valid: true}

		err := s.store.UpdateAPIKey(s.Context(), key)
		require.ErrorIs(err, errors.ErrNotFound)
	})
}

// TestReplaceAPIKeyPermissions verifies atomically replacing an API key's permission set.
func (s *storeSuite) TestReplaceAPIKeyPermissions() {
	require := s.Require()
	keyID := ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT")

	// Setup: read-only fixture key has three permissions.
	key, err := s.store.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err)
	require.Len(key.Permissions, 3)

	// Action: swap to content:modify + content:delete only.
	err = s.store.ReplaceAPIKeyPermissions(s.Context(), keyID, []int64{1, 3})
	require.NoError(err)

	// Assert: permission set matches replacement.
	key, err = s.store.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err)
	require.Len(key.Permissions, 2)
	require.Contains(models.PermissionTitles(key.Permissions), "content:modify")
	require.Contains(models.PermissionTitles(key.Permissions), "content:delete")

	// Action: clear all permissions.
	err = s.store.ReplaceAPIKeyPermissions(s.Context(), keyID, nil)
	require.NoError(err)

	// Assert: key has no direct permissions.
	key, err = s.store.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err)
	require.Empty(key.Permissions)
}

// TestCreateAPIKeyForCreator verifies delegated key creation with creator authorization checks.
func (s *storeSuite) TestCreateAPIKeyForCreator() {
	s.Run("NoIDOnCreate", func() {
		key := &models.APIKey{
			Description: sql.NullString{String: "Delegated key", Valid: true},
			ClientID:    "delegatedClient01",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$IoXhViMsvBCFkA6NqU38vw==$zmsktbWyYxOuX55yB1o6KC8I3hZltxtU53rOWggs2IM=",
		}
		key.ID = ulid.Make()

		_, err := s.store.CreateAPIKeyFor(s.Context(), key, ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT"))
		s.Require().ErrorIs(err, errors.ErrNoIDOnCreate)
	})

	s.Run("RevokedCreator", func() {
		creator, err := s.store.RetrieveAPIKeyByClientID(s.Context(), "yfoPxjgVyleDkpOPnNfsBG")
		s.Require().NoError(err)

		key := &models.APIKey{
			Description: sql.NullString{String: "Delegated key", Valid: true},
			ClientID:    "delegatedClient02",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$/pvm23rK6tGeC+rh91UiJA==$ZAax7OxfvSr7yr3sxC2ViOhtxCDSWr70RFHAV2F76FM=",
		}

		_, err = s.store.CreateAPIKeyFor(s.Context(), key, creator.ID)
		s.Require().ErrorIs(err, errors.ErrNotAuthorized)
	})

	s.Run("HappyPath", func() {
		require := s.Require()
		keyCount := s.count("api_keys")
		creatorID := ulid.MustParse("01JNH8ZKWFJ2Z8E3GJTQTFPQCT")

		key := &models.APIKey{
			Description: sql.NullString{String: "Delegated key", Valid: true},
			ClientID:    "delegatedClient03",
			Secret:      "$argon2id$v=19$m=65536,t=1,p=2$nXCe+4HPx0YfO/BMRTtePQ==$vRxaszj/Y4NtfqL7DYDKp3zILXuAnEpzxCtCAc1fdTk=",
		}
		key.Permissions = []models.Permission{{Title: "content:view"}}

		// Action: creator key delegates a new key; CreatedBy should inherit from creator.
		created, err := s.store.CreateAPIKeyFor(s.Context(), key, creatorID)
		require.NoError(err)
		require.False(created.ID.IsZero())
		require.Equal("01JMJMGHQSA2SHQ8S1T4JXABFJ", created.CreatedBy.String())
		require.Contains(models.PermissionTitles(created.Permissions), "content:view")
		require.Equal(keyCount+1, s.count("api_keys"))
	})
}

// TestRevokeAPIKey verifies soft-revoke sets revoked timestamp without deleting rows.
func (s *storeSuite) TestRevokeAPIKey() {
	require := s.Require()
	keyCount := s.count("api_keys")
	keyPermissionsCount := s.count("api_key_permissions")

	keyID := ulid.MustParse("01JX2EX9XHAR5XHRWVZFCGAYK1")

	err := s.store.RevokeAPIKey(s.Context(), keyID)
	require.NoError(err)

	require.Equal(keyCount, s.count("api_keys"))
	require.Equal(keyPermissionsCount, s.count("api_key_permissions"))

	key, err := s.store.RetrieveAPIKey(s.Context(), keyID)
	require.NoError(err)
	require.WithinDuration(time.Now(), key.Revoked.Time, 3*time.Second)
}

// TestDeleteAPIKey verifies cascade to api_key_permissions and idempotent not-found.
func (s *storeSuite) TestDeleteAPIKey() {
	require := s.Require()
	keyCount := s.count("api_keys")
	keyPermissionsCount := s.count("api_key_permissions")

	keyID := ulid.MustParse("01JX2EX9XHAR5XHRWVZFCGAYK1")

	err := s.store.DeleteAPIKey(s.Context(), keyID)
	require.NoError(err)

	require.Equal(keyCount-1, s.count("api_keys"))
	require.Equal(keyPermissionsCount-2, s.count("api_key_permissions"))

	err = s.store.DeleteAPIKey(s.Context(), keyID)
	require.ErrorIs(err, errors.ErrNotFound)
}
