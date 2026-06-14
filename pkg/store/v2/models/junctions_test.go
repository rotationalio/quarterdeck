package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// Database Conformance Tests
//=============================================================================

// TestUserRoleCRUDConformance verifies UserRole tidal shape and scan against the live schema.
func (s *modelSuite) TestUserRoleCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*UserRole]{
		Table: "user_roles",
		Create: func() *UserRole {
			return &UserRole{
				UserID: ulid.MakeSecure(),
				RoleID: 1,
			}
		},
		Update: func(_ *UserRole) {},
		// CRUDRoundTrip skipped: This table uses a composite primary key (not a single id),
		// so inserting requires valid related user_id and role_id records. Junction linking is
		// already tested in db/roles_test.
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan},
	})
}

// TestRolePermissionCRUDConformance verifies RolePermission tidal shape and scan.
func (s *modelSuite) TestRolePermissionCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*RolePermission]{
		Table: "role_permissions",
		Create: func() *RolePermission {
			return &RolePermission{
				RoleID:       1,
				PermissionID: 1,
			}
		},
		Update: func(_ *RolePermission) {},
		// CRUDRoundTrip skipped: This table uses a composite primary key (not a single id),
		// so inserting requires valid related role_id and permission_id records. Junction linking is
		// already tested in db/roles_test.
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan},
	})
}

// TestAPIKeyPermissionCRUDConformance verifies APIKeyPermission tidal shape and scan.
func (s *modelSuite) TestAPIKeyPermissionCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*APIKeyPermission]{
		Table: "api_key_permissions",
		Create: func() *APIKeyPermission {
			return &APIKeyPermission{
				APIKeyID:     ulid.MakeSecure(),
				PermissionID: 1,
			}
		},
		Update: func(_ *APIKeyPermission) {},
		// CRUDRoundTrip skipped: This table uses a composite primary key (not a single id),
		// so inserting requires valid related api_key_id and permission_id records. Junction linking is
		// already tested in db/apikeys_test.
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan},
	})
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestJunctionPrepare verifies Prepare sets Created on create when not already set.
func TestJunctionPrepare(t *testing.T) {
	t.Run("UserRole", func(t *testing.T) {
		ur := &UserRole{UserID: ulid.MakeSecure(), RoleID: 1}
		ur.Prepare(tidal.Create)
		require.False(t, ur.Created.IsZero())
	})

	t.Run("RolePermission", func(t *testing.T) {
		rp := &RolePermission{RoleID: 1, PermissionID: 2}
		rp.Prepare(tidal.Create)
		require.False(t, rp.Created.IsZero())
	})

	t.Run("APIKeyPermission", func(t *testing.T) {
		ap := &APIKeyPermission{APIKeyID: ulid.MakeSecure(), PermissionID: 3}
		ap.Prepare(tidal.Create)
		require.False(t, ap.Created.IsZero())
	})

	t.Run("PreservesExistingCreated", func(t *testing.T) {
		stamp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		ur := &UserRole{Created: stamp}
		ur.Prepare(tidal.Create)
		require.Equal(t, stamp, ur.Created)
	})
}
