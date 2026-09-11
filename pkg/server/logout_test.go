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

// The normal browser logout flow used by Endeavor's anchor link must clear both
// auth cookies and redirect to login.
func TestLogoutRouteGET(t *testing.T) {
	s := newLogoutServer(t)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	resp := httptest.NewRecorder()
	s.router.ServeHTTP(resp, req)

	// Browser navigation must receive the existing redirect status and target.
	require.Equal(t, http.StatusSeeOther, resp.Code)
	require.Equal(t, "https://quarterdeck.example.com/login", resp.Header().Get("Location"))

	// Logout must expire the same secure, HTTP-only cookies as the POST flow.
	cookies := resp.Result().Cookies()
	require.Len(t, cookies, 2)
	for _, cookie := range cookies {
		require.Contains(t, []string{auth.AccessTokenCookie, auth.RefreshTokenCookie}, cookie.Name)
		require.Equal(t, "app.example.com", cookie.Domain)
		require.Equal(t, -1, cookie.MaxAge)
		require.True(t, cookie.HttpOnly)
		require.True(t, cookie.Secure)
	}
}

// Browser-compatible GET logout must not weaken the existing CSRF protection
// for POST /logout.
func TestLogoutRoutePOSTRequiresCSRF(t *testing.T) {
	s := newLogoutServer(t)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	resp := httptest.NewRecorder()
	s.router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code)
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
	conf.RateLimit = ratelimit.Config{
		Type:      ratelimit.TypeNone,
		PerSecond: 1,
		Burst:     1,
		CacheTTL:  time.Minute,
	}

	s := &Server{
		conf:   conf,
		router: gin.New(),
		csrf:   &sameSiteCSRF{TokenHandler: handler},
	}
	require.NoError(t, s.setupRoutes())
	return s
}
