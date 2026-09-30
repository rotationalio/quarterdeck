package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	gimletauth "go.rtnl.ai/gimlet/auth"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/ulid"
)

// Allows browser preflight from the configured app origin and rejects preflight
// from an untrusted origin. This checks CORS, not CSRF: OPTIONS is a safe method.
func TestCSRFCORSPreflight(t *testing.T) {
	s := newCSRFCORSServer(t)
	for _, test := range []struct {
		name       string
		origin     string
		wantStatus int
	}{
		{
			name:       "approved origin",
			origin:     testBrowserOrigin,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unapproved origin",
			origin:     "https://attacker.example",
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "https://auth.endeavor.local/v1/users", nil)
			request.Header.Set("Origin", test.origin)
			request.Header.Set("Access-Control-Request-Method", "POST")
			request.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type,HX-Request,HX-Target,HX-Current-URL")
			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, request)

			require.Equal(t, test.wantStatus, recorder.Code)
			if test.wantStatus == http.StatusForbidden {
				require.Empty(t, recorder.Header().Get("Access-Control-Allow-Origin"))
				return
			}
			require.Equal(t, test.origin, recorder.Header().Get("Access-Control-Allow-Origin"))
			require.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
			allowed := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers"))
			require.Contains(t, allowed, "authorization")
			require.Contains(t, allowed, "hx-request")
		})
	}
}

// Rejects cross-site requests to every registered unsafe route using its actual
// HTTP method. Each must return a CSRF-specific 403 and expose the error header
// through CORS, including routes added to the application in the future.
func TestCSRFGlobalRoutes(t *testing.T) {
	s := newCSRFCORSServer(t)
	resourceID := ulid.Make().String()
	checked := 0
	for _, route := range s.router.Routes() {
		switch route.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			continue
		}
		checked++
		t.Run(route.Method+" "+route.Path, func(t *testing.T) {
			// Resolve Gin parameters so the request targets a concrete resource path.
			segments := strings.Split(route.Path, "/")
			for i, segment := range segments {
				if strings.HasPrefix(segment, ":") || strings.HasPrefix(segment, "*") {
					segments[i] = resourceID
				}
			}
			path := strings.Join(segments, "/")
			request := httptest.NewRequest(route.Method, "https://auth.endeavor.local"+path, nil)
			// Pass CORS so the rejection must come from CSRF, not the origin allowlist.
			request.Header.Set("Origin", testBrowserOrigin)
			request.Header.Set("Sec-Fetch-Site", "cross-site")
			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, csrf.ErrorRequestRejected, recorder.Header().Get(s.conf.CSRF.ErrorHeader()))
			require.Contains(t, strings.ToLower(recorder.Header().Get("Access-Control-Expose-Headers")), strings.ToLower(s.conf.CSRF.ErrorHeader()))
		})
	}
	require.NotZero(t, checked, "must exercise at least one unsafe route")
}

// Allows a same-site POST from the approved app origin to complete logout and
// return its normal login redirect, with no CSRF error and valid CORS headers.
func TestCSRFCORSApprovedSameSite(t *testing.T) {
	s := newCSRFCORSServer(t)
	request := httptest.NewRequest(http.MethodPost, "https://auth.endeavor.local/logout", nil)
	request.Header.Set("Origin", testBrowserOrigin)
	request.Header.Set("Sec-Fetch-Site", "same-site")
	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusSeeOther, recorder.Code)
	require.Equal(t, s.conf.Auth.LogoutRedirect, recorder.Header().Get("Location"))
	require.Empty(t, recorder.Header().Get(s.conf.CSRF.ErrorHeader()))
	require.Equal(t, testBrowserOrigin, recorder.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", recorder.Header().Get("Access-Control-Allow-Credentials"))
}

// Allows logout with a verified bearer token when site, Origin, and Referer are
// absent, but rejects the same token with explicit cross-site or none metadata.
func TestCSRFGlobalBearerFallback(t *testing.T) {
	s := newCSRFCORSServer(t)
	token, err := s.issuer.CreateAccessToken(&gimletauth.Claims{})
	require.NoError(t, err)
	signed, err := s.issuer.Sign(token)
	require.NoError(t, err)

	for _, test := range []struct {
		name       string
		site       string
		wantStatus int
	}{
		{
			name:       "missing metadata permits bearer fallback",
			wantStatus: http.StatusSeeOther,
		},
		{
			name:       "cross-site rejects valid bearer",
			site:       "cross-site",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "none rejects valid bearer",
			site:       "none",
			wantStatus: http.StatusForbidden,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "https://auth.endeavor.local/logout", nil)
			request.Header.Set("Authorization", "Bearer "+signed)
			if test.site != "" {
				request.Header.Set("Sec-Fetch-Site", test.site)
			}
			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, request)

			require.Equal(t, test.wantStatus, recorder.Code)
			if test.wantStatus == http.StatusForbidden {
				require.Equal(t, csrf.ErrorRequestRejected, recorder.Header().Get(s.conf.CSRF.ErrorHeader()))
			} else {
				require.Equal(t, s.conf.Auth.LogoutRedirect, recorder.Header().Get("Location"))
				require.Empty(t, recorder.Header().Get(s.conf.CSRF.ErrorHeader()))
			}
		})
	}
}

const testBrowserOrigin = "https://endeavor.local"

// Configures the real routes with a signing key and an app origin approved by
// both CORS and CSRF. Successful requests use logout, which does not need a store.
func newCSRFCORSServer(t *testing.T) *Server {
	t.Helper()
	conf := config.Config{
		AllowOrigins: []string{testBrowserOrigin},
		CSRF: config.CSRFConfig{
			Namespace:       "quarterdeck",
			ExpectedOrigins: []string{testBrowserOrigin},
		},
		Auth: config.AuthConfig{
			Issuer:          "https://auth.endeavor.local",
			Audience:        []string{testBrowserOrigin},
			LogoutRedirect:  "https://auth.endeavor.local/login",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: 2 * time.Hour,
			TokenOverlap:    -15 * time.Minute,
		},
		RateLimit: ratelimit.Config{
			Type:      ratelimit.TypeNone,
			PerSecond: 1,
			Burst:     1,
			CacheTTL:  time.Minute,
		},
	}
	issuer, err := auth.NewIssuer(conf.Auth)
	require.NoError(t, err)
	s := &Server{
		conf:   conf,
		issuer: issuer,
		router: gin.New(),
	}
	s.router.HandleMethodNotAllowed = true
	require.NoError(t, s.setupRoutes())
	return s
}
