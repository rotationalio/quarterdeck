package models_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// Database Conformance Tests
//=============================================================================

// TestAPIKeyCRUDConformance verifies APIKey satisfies tidal CRUD shape expectations against the api_keys table.
func (s *modelSuite) TestAPIKeyCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*APIKey]{
		Table: "api_keys",
		Create: func() *APIKey {
			return &APIKey{
				Description: sql.NullString{Valid: true, String: "Conformance API Key"},
				ClientID:    fmt.Sprintf("client-%s", ulid.MakeSecure().String()),
				Secret:      fmt.Sprintf("secret-%s", ulid.MakeSecure().String()),
				CreatedBy:   fixtureAdminUserID,
			}
		},
		Update: func(k *APIKey) {
			k.Description = sql.NullString{Valid: true, String: "Updated conformance key"}
		},
		FieldMap: map[string]string{
			"client_id": "ClientID",
		},
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan, tsuite.CRUDRoundTrip},
	})
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestAPIKeyScan verifies Scan behavior Shape conformance does not cover: List projection,
// nullable columns, and scanner error propagation.
func TestAPIKeyScan(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		// Setup: list projection omits secret and revoked columns.
		data := []any{
			ulid.MakeSecure().String(),
			"Test api keys for development",
			"XUiRZrNDUnLjeenQQmblpv",
			ulid.MakeSecure().String(),
			time.Now().Add(-1 * time.Hour),
			nil,
			time.Now().Add(-14 * time.Hour),
			time.Now().Add(-30 * time.Minute),
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan row using List shape.
		model := &APIKey{}
		err := model.Scan(tidal.List, mockScanner)
		require.NoError(t, err)
		mockScanner.AssertScanned(t, len(data))

		// Assert: omitted columns stay zero and nullable fields are handled.
		require.Equal(t, data[0], model.ID.String())
		require.Equal(t, data[1], model.Description.String)
		require.Equal(t, data[2], model.ClientID)
		require.Zero(t, model.Secret)
		require.Equal(t, data[3], model.CreatedBy.String())
		require.Equal(t, data[4], model.LastSeen.Time)
		require.False(t, model.Revoked.Valid)
		require.Equal(t, data[6], model.Created)
		require.Equal(t, data[7], model.Modified)
	})

	t.Run("Nulls", func(t *testing.T) {
		// Setup: nullable columns and zero modified timestamp.
		data := []any{
			ulid.MakeSecure().String(),
			nil,
			"XUiRZrNDUnLjeenQQmblpv",
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=",
			ulid.MakeSecure().String(),
			nil,
			nil,
			time.Now(),
			time.Time{},
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan nullable row.
		model := &APIKey{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.NoError(t, err)
		mockScanner.AssertScanned(t, len(data))

		// Assert: null SQL values produce invalid Null* fields and zero modified.
		require.False(t, model.Description.Valid)
		require.False(t, model.LastSeen.Valid)
		require.False(t, model.Revoked.Valid)
		require.True(t, model.Modified.IsZero())
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &APIKey{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.ErrorIs(t, err, ErrModelScan)
	})
}

// TestAPIKeyStatus verifies Status returns the correct enum for revoked and last-seen combinations.
func TestAPIKeyStatus(t *testing.T) {
	tests := []struct {
		key      *APIKey
		expected enum.APIKeyStatus
	}{
		{
			key: &APIKey{
				BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
				Revoked:   sql.NullTime{Valid: false},
				LastSeen:  sql.NullTime{Valid: false},
			},
			expected: enum.APIKeyStatusUnused,
		},
		{
			key: &APIKey{
				BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
				Revoked:   sql.NullTime{Valid: false},
				LastSeen:  sql.NullTime{Valid: true, Time: time.Now().Add(1492 * time.Hour)},
			},
			expected: enum.APIKeyStatusActive,
		},
		{
			key: &APIKey{
				BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
				Revoked:   sql.NullTime{Valid: false},
				LastSeen:  sql.NullTime{Valid: true, Time: time.Now().Add(-3138 * time.Hour)},
			},
			expected: enum.APIKeyStatusStale,
		},
		{
			key: &APIKey{
				BaseModel: tidal.BaseModel{ID: modelID, Created: created, Modified: modified},
				Revoked:   sql.NullTime{Valid: true, Time: time.Now().Add(-1 * time.Hour)},
				LastSeen:  sql.NullTime{Valid: true, Time: time.Now().Add(-1492 * time.Hour)},
			},
			expected: enum.APIKeyStatusRevoked,
		},
	}

	for i, tc := range tests {
		require.Equal(t, tc.expected, tc.key.Status(), "status test failed on test case %d", i)
	}
}
