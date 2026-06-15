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

// TestRoleCRUDConformance verifies Role satisfies tidal CRUD shape expectations against the roles table.
func (s *modelSuite) TestRoleCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*Role]{
		Table: "roles",
		Create: func() *Role {
			return &Role{
				Title:       fmt.Sprintf("conformance-role-%s", ulid.MakeSecure().String()),
				Description: "Conformance role",
			}
		},
		Update: func(r *Role) {
			r.Description = "Updated conformance role"
		},
		// CRUDScan skipped: Scan always reads id but Fields(Create) omits it (BIGSERIAL is DB-assigned),
		// so tidal's generic Scan phase fails on Create. TestRoleScan covers row mapping and errors.
		//
		// CRUDRoundTrip skipped: tidal looks up id in Params(Create), but serial ids are assigned by the DB.
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape},
	})
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestRoleScan verifies Scan maps database rows and propagates scanner errors.
func TestRoleScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Setup: full roles row.
		data := []any{
			int64(19),
			"Observer",
			"Read only access",
			false,
			created,
			modified,
		}

		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan row using Retrieve shape.
		role := &Role{}
		err := role.Scan(tidal.Retrieve, mockScanner)
		require.NoError(t, err)

		// Assert: every column maps to the corresponding model field.
		require.Equal(t, int64(19), role.ID)
		require.Equal(t, "Observer", role.Title)
		require.Equal(t, "Read only access", role.Description)
		require.False(t, role.IsDefault)
		require.Equal(t, created, role.Created)
		require.Equal(t, modified, role.Modified)
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		role := &Role{}
		err := role.Scan(tidal.Retrieve, mockScanner)
		require.ErrorIs(t, err, ErrModelScan)
	})
}
