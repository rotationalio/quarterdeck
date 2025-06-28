package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func TestAPIKeyParams(t *testing.T) {
	apikey := &APIKey{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Description: sql.NullString{Valid: true, String: "Test API Key"},
		ClientID:    "XUiRZrNDUnLjeenQQmblpv",
		Secret:      "$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=",
		CreatedBy:   ulid.MakeSecure(),
		Revoked:     sql.NullTime{Valid: false},
		LastSeen:    sql.NullTime{Valid: true, Time: time.Now()},
	}

	CheckParams(t, apikey.Params(),
		[]string{
			"id", "description", "clientID", "secret", "createdBy", "lastSeen", "revoked", "created", "modified",
		},
		[]any{
			apikey.ID, apikey.Description, apikey.ClientID, apikey.Secret, apikey.CreatedBy, apikey.LastSeen, apikey.Revoked, apikey.Created, apikey.Modified,
		},
	)
}

func TestAPIKeyScan(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(),      // ID
			"Test api keys for development", // Description
			"XUiRZrNDUnLjeenQQmblpv",        // ClientID
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=", // Secret
			ulid.MakeSecure().String(),        // CreatedBy
			time.Now().Add(-1 * time.Hour),    // LastSeen
			time.Now().Add(-30 * time.Minute), // Revoked
			time.Now().Add(-14 * time.Hour),   // Created
			time.Now().Add(-30 * time.Minute), // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &APIKey{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Description.String, "expected field Description to match data[1]")
		require.Equal(t, data[2], model.ClientID, "expected field ClientID to match data[2]")
		require.Equal(t, data[3], model.Secret, "expected field Secret to match data[3]")
		require.Equal(t, data[4], model.CreatedBy.String(), "expected field CreatedBy to match data[4]")
		require.Equal(t, data[5], model.LastSeen.Time, "expected field LastSeen to match data[5]")
		require.Equal(t, data[6], model.Revoked.Time, "expected field Revoked to match data[6]")
		require.Equal(t, data[7], model.Created, "expected field Created to match data[7]")
		require.Equal(t, data[8], model.Modified, "expected field Modified to match data[8]")
	})

	t.Run("Nulls", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(), // ID
			nil,                        // Description (testing null string)
			"XUiRZrNDUnLjeenQQmblpv",   // ClientID
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=", // Secret
			ulid.MakeSecure().String(), // CreatedBy
			nil,                        // LastSeen (testing null time)
			nil,                        // Revoked
			time.Now(),                 // Created
			time.Time{},                // Modified (testing zero time)
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		//test
		model := &APIKey{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.False(t, model.Description.Valid, "expected field Description to be invalid (null)")
		require.False(t, model.LastSeen.Valid, "expected field LastSeen to be invalid (null)")
		require.False(t, model.Revoked.Valid, "expected field Revoked to be invalid (null)")
		require.True(t, model.Modified.IsZero(), "expected field Modified to be zero time")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &APIKey{}
		err := model.Scan(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")
	})
}

func TestAPIKeyScanSummary(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(),        // ID
			"Test api keys for development",   // Description
			"XUiRZrNDUnLjeenQQmblpv",          // ClientID
			ulid.MakeSecure().String(),        // CreatedBy
			time.Now().Add(-1 * time.Hour),    // LastSeen
			nil,                               // Revoked
			time.Now().Add(-14 * time.Hour),   // Created
			time.Now().Add(-30 * time.Minute), // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &APIKey{}
		err := model.ScanSummary(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Description.String, "expected field Description to match data[1]")
		require.Equal(t, data[2], model.ClientID, "expected field ClientID to match data[2]")
		require.Zero(t, model.Secret, "!important expected field Secret to be empty!")
		require.Equal(t, data[3], model.CreatedBy.String(), "expected field CreatedBy to match data[4]")
		require.Equal(t, data[4], model.LastSeen.Time, "expected field LastSeen to match data[5]")
		require.False(t, model.Revoked.Valid, "expected field Revoked to be null")
		require.Equal(t, data[6], model.Created, "expected field Created to match data[7]")
		require.Equal(t, data[7], model.Modified, "expected field Modified to match data[8]")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &APIKey{}
		err := model.ScanSummary(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")
	})
}

func TestAPIKeyPermissions(t *testing.T) {
	apikey := &APIKey{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		Description: sql.NullString{Valid: true, String: "Test API Key"},
		ClientID:    "XUiRZrNDUnLjeenQQmblpv",
		Secret:      "$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=",
		CreatedBy:   ulid.MakeSecure(),
		LastSeen:    sql.NullTime{Valid: false},
	}

	require.Empty(t, apikey.Permissions(), "expected no permissions before setting them")
	permissions := []string{"read", "write", "delete"}
	apikey.SetPermissions(permissions)
	require.Equal(t, permissions, apikey.Permissions())
}

func TestAPIKeyStatus(t *testing.T) {
	tests := []struct {
		key      *APIKey
		expected enum.APIKeyStatus
	}{
		{
			key: &APIKey{
				Model: Model{
					ID:       modelID,
					Created:  created,
					Modified: modified,
				},
				Revoked:  sql.NullTime{Valid: false},
				LastSeen: sql.NullTime{Valid: false},
			},
			expected: enum.APIKeyStatusUnused,
		},
		{
			key: &APIKey{
				Model: Model{
					ID:       modelID,
					Created:  created,
					Modified: modified,
				},
				Revoked:  sql.NullTime{Valid: false},
				LastSeen: sql.NullTime{Valid: true, Time: time.Now().Add(1492 * time.Hour)},
			},
			expected: enum.APIKeyStatusActive,
		},
		{
			key: &APIKey{
				Model: Model{
					ID:       modelID,
					Created:  created,
					Modified: modified,
				},
				Revoked:  sql.NullTime{Valid: false},
				LastSeen: sql.NullTime{Valid: true, Time: time.Now().Add(-3138 * time.Hour)},
			},
			expected: enum.APIKeyStatusStale,
		},
		{
			key: &APIKey{
				Model: Model{
					ID:       modelID,
					Created:  created,
					Modified: modified,
				},
				Revoked:  sql.NullTime{Valid: true, Time: time.Now().Add(-1 * time.Hour)},
				LastSeen: sql.NullTime{Valid: true, Time: time.Now().Add(-1492 * time.Hour)},
			},
			expected: enum.APIKeyStatusRevoked,
		},
	}

	for i, tc := range tests {
		require.Equal(t, tc.expected, tc.key.Status(), "status test failed on test case %d: expected %s got %s", i, tc.expected, tc.key.Status())
	}
}
