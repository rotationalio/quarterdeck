package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/csrf"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// A browser-compatible POST with the valid CSRF double-submit pair must clear
// both auth cookies and preserve the existing redirect behavior.
func TestLogoutRoutePOST(t *testing.T) {
	s := newLogoutServer(t)
	names := s.csrf.(csrf.Namespacer).Namespace()

	// Bootstrap the CSRF cookies through the same endpoint used by browser clients.
	bootstrapRequest := httptest.NewRequest(http.MethodGet, "https://quarterdeck.example.com/csrf", nil)
	bootstrapResponse := httptest.NewRecorder()
	s.router.ServeHTTP(bootstrapResponse, bootstrapRequest)
	require.Equal(t, http.StatusNoContent, bootstrapResponse.Code)

	var tokenCookie, referenceCookie *http.Cookie
	for _, cookie := range bootstrapResponse.Result().Cookies() {
		switch cookie.Name {
		case names.Cookie:
			tokenCookie = cookie
		case names.ReferenceCookie:
			referenceCookie = cookie
		}
	}
	require.NotNil(t, tokenCookie)
	require.NotNil(t, referenceCookie)

	logoutRequest := httptest.NewRequest(http.MethodPost, "https://quarterdeck.example.com/logout", nil)
	logoutRequest.AddCookie(tokenCookie)
	logoutRequest.AddCookie(referenceCookie)
	logoutRequest.Header.Set(names.Header, tokenCookie.Value)
	logoutResponse := httptest.NewRecorder()
	s.router.ServeHTTP(logoutResponse, logoutRequest)

	// Browser navigation must receive the existing redirect status and target.
	require.Equal(t, http.StatusSeeOther, logoutResponse.Code)
	require.Equal(t, "https://quarterdeck.example.com/login", logoutResponse.Header().Get("Location"))

	// Logout must expire the secure, HTTP-only authentication cookies.
	cookies := logoutResponse.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, cookie := range cookies {
		require.Contains(t, []string{auth.AccessTokenCookie, auth.RefreshTokenCookie}, cookie.Name)
		require.Equal(t, "app.example.com", cookie.Domain)
		require.Equal(t, -1, cookie.MaxAge)
		require.True(t, cookie.HttpOnly)
		require.True(t, cookie.Secure)
	}
}

// Logout is a state-changing operation and must not be available through an
// unprotected browser GET request.
func TestLogoutRouteGETNotAllowed(t *testing.T) {
	s := newLogoutServer(t)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	resp := httptest.NewRecorder()
	s.router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusMethodNotAllowed, resp.Code)
}

// Initializes the real route table with only the test-specific auth and redirect
// configuration needed by the logout assertions.
func newLogoutServer(t *testing.T) *Server {
	t.Helper()

	handler, err := csrf.NewTokenHandlerWithNamespace(
		5*time.Minute, "/", nil, make([]byte, 32), "quarterdeck",
	)
	require.NoError(t, err)

	conf, err := config.Get()
	require.NoError(t, err)
	conf.Auth.Audience = []string{"https://app.example.com"}
	conf.Auth.LogoutRedirect = "https://quarterdeck.example.com/login"
	conf.CSRF.Disabled = false
	conf.RateLimit = ratelimit.Config{
		Type:      ratelimit.TypeNone,
		PerSecond: 1,
		Burst:     1,
		CacheTTL:  time.Minute,
	}

	router := gin.New()
	router.HandleMethodNotAllowed = true
	s := &Server{
		conf:   conf,
		router: router,
		csrf:   &sameSiteCSRF{TokenHandler: handler},
	}
	require.NoError(t, s.setupRoutes())
	return s
}
