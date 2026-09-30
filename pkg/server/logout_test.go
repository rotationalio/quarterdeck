package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// Same-origin and approved same-site browser POSTs must clear auth cookies and preserve redirects.
func TestLogoutRoutePOST(t *testing.T) {
	for _, test := range []struct {
		site   string
		origin string
	}{
		{
			site: "same-origin",
		},
		{
			site:   "same-site",
			origin: "https://app.example.com",
		},
	} {
		t.Run(test.site, func(t *testing.T) {
			s := newLogoutServer(t)
			logoutRequest := httptest.NewRequest(http.MethodPost, "https://quarterdeck.example.com/logout", nil)
			logoutRequest.Header.Set(csrf.HeaderSecFetchSite, test.site)
			if test.origin != "" {
				logoutRequest.Header.Set(csrf.HeaderOrigin, test.origin)
			}
			logoutResponse := httptest.NewRecorder()
			s.router.ServeHTTP(logoutResponse, logoutRequest)

			// Browser navigation must receive the existing redirect status and target.
			require.Equal(t, http.StatusSeeOther, logoutResponse.Code)
			require.Equal(t, "https://quarterdeck.example.com/login", logoutResponse.Header().Get("Location"))
			require.Empty(t, logoutResponse.Header().Get(s.conf.CSRF.ErrorHeader()))

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
		})
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

	conf, err := config.Get()
	require.NoError(t, err)
	conf.Auth.Audience = []string{"https://app.example.com"}
	conf.Auth.LogoutRedirect = "https://quarterdeck.example.com/login"
	conf.CSRF.Disabled = false
	conf.CSRF.ExpectedOrigins = []string{"https://app.example.com"}
	conf.AllowOrigins = []string{"https://app.example.com"}
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
	}
	require.NoError(t, s.setupRoutes())
	return s
}
