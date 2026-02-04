package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/ulid"
)

func TestOIDCClientValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		o := validOIDCClient()
		require.NoError(t, o.Validate(true))
	})

	t.Run("IDNotZero", func(t *testing.T) {
		o := validOIDCClient()
		o.ID = ulid.MakeSecure()
		assertSingleValidationError(t, o.Validate(true), "read-only field id: this field cannot be written by the user", nil)
	})

	t.Run("ClientIDSet", func(t *testing.T) {
		o := validOIDCClient()
		o.ClientID = "some-client-id"
		assertSingleValidationError(t, o.Validate(true), "read-only field client_id: this field cannot be written by the user", nil)
	})

	t.Run("SecretSet", func(t *testing.T) {
		o := validOIDCClient()
		o.Secret = "some-secret"
		assertSingleValidationError(t, o.Validate(true), "read-only field secret: this field cannot be written by the user", nil)
	})

	t.Run("CreatedBySet", func(t *testing.T) {
		o := validOIDCClient()
		o.CreatedBy = ulid.MakeSecure()
		assertSingleValidationError(t, o.Validate(true), "read-only field created_by: this field cannot be written by the user", nil)
	})

	t.Run("CreatedSet", func(t *testing.T) {
		o := validOIDCClient()
		o.Created = time.Now()
		assertSingleValidationError(t, o.Validate(true), "read-only field created: this field cannot be written by the user", nil)
	})

	t.Run("ModifiedSet", func(t *testing.T) {
		o := validOIDCClient()
		o.Modified = time.Now()
		assertSingleValidationError(t, o.Validate(true), "read-only field modified: this field cannot be written by the user", nil)
	})

	t.Run("RevokedSet", func(t *testing.T) {
		o := validOIDCClient()
		revoked := time.Now()
		o.Revoked = &revoked
		assertSingleValidationError(t, o.Validate(true), "invalid field revoked: this field cannot be set on create", nil)
	})

	t.Run("RedirectURIsEmpty", func(t *testing.T) {
		o := validOIDCClient()
		o.RedirectURIs = nil
		assertSingleValidationError(t, o.Validate(true), "missing redirect_uris: this field is required", nil)
	})

	t.Run("RedirectURIsEmptySlice", func(t *testing.T) {
		o := validOIDCClient()
		o.RedirectURIs = []string{}
		assertSingleValidationError(t, o.Validate(true), "missing redirect_uris: this field is required", nil)
	})

	t.Run("RedirectURIEmptyString", func(t *testing.T) {
		o := validOIDCClient()
		o.RedirectURIs = []string{""}
		assertSingleValidationError(t, o.Validate(true), "", []string{"redirect_uris[0]", "cannot be empty"})
	})

	t.Run("RedirectURINotAbsolute", func(t *testing.T) {
		o := validOIDCClient()
		o.RedirectURIs = []string{"/relative"}
		assertSingleValidationError(t, o.Validate(true), "", []string{"redirect_uris[0]", "must be an absolute URL with scheme and host"})
	})

	t.Run("RedirectURIInvalid", func(t *testing.T) {
		o := validOIDCClient()
		o.RedirectURIs = []string{"://bad"}
		assertSingleValidationError(t, o.Validate(true), "", []string{"redirect_uris[0]"})
	})

	t.Run("ClientURIInvalid", func(t *testing.T) {
		o := validOIDCClient()
		invalid := "not-a-url"
		o.ClientURI = &invalid
		assertSingleValidationError(t, o.Validate(true), "", []string{"client_uri", "invalid field"})
	})

	t.Run("LogoURIInvalid", func(t *testing.T) {
		o := validOIDCClient()
		invalid := "/relative"
		o.LogoURI = &invalid
		assertSingleValidationError(t, o.Validate(true), "", []string{"logo_uri", "must be an absolute URL with scheme and host"})
	})

	t.Run("PolicyURIInvalid", func(t *testing.T) {
		o := validOIDCClient()
		invalid := "ftp://ftp.example.com"
		o.PolicyURI = &invalid
		assertSingleValidationError(t, o.Validate(true), "", []string{"policy_uri", "scheme must be http or https"})
	})

	t.Run("TOSURIInvalid", func(t *testing.T) {
		o := validOIDCClient()
		invalid := "no-scheme.com"
		o.TOSURI = &invalid
		assertSingleValidationError(t, o.Validate(true), "", []string{"tos_uri", "must be an absolute URL with scheme and host"})
	})

	t.Run("ContactInvalidEmail", func(t *testing.T) {
		o := validOIDCClient()
		o.Contacts = []string{"notanemail"}
		assertSingleValidationError(t, o.Validate(true), "", []string{"contacts[0]", "invalid field"})
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// validOIDCClient returns a full valid OIDCClient for create: all optional
// fields set, redirect_uris and contacts each have 2+ items; id, client_id,
// secret, created_by, created, modified zero/empty. Note that ClientID and
// Secret are set during the creation, so they are unset on a valid client.
func validOIDCClient() *api.OIDCClient {
	cu := "https://example.com"
	lu := "https://example.com/logo"
	pu := "https://example.com/policy"
	tu := "https://example.com/tos"
	return &api.OIDCClient{
		ClientName:   "Test Client",
		RedirectURIs: []string{"https://example.com/cb", "https://app.example.com/callback"},
		ClientURI:    &cu,
		LogoURI:      &lu,
		PolicyURI:    &pu,
		TOSURI:       &tu,
		Contacts:     []string{"admin@example.com", "dev@example.com"},
	}
}

// assertSingleValidationError asserts err is a ValidationErrors with exactly one
// entry. If wantExact is non-empty, the error message must match exactly;
// otherwise each substring in wantContains must be present in the message.
func assertSingleValidationError(t *testing.T, err error, wantExact string, wantContains []string) {
	var verr api.ValidationErrors
	require.ErrorAs(t, err, &verr)
	require.Len(t, verr, 1)

	msg := verr[0].Error()
	if wantExact != "" {
		require.Equal(t, wantExact, msg)
		return
	}

	for _, sub := range wantContains {
		require.Contains(t, msg, sub)
	}
}
