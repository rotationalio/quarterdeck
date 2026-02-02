package models_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func TestOIDCClientParams(t *testing.T) {
	redirectURIs := []string{"https://example.com/callback", "https://www.example.com/callback"}
	contacts := []string{"first@example.com", "second@example.com"}

	client := &OIDCClient{
		Model: Model{
			ID:       modelID,
			Created:  created,
			Modified: modified,
		},
		ClientName:   "Test OIDC Client",
		ClientURI:    sql.NullString{Valid: true, String: "http://example.com"},
		LogoURI:      sql.NullString{Valid: true, String: "http://example.com/logo"},
		PolicyURI:    sql.NullString{Valid: true, String: "http://example.com/policy"},
		TOSURI:       sql.NullString{Valid: true, String: "http://example.com/tos"},
		Contacts:     []sql.NullString{{Valid: true, String: contacts[0]}, {Valid: true, String: contacts[1]}},
		RedirectURIs: redirectURIs,
		ClientID:     "XUiRZrNDUnLjeenQQmblpv",
		Secret:       "$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=",
		CreatedBy:    ulid.MakeSecure(),
		Revoked:      sql.NullTime{Valid: false},
	}

	redirectURIsJSON, _ := json.Marshal(redirectURIs)
	contactsJSON, _ := json.Marshal(contacts)

	CheckParams(t, client.Params(),
		[]string{
			"id", "clientName", "clientURI", "logoURI", "policyURI", "tosURI",
			"redirectURIs", "contacts", "clientID", "secret", "createdBy",
			"revoked", "created", "modified",
		},
		[]any{
			client.ID, client.ClientName, client.ClientURI, client.LogoURI, client.PolicyURI, client.TOSURI,
			string(redirectURIsJSON), string(contactsJSON), client.ClientID, client.Secret, client.CreatedBy,
			client.Revoked, client.Created, client.Modified,
		},
	)
}

func TestOIDCClientScan(t *testing.T) {
	t.Run("NotNull", func(t *testing.T) {
		redirectURIsJSON := `["https://example.com/callback","https://www.example.com/callback"]`
		contactsJSON := `["first@example.com","second@example.com"]`

		data := []any{
			ulid.MakeSecure().String(),  // ID
			"Test OIDC client",          // ClientName
			"https://example.com",       // ClientURI (driver returns string)
			"http://example.com/logo",   // LogoURI
			"http://example.com/policy", // PolicyURI
			"http://example.com/tos",    // TOSURI
			redirectURIsJSON,            // redirect_uris (driver returns string)
			contactsJSON,                // contacts (driver returns string)
			"XUiRZrNDUnLjeenQQmblpv",    // ClientID
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=", // Secret
			ulid.MakeSecure().String(),        // CreatedBy
			time.Now().Add(-30 * time.Minute), // Revoked
			time.Now().Add(-14 * time.Hour),   // Created
			time.Now().Add(-30 * time.Minute), // Modified
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &OIDCClient{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.ClientName, "expected field ClientName to match data[1]")
		require.True(t, model.ClientURI.Valid, "expected field ClientURI to be a valid string")
		require.Equal(t, "https://example.com", model.ClientURI.String, "expected field ClientURI to match")
		require.True(t, model.LogoURI.Valid, "expected field LogoURI to be a valid string")
		require.Equal(t, "http://example.com/logo", model.LogoURI.String, "expected field LogoURI to match")
		require.True(t, model.PolicyURI.Valid, "expected field PolicyURI to be a valid string")
		require.Equal(t, "http://example.com/policy", model.PolicyURI.String, "expected field PolicyURI to match")
		require.True(t, model.TOSURI.Valid, "expected field TOSURI to be a valid string")
		require.Equal(t, "http://example.com/tos", model.TOSURI.String, "expected field TOSURI to match")
		require.Equal(t, []string{"https://example.com/callback", "https://www.example.com/callback"}, model.RedirectURIs, "expected RedirectURI parsed from JSON")
		require.Len(t, model.Contacts, 2, "expected two contacts")
		require.True(t, model.Contacts[0].Valid, "expected contact email to be a valid string")
		require.Equal(t, "first@example.com", model.Contacts[0].String, "expected contact email to match")
		require.True(t, model.Contacts[1].Valid, "expected contact email to be a valid string")
		require.Equal(t, "second@example.com", model.Contacts[1].String, "expected contact email to match")
		require.Equal(t, data[8], model.ClientID, "expected field ClientID to match data[8]")
		require.Equal(t, data[9], model.Secret, "expected field Secret to match data[9]")
		require.Equal(t, data[10], model.CreatedBy.String(), "expected field CreatedBy to match data[10]")
		require.Equal(t, data[11], model.Revoked.Time, "expected field Revoked to match data[11]")
		require.Equal(t, data[12], model.Created, "expected field Created to match data[12]")
		require.Equal(t, data[13], model.Modified, "expected field Modified to match data[13]")
	})

	t.Run("Nulls", func(t *testing.T) {
		data := []any{
			ulid.MakeSecure().String(), // ID
			"",                         // ClientName (empty)
			nil,                        // ClientURI (driver returns nil)
			nil,                        // LogoURI
			nil,                        // PolicyURI
			nil,                        // TOSURI
			nil,                        // redirect_uris (null)
			nil,                        // contacts (null)
			"XUiRZrNDUnLjeenQQmblpv",   // ClientID
			"$argon2id$v=19$m=65536,t=1,p=2$GCSPNYPRVwBT9E559vqOnQ==$QMiOdjzXvvyNiQid3G7WY6E2zprY00UI4xJDCbd1HkM=", // Secret
			ulid.MakeSecure().String(), // CreatedBy
			nil,                        // Revoked
			time.Now(),                 // Created
			time.Time{},                // Modified (testing zero time)
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		model := &OIDCClient{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors when scanning")
		mockScanner.AssertScanned(t, len(data))

		require.Empty(t, model.ClientName, "expected ClientName empty")
		require.False(t, model.ClientURI.Valid, "expected ClientURI invalid (null)")
		require.Nil(t, model.RedirectURIs, "expected RedirectURI nil when JSON null")
		require.Nil(t, model.Contacts, "expected Contacts nil when JSON null")
		require.False(t, model.Revoked.Valid, "expected field Revoked to be invalid (null)")
		require.True(t, model.Modified.IsZero(), "expected field Modified to be zero time")
	})

	t.Run("Error", func(t *testing.T) {
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(ErrModelScan)

		model := &OIDCClient{}
		err := model.Scan(mockScanner)
		require.ErrorIs(t, err, ErrModelScan, "expected error when scanning with mock scanner")
	})
}

func TestOIDCClientScanSummary(t *testing.T) {
	redirectURIsJSON := `["https://example.com/callback","https://www.example.com/callback"]`
	contactsJSON := `["first@example.com","second@example.com"]`

	data := []any{
		ulid.MakeSecure().String(),        // ID
		"Test OIDC client",                // ClientName
		"https://example.com",             // ClientURI (driver returns string)
		"http://example.com/logo",         // LogoURI
		"http://example.com/policy",       // PolicyURI
		"http://example.com/tos",          // TOSURI
		redirectURIsJSON,                  // redirect_uris (driver returns string)
		contactsJSON,                      // contacts (driver returns string)
		"XUiRZrNDUnLjeenQQmblpv",          // ClientID
		ulid.MakeSecure().String(),        // CreatedBy
		time.Now().Add(-30 * time.Minute), // Revoked
		time.Now().Add(-14 * time.Hour),   // Created
		time.Now().Add(-30 * time.Minute), // Modified
	}
	mockScanner := &mock.Scanner{}
	mockScanner.SetData(data)

	model := &OIDCClient{}
	err := model.ScanSummary(mockScanner)
	require.NoError(t, err, "expected no errors when scanning")
	mockScanner.AssertScanned(t, len(data))

	require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
	require.Equal(t, data[1], model.ClientName, "expected field ClientName to match data[1]")
	require.True(t, model.ClientURI.Valid, "expected field ClientURI to be a valid string")
	require.Equal(t, "https://example.com", model.ClientURI.String, "expected field ClientURI to match")
	require.True(t, model.LogoURI.Valid, "expected field LogoURI to be a valid string")
	require.Equal(t, "http://example.com/logo", model.LogoURI.String, "expected field LogoURI to match")
	require.True(t, model.PolicyURI.Valid, "expected field PolicyURI to be a valid string")
	require.Equal(t, "http://example.com/policy", model.PolicyURI.String, "expected field PolicyURI to match")
	require.True(t, model.TOSURI.Valid, "expected field TOSURI to be a valid string")
	require.Equal(t, "http://example.com/tos", model.TOSURI.String, "expected field TOSURI to match")
	require.Equal(t, []string{"https://example.com/callback", "https://www.example.com/callback"}, model.RedirectURIs, "expected RedirectURI parsed from JSON")
	require.Len(t, model.Contacts, 2, "expected two contacts")
	require.True(t, model.Contacts[0].Valid, "expected contact email to be a valid string")
	require.Equal(t, "first@example.com", model.Contacts[0].String, "expected contact email to match")
	require.True(t, model.Contacts[1].Valid, "expected contact email to be a valid string")
	require.Equal(t, "second@example.com", model.Contacts[1].String, "expected contact email to match")
	require.Equal(t, data[8], model.ClientID, "expected field ClientID to match data[8]")
	require.Equal(t, "", model.Secret, "expected field Secret to be empty") // This is the only difference from Scan()
	require.Equal(t, data[9], model.CreatedBy.String(), "expected field CreatedBy to match data[9]")
	require.Equal(t, data[10], model.Revoked.Time, "expected field Revoked to match data[10]")
	require.Equal(t, data[11], model.Created, "expected field Created to match data[11]")
	require.Equal(t, data[12], model.Modified, "expected field Modified to match data[12]")
}

func TestOIDCClientValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		client := &OIDCClient{
			Model: Model{
				ID:       modelID,
				Created:  created,
				Modified: modified,
			},
			CreatedBy:    ulid.MakeSecure(),
			Revoked:      sql.NullTime{Valid: false},
			ClientName:   "Test OIDC Client",
			ClientURI:    sql.NullString{Valid: true, String: "http://example.com"},
			LogoURI:      sql.NullString{Valid: true, String: "http://example.com/logo"},
			PolicyURI:    sql.NullString{Valid: true, String: "http://example.com/policy"},
			TOSURI:       sql.NullString{Valid: true, String: "http://example.com/tos"},
			Contacts:     []sql.NullString{{Valid: true, String: "first@example.com"}, {Valid: true, String: "second@example.com"}},
			ClientID:     "XUiRZrNDUnLjeenQQmblpv",
			Secret:       "$argon2id$v=19$m=65536,t=1,p=2$Bk7GvOXGHdfDdSZH1OUyIA==$1AcYMKcJwm/DngmCw9db/J7PbvPzav/i/kk+Z0EKd44=",
			RedirectURIs: []string{"https://example.com/callback", "https://www.example.com/callback"},
		}
		require.NoError(t, client.Validate())
	})

	t.Run("ValidMinimal", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{"https://app.example.com/oauth/cb"},
		}
		require.NoError(t, client.Validate())
	})

	t.Run("InvalidRedirectURIsEmpty", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "redirect_uris")
		require.Contains(t, err.Error(), "at least one")
	})

	t.Run("InvalidRedirectURIsNil", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: nil,
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "redirect_uris")
	})

	t.Run("InvalidRedirectURINotURL", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{"not-a-url"},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "redirect_uris")
	})

	t.Run("InvalidRedirectURIHttp", func(t *testing.T) {
		gin.SetMode(gin.ReleaseMode)
		defer gin.SetMode(gin.TestMode)
		client := &OIDCClient{
			RedirectURIs: []string{"http://example.com/cb"},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "https")
	})

	t.Run("InvalidRedirectURILocalhost", func(t *testing.T) {
		gin.SetMode(gin.ReleaseMode)
		defer gin.SetMode(gin.TestMode)
		client := &OIDCClient{
			RedirectURIs: []string{"https://localhost/cb"},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "localhost")
	})

	t.Run("ValidDebugHttp", func(t *testing.T) {
		gin.SetMode(gin.DebugMode)
		defer gin.SetMode(gin.TestMode)
		client := &OIDCClient{
			RedirectURIs: []string{"http://example.com/cb"},
		}
		require.NoError(t, client.Validate())
	})

	t.Run("ValidDebugLocalhost", func(t *testing.T) {
		gin.SetMode(gin.DebugMode)
		defer gin.SetMode(gin.TestMode)
		client := &OIDCClient{
			RedirectURIs: []string{"https://localhost/cb", "http://127.0.0.1:3000/callback", "http://[::1]/cb"},
		}
		require.NoError(t, client.Validate())
	})

	t.Run("ValidDebugHttpLocalhost", func(t *testing.T) {
		gin.SetMode(gin.DebugMode)
		defer gin.SetMode(gin.TestMode)
		client := &OIDCClient{
			RedirectURIs: []string{"http://localhost:3000/oauth/callback"},
		}
		require.NoError(t, client.Validate())
	})

	t.Run("InvalidClientURI", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{"https://example.com/cb"},
			ClientURI:    sql.NullString{Valid: true, String: "://bad"},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "client_uri")
	})

	t.Run("InvalidLogoURI", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{"https://example.com/cb"},
			LogoURI:      sql.NullString{Valid: true, String: "not-a-valid-url"},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "logo_uri")
	})

	t.Run("InvalidContacts", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{"https://example.com/cb"},
			Contacts:     []sql.NullString{{Valid: true, String: "not-an-email"}},
		}
		err := client.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "contacts")
	})

	t.Run("MultipleErrors", func(t *testing.T) {
		client := &OIDCClient{
			RedirectURIs: []string{},
			Contacts:     []sql.NullString{{Valid: true, String: "bad"}},
		}
		err := client.Validate()
		require.Error(t, err)
		var joinErr interface{ Unwrap() []error }
		require.True(t, errors.As(err, &joinErr), "expected errors.Join result")
		require.GreaterOrEqual(t, len(joinErr.Unwrap()), 2, "expected at least two validation errors")
	})
}
