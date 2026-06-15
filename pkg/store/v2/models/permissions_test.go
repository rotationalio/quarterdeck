package models_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// Database Conformance Tests
//=============================================================================

// TestPermissionCRUDConformance verifies Permission satisfies tidal CRUD shape expectations against the permissions table.
func (s *modelSuite) TestPermissionCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*Permission]{
		Table: "permissions",
		Create: func() *Permission {
			return &Permission{
				Title:       fmt.Sprintf("conformance:%s", ulid.MakeSecure().String()),
				Description: "Conformance permission",
			}
		},
		Update: func(p *Permission) {
			p.Description = "Updated conformance permission"
		},
		// CRUDScan skipped: same Create Fields/Scan id mismatch as Role. TestPermissionScan covers row mapping.
		// CRUDRoundTrip skipped: tidal expects id in Params(Create), but serial ids are assigned by the DB.
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape},
	})
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestPermissionScan verifies Scan maps database rows and propagates scanner errors.
func TestPermissionScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Setup: full permissions row.
		data := []any{
			int64(120),
			"dashboard:read",
			"Read access to the dashboard",
			created,
			modified,
		}

		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan row using Retrieve shape.
		permission := &Permission{}
		err := permission.Scan(tidal.Retrieve, mockScanner)
		require.NoError(t, err)

		// Assert: every column maps to the corresponding model field.
		require.Equal(t, int64(120), permission.ID)
		require.Equal(t, "dashboard:read", permission.Title)
		require.Equal(t, "Read access to the dashboard", permission.Description)
		require.Equal(t, created, permission.Created)
		require.Equal(t, modified, permission.Modified)
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		permission := &Permission{}
		err := permission.Scan(tidal.Retrieve, mockScanner)
		require.ErrorIs(t, err, ErrModelScan)
	})
}
