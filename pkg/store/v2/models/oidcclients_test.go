package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/fields"
	tsuite "go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// Database Conformance Tests
//=============================================================================

// TestOIDCClientCRUDConformance verifies OIDCClient satisfies tidal CRUD shape expectations against the oidc_clients table.
func (s *modelSuite) TestOIDCClientCRUDConformance() {
	tsuite.ConformsCRUD(&s.DatabaseSuite, tsuite.CRUDConformance[*OIDCClient]{
		Table: "oidc_clients",
		Create: func() *OIDCClient {
			return &OIDCClient{
				ClientName:   "Conformance OIDC Client",
				RedirectURIs: fields.StringArray{"https://example.com/callback"},
				Contacts:     fields.StringArray{"ops@example.com"},
				ClientID:     fmt.Sprintf("oidc-%s", ulid.MakeSecure().String()),
				Secret:       fmt.Sprintf("secret-%s", ulid.MakeSecure().String()),
				CreatedBy:    fixtureAdminUserID,
			}
		},
		Update: func(c *OIDCClient) {
			c.ClientName = "Updated Conformance OIDC Client"
		},
		Equal: equalOIDCClientConformance,
		FieldMap: map[string]string{
			"client_uri":    "ClientURI",
			"logo_uri":      "LogoURI",
			"policy_uri":    "PolicyURI",
			"tos_uri":       "TOSURI",
			"redirect_uris": "RedirectURIs",
			"client_id":     "ClientID",
		},
		Phases: []tsuite.CRUDPhase{tsuite.CRUDShape, tsuite.CRUDScan, tsuite.CRUDRoundTrip},
	})
}

// equalOIDCClientConformance compares OIDC clients for tidal conformance runs,
// accounting for list projection differences and timestamp precision.
func equalOIDCClientConformance(a, b *OIDCClient) bool {
	if a == nil || b == nil {
		return a == b
	}

	if a.ID != b.ID || a.ClientName != b.ClientName {
		return false
	}
	if a.ClientURI != b.ClientURI || a.LogoURI != b.LogoURI || a.PolicyURI != b.PolicyURI || a.TOSURI != b.TOSURI {
		return false
	}
	if !a.RedirectURIs.Equal(b.RedirectURIs) || !a.Contacts.Equal(b.Contacts) {
		return false
	}
	if a.ClientID != b.ClientID || a.CreatedBy != b.CreatedBy {
		return false
	}

	// List projection omits secret; require equality only when both sides include it.
	if a.Secret != "" && b.Secret != "" && a.Secret != b.Secret {
		return false
	}

	return timeEqual(a.Created, b.Created) && timeEqual(a.Modified, b.Modified)
}

// timeEqual normalizes UTC location and second precision for DB round-trip checks.
func timeEqual(a, b time.Time) bool {
	return a.UTC().Truncate(time.Second).Equal(b.UTC().Truncate(time.Second))
}

//=============================================================================
// Unit Tests
//=============================================================================

// TestOIDCClientScan verifies Scan behavior Shape conformance does not cover: List projection
// (omitted secret) and scanner error propagation.
func TestOIDCClientScan(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		// Setup: list projection omits secret column.
		redirectURIsJSON := `["https://example.com/callback"]`
		contactsJSON := `["first@example.com"]`

		data := []any{
			ulid.MakeSecure().String(),
			"Test OIDC client",
			"https://example.com",
			nil,
			nil,
			nil,
			redirectURIsJSON,
			contactsJSON,
			"XUiRZrNDUnLjeenQQmblpv",
			ulid.MakeSecure().String(),
			time.Now().Add(-14 * time.Hour),
			time.Now().Add(-30 * time.Minute),
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Action: scan row using List shape.
		model := &OIDCClient{}
		err := model.Scan(tidal.List, mockScanner)
		require.NoError(t, err)
		mockScanner.AssertScanned(t, len(data))

		// Assert: secret stays zero and client_id maps correctly.
		require.Zero(t, model.Secret)
		require.Equal(t, data[8], model.ClientID)
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &OIDCClient{}
		err := model.Scan(tidal.Retrieve, mockScanner)
		require.ErrorIs(t, err, ErrModelScan)
	})
}
