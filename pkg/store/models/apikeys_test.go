package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
		LastSeen:    sql.NullTime{Valid: false},
	}

	CheckParams(t, apikey.Params(),
		[]string{
			"id", "description", "clientID", "secret", "lastSeen", "created", "modified",
		},
		[]any{
			apikey.ID, apikey.Description, apikey.ClientID, apikey.Secret, apikey.LastSeen, apikey.Created, apikey.Modified,
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
			time.Now().Add(-1 * time.Hour),  // LastSeen
			time.Now().Add(-14 * time.Hour), // Created
			time.Now().Add(-1 * time.Hour),  // Modified
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
		require.Equal(t, data[4], model.LastSeen.Time, "expected field LastSeen to match data[4]")
		require.Equal(t, data[5], model.Created, "expected field Created to match data[5]")
		require.Equal(t, data[6], model.Modified, "expected field Modified to match data[6]")
	})

	t.Run("Nulls", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(), // ID
			nil,                        // Description (testing null string)
			"XUiRZrNDUnLjeenQQmblpv",   // ClientID
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=", // Secret
			nil,         // LastSeen (testing null time)
			time.Now(),  // Created
			time.Time{}, // Modified (testing zero time)
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
			ulid.MakeSecure().String(),      // ID
			"Test api keys for development", // Description
			"XUiRZrNDUnLjeenQQmblpv",        // ClientID
			time.Now().Add(-1 * time.Hour),  // LastSeen
			time.Now().Add(-14 * time.Hour), // Created
			time.Now().Add(-1 * time.Hour),  // Modified
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
		require.Equal(t, data[3], model.LastSeen.Time, "expected field LastSeen to match data[3]")
		require.Equal(t, data[4], model.Created, "expected field Created to match data[4]")
		require.Equal(t, data[5], model.Modified, "expected field Modified to match data[5]")
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
		LastSeen:    sql.NullTime{Valid: false},
	}

	require.Empty(t, apikey.Permissions(), "expected no permissions before setting them")
	permissions := []string{"read", "write", "delete"}
	apikey.SetPermissions(permissions)
	require.Equal(t, permissions, apikey.Permissions())
}
