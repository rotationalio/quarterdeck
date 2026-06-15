package models_test

import (
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

// TestVeroTokenCRUDConformance verifies VeroToken satisfies tidal CRUD shape and round-trip expectations against vero_tokens.
func (s *modelSuite) TestVeroTokenCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*VeroToken]{
		Table: "vero_tokens",
		Create: func() *VeroToken {
			return &VeroToken{
				TokenType:  enum.TokenTypeVerifyEmail,
				Email:      fmt.Sprintf("conformance-%s@example.com", ulid.MakeSecure().String()),
				Expiration: time.Now().Add(24 * time.Hour).UTC(),
			}
		},
		Update: func(v *VeroToken) {
			v.Email = "updated-conformance@example.com"
		},
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan, tsuite.CRUDRoundTrip},
	})
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestVeroScan verifies Scan maps signature bytes and propagates scanner errors.
// Nullable field round-trips are covered by CRUDScan conformance.
func TestVeroScan(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		// Setup: full vero_tokens row with signature and sent_on populated.
		data := []any{
			ulid.MakeSecure().String(),
			"verify_email",
			ulid.MakeSecure().String(),
			"gdavies@example.com",
			time.Now().Add(5 * time.Hour),
			[]byte{1, 151, 177, 135, 47, 223, 130, 157, 127, 242, 128, 127, 227, 101, 35, 22, 240, 132, 190, 169, 154, 183, 165, 173, 57, 118, 185, 132, 29, 203, 211, 168, 124, 90, 31, 213, 27, 248, 40, 121, 14, 77, 32, 181, 246, 249, 62, 29, 198, 177, 155, 26, 116, 10, 166, 157, 97, 246, 236, 118, 118, 115, 153, 231, 223, 15, 14, 116, 217, 70, 7, 173, 160, 173, 45, 105, 246, 206, 228, 34, 99, 182, 240, 208, 148, 77, 182, 85, 57, 207, 18, 56, 55, 248, 8, 136, 41, 191, 29, 132, 155, 17, 84, 169, 47, 240, 196, 8, 96, 122, 188, 6, 73, 116, 137, 102, 209, 42, 201, 110, 101},
			time.Now().Add(-1 * time.Hour),
			time.Now().Add(-1 * time.Hour),
			time.Now().Add(-1 * time.Hour),
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan row using Retrieve shape.
		model := &VeroToken{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.NoError(t, err)
		mockScanner.AssertScanned(t, len(data))

		// Assert: token type, email, and signature map correctly.
		require.Equal(t, data[0], model.ID.String())
		require.Equal(t, enum.TokenTypeVerifyEmail, model.TokenType)
		require.Equal(t, data[3], model.Email)
		require.NotNil(t, model.Signature)
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &VeroToken{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.ErrorIs(t, err, ErrModelScan)
	})
}

// TestVeroIsExpired verifies IsExpired treats past, future, and zero expiration correctly.
func TestVeroIsExpired(t *testing.T) {
	require.True(t, (&VeroToken{Expiration: time.Now().Add(-1 * time.Second)}).IsExpired())
	require.False(t, (&VeroToken{Expiration: time.Now().Add(1 * time.Second)}).IsExpired())
	require.True(t, (&VeroToken{Expiration: time.Time{}}).IsExpired(), "zero value expiration should be expired")
}
