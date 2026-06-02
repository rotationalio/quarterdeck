package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
)

func TestAPIKeyStatus(t *testing.T) {
	tests := []struct {
		key      *APIKey
		expected enum.APIKeyStatus
	}{
		{
			key: &APIKey{
				BaseModel: BaseModel{
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
				BaseModel: BaseModel{
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
				BaseModel: BaseModel{
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
				BaseModel: BaseModel{
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
