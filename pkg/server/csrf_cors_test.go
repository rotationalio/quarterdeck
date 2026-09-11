package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/csrf"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// Confirms the direct browser bootstrap contract and cookie preservation.
func TestCSRFCORSBootstrap(t *testing.T) {
	// Set up the production-equivalent router and perform the first safe
	// request that should issue the namespaced CSRF cookie pair.
	fixture := newCSRFCORSTestFixture(t)
	first := fixture.bootstrap(t)

	// Confirm the response is empty and that both cookies have the required
	// security attributes for the shared cross-subdomain deployment.
	require.Equal(t, http.StatusNoContent, first.Code)
	require.Empty(t, first.Body.Bytes())
	require.NotNil(t, fixture.token)
	require.NotNil(t, fixture.reference)
	require.False(t, fixture.token.HttpOnly)
	require.True(t, fixture.reference.HttpOnly)
	require.Equal(t, "/", fixture.token.Path)
	require.Equal(t, "/", fixture.reference.Path)
	require.True(t, fixture.token.Secure)
	require.True(t, fixture.reference.Secure)
	require.Equal(t, "example.com", fixture.token.Domain)
	require.Equal(t, "example.com", fixture.reference.Domain)
	require.Equal(t, http.SameSiteLaxMode, fixture.token.SameSite)
	require.Equal(t, http.SameSiteLaxMode, fixture.reference.SameSite)

	// Repeat the bootstrap request with the existing cookies to verify that a
	// valid pair is preserved rather than rotated unnecessarily.
	originalToken := fixture.token.Value
	originalReference := fixture.reference.Value
	second := fixture.bootstrap(t)

	// Confirm the second response remains empty and the cookie values are
	// unchanged.
	require.Equal(t, http.StatusNoContent, second.Code)
	require.Empty(t, second.Result().Cookies())
	require.Equal(t, originalToken, fixture.token.Value)
	require.Equal(t, originalReference, fixture.reference.Value)
}

// Confirms that preflight responses allow only the configured credentialed
// browser contract and use Gimlet's generated CSRF header name.
func TestCSRFCORSPreflight(t *testing.T) {
	// Set up the real CORS middleware with one explicitly allowed browser
	// origin and table-drive both the allowed and attacker cases.
	fixture := newCSRFCORSTestFixture(t)
	tests := []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{name: "allowed", origin: testBrowserOrigin, wantStatus: http.StatusNoContent},
		{name: "unconfigured sibling disallowed", origin: "https://other.example.com", wantStatus: http.StatusForbidden},
		{name: "attacker disallowed", origin: "https://attacker.example", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Build the browser preflight for a protected user mutation, including
			// the CSRF and HTMX headers used by the direct Endeavor client.
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodOptions, "https://auth.example.com/v1/users", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Access-Control-Request-Method", http.MethodPost)
			request.Header.Set("Access-Control-Request-Headers", strings.Join([]string{
				fixture.names.Header, config.CSRFRetryHeader, "HX-Request", "HX-Target", "HX-Current-URL",
			}, ","))
			// Execute the preflight through the actual router and check that only
			// the configured origin receives credentialed access.
			fixture.router.ServeHTTP(recorder, request)

			require.Equal(t, test.wantStatus, recorder.Code)
			if test.name == "allowed" {
				require.Equal(t, testBrowserOrigin, recorder.Header().Get("Access-Control-Allow-Origin"))
				require.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
				require.NotEqual(t, "*", recorder.Header().Get("Access-Control-Allow-Origin"))
				allowMethods := recorder.Header().Get("Access-Control-Allow-Methods")
				require.Contains(t, allowMethods, "POST")
				require.Contains(t, allowMethods, "PUT")
				require.Contains(t, allowMethods, "DELETE")
				allowHeaders := recorder.Header().Get("Access-Control-Allow-Headers")
				require.Contains(t, strings.ToLower(allowHeaders), strings.ToLower(fixture.names.Header))
				require.Contains(t, strings.ToLower(allowHeaders), strings.ToLower(config.CSRFRetryHeader))
				require.Contains(t, strings.ToLower(allowHeaders), "hx-request")
				require.Contains(t, strings.ToLower(allowHeaders), "hx-target")
				require.Contains(t, strings.ToLower(allowHeaders), "hx-current-url")
				require.NotContains(t, strings.ToLower(allowHeaders), "x-endeavor-csrf-token")
			} else {
				require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

// Confirms that a valid browser token reaches the normal route handling and
// that each missing or invalid CSRF component is rejected first.
func TestCSRFCORSMutations(t *testing.T) {
	// Set up the protected mutation route and obtain the cookie pair required
	// for the valid and invalid browser-request cases.
	fixture := newCSRFCORSTestFixture(t)
	fixture.bootstrap(t)
	require.NotNil(t, fixture.token)
	require.NotNil(t, fixture.reference)

	t.Run("valid token reaches handler", func(t *testing.T) {
		// Send a complete credentialed browser mutation. The fixture handler
		// deliberately returns Unauthorized so reaching it proves CSRF passed
		// without treating the token as authorization.
		recorder := fixture.mutation(testBrowserOrigin, fixture.token.Value, true, true)

		// Confirm normal authentication handling runs and no CSRF rejection is
		// reported first.
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.True(t, fixture.reached)
		require.NotEqual(t, csrf.ErrorTokenInvalid, recorder.Header().Get(fixture.names.ErrorHeader))
	})

	tests := []struct {
		name             string
		header           string
		includeToken     bool
		includeReference bool
	}{
		{name: "missing header", includeToken: true, includeReference: true},
		{name: "mismatched header", header: "wrong", includeToken: true, includeReference: true},
		{name: "missing token cookie", header: fixture.token.Value, includeReference: true},
		{name: "missing reference cookie", header: fixture.token.Value, includeToken: true},
		{name: "invalid token", header: "invalid", includeToken: true, includeReference: true},
		{name: "malformed token", header: "%<", includeToken: true, includeReference: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Vary one CSRF input at a time while keeping the request otherwise
			// equivalent to a browser mutation.
			fixture.reached = false
			recorder := fixture.mutation(testBrowserOrigin, test.header, test.includeToken, test.includeReference)

			// Every incomplete or invalid token combination must fail before the
			// mutation handler and expose only the generic CSRF failure signal.
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, csrf.ErrorTokenInvalid, recorder.Header().Get(fixture.names.ErrorHeader))
			require.False(t, fixture.reached)
		})
	}

	t.Run("valid token with disallowed origin", func(t *testing.T) {
		// Reuse valid CSRF credentials from an unconfigured origin to ensure
		// origin policy cannot be bypassed with a valid token.
		fixture.reached = false
		recorder := fixture.mutation("https://attacker.example", fixture.token.Value, true, true)

		// Confirm the request is rejected before the mutation handler and does
		// not receive a credentialed CORS response.
		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.False(t, fixture.reached)
		require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("valid non-browser request without origin", func(t *testing.T) {
		// Exercise the explicit non-browser policy with a valid cookie/header
		// pair but no Origin header.
		fixture.reached = false
		recorder := fixture.mutation("", fixture.token.Value, true, true)

		// Confirm non-browser requests may proceed to ordinary authentication
		// handling without weakening the browser-origin checks above.
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.True(t, fixture.reached)
	})
}

// The app uses the apex host while Quarterdeck uses its auth subdomain.
const testBrowserOrigin = "https://example.com"

// The fixture uses the production CORS middleware, namespaced Gimlet handler,
// bootstrap handler, and protected mutation middleware while replacing only
// authentication and persistence with observable test handlers.
type csrfCORSTestFixture struct {
	router    *gin.Engine
	names     csrf.Namespace
	token     *http.Cookie
	reference *http.Cookie
	reached   bool
}

// Creates a router with the same middleware contract used by the production
// server and a harmless protected mutation endpoint.
func newCSRFCORSTestFixture(t *testing.T) *csrfCORSTestFixture {
	t.Helper()
	conf := config.Config{
		AllowOrigins: []string{testBrowserOrigin},
		CSRF: config.CSRFConfig{
			CookieTTL:    5 * time.Minute,
			Secret:       "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5388ee9088f7ace2efcde9",
			Namespace:    "quarterdeck",
			CookieDomain: "example.com",
		},
	}

	handler, err := csrf.NewTokenHandlerWithNamespace(
		conf.CSRF.CookieTTL,
		"/",
		conf.CSRF.CookieDomains(),
		conf.CSRF.GetSecret(),
		conf.CSRF.Namespace,
	)
	require.NoError(t, err)

	fixture := &csrfCORSTestFixture{
		router: gin.New(),
		// HACK: gimlet/csrf has no namespace helper, so we create a throwaway
		// handler to get the names for now.
		// TODO: add `csrf.NewNamespace(namespace string) csrf.Namespace` to gimlet then use here.
		names: handler.(csrf.Namespacer).Namespace(),
	}
	server := &Server{conf: conf, csrf: &sameSiteCSRF{TokenHandler: handler}}
	fixture.router.Use(cors.New(conf.CORS()))
	fixture.router.GET("/csrf", server.CSRFToken)
	// csrfProtection(server.csrf) is the same csrf handler used by the real
	// server, so we use it here to test the actual CSRF protection logic.
	fixture.router.POST("/v1/users", csrfProtection(server.csrf), func(c *gin.Context) {
		// This stands in for the real authentication middleware. Reaching this
		// handler demonstrates that CSRF did not reject the request first.
		c.Set("authenticated", true)
		fixture.reached = true
		c.Status(http.StatusUnauthorized)
	})
	return fixture
}

// Sends the safe bootstrap request and retains the issued cookie pair for
// subsequent browser requests.
func (f *csrfCORSTestFixture) bootstrap(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "https://auth.example.com/csrf", nil)
	request.Header.Set("Origin", testBrowserOrigin)
	if f.token != nil {
		request.AddCookie(f.token)
	}
	if f.reference != nil {
		request.AddCookie(f.reference)
	}
	f.router.ServeHTTP(recorder, request)
	for _, cookie := range recorder.Result().Cookies() {
		switch cookie.Name {
		case f.names.Cookie:
			f.token = cookie
		case f.names.ReferenceCookie:
			f.reference = cookie
		}
	}
	return recorder
}

// Builds a browser mutation with the supplied CSRF and origin variations.
func (f *csrfCORSTestFixture) mutation(origin, header string, includeToken, includeReference bool) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://auth.example.com/v1/users", nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if header != "" {
		request.Header.Set(f.names.Header, header)
	}
	if includeToken && f.token != nil {
		request.AddCookie(f.token)
	}
	if includeReference && f.reference != nil {
		request.AddCookie(f.reference)
	}
	f.router.ServeHTTP(recorder, request)
	return recorder
}
