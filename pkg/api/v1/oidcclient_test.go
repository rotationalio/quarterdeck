package api_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/ulid"
)

func TestOIDCClientValidate_Valid(t *testing.T) {
	// setup
	o := validOIDCClient()

	// test
	require.NoError(t, o.Validate())
}

func TestOIDCClientValidate_IDNotZero(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.ID = ulid.MakeSecure()

	// test
	assertSingleValidationError(t, o.Validate(), "read-only field id: this field cannot be written by the user", nil)
}

func TestOIDCClientValidate_ClientIDSet(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.ClientID = "some-client-id"

	// test
	assertSingleValidationError(t, o.Validate(), "read-only field client_id: this field cannot be written by the user", nil)
}

func TestOIDCClientValidate_SecretSet(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.Secret = "some-secret"

	// test
	assertSingleValidationError(t, o.Validate(), "read-only field secret: this field cannot be written by the user", nil)
}

func TestOIDCClientValidate_CreatedBySet(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.CreatedBy = ulid.MakeSecure()

	// test
	assertSingleValidationError(t, o.Validate(), "read-only field created_by: this field cannot be written by the user", nil)
}

func TestOIDCClientValidate_CreatedSet(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.Created = time.Now()

	// test
	assertSingleValidationError(t, o.Validate(), "read-only field created: this field cannot be written by the user", nil)
}

func TestOIDCClientValidate_ModifiedSet(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.Modified = time.Now()

	// test
	assertSingleValidationError(t, o.Validate(), "read-only field modified: this field cannot be written by the user", nil)
}

func TestOIDCClientValidate_RevokedSet(t *testing.T) {
	// setup
	o := validOIDCClient()
	revoked := time.Now()
	o.Revoked = &revoked

	// test
	assertSingleValidationError(t, o.Validate(), "invalid field revoked: this field cannot be set on create", nil)
}

func TestOIDCClientValidate_RedirectURIsEmpty(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.RedirectURIs = nil

	// test
	assertSingleValidationError(t, o.Validate(), "missing redirect_uris: this field is required", nil)
}

func TestOIDCClientValidate_RedirectURIsEmptySlice(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.RedirectURIs = []string{}

	// test
	assertSingleValidationError(t, o.Validate(), "missing redirect_uris: this field is required", nil)
}

func TestOIDCClientValidate_RedirectURIEmptyString(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.RedirectURIs = []string{""}

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"redirect_uris[0]", "cannot be empty"})
}

func TestOIDCClientValidate_RedirectURINotAbsolute(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.RedirectURIs = []string{"/relative"}

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"redirect_uris[0]", "must be an absolute URL with scheme and host"})
}

func TestOIDCClientValidate_RedirectURIInvalid(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.RedirectURIs = []string{"://bad"}

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"redirect_uris[0]"})
}

func TestOIDCClientValidate_ClientURIInvalid(t *testing.T) {
	// setup
	o := validOIDCClient()
	invalid := "not-a-url"
	o.ClientURI = &invalid

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"client_uri", "invalid field"})
}

func TestOIDCClientValidate_LogoURIInvalid(t *testing.T) {
	// setup
	o := validOIDCClient()
	invalid := "/relative"
	o.LogoURI = &invalid

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"logo_uri", "must be an absolute URL with scheme and host"})
}

func TestOIDCClientValidate_PolicyURIInvalid(t *testing.T) {
	// setup
	o := validOIDCClient()
	invalid := "ftp://ftp.example.com"
	o.PolicyURI = &invalid

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"policy_uri", "scheme must be http or https"})
}

func TestOIDCClientValidate_TOSURIInvalid(t *testing.T) {
	// setup
	o := validOIDCClient()
	invalid := "no-scheme.com"
	o.TOSURI = &invalid

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"tos_uri", "must be an absolute URL with scheme and host"})
}

func TestOIDCClientValidate_ContactInvalidEmail(t *testing.T) {
	// setup
	o := validOIDCClient()
	o.Contacts = []string{"notanemail"}

	// test
	assertSingleValidationError(t, o.Validate(), "", []string{"contacts[0]", "invalid field"})
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
