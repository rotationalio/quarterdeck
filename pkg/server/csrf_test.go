package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/auth"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/x/rlog"
)

// Checks default metadata and origin policies and verifies bearer credentials only when Gimlet reaches fallback.
func TestCSRFProtection(t *testing.T) {
	conf := config.CSRFConfig{
		Namespace:       "quarterdeck",
		ExpectedOrigins: []string{"https://app.example.com"},
	}
	tests := []struct {
		name, method, site, origin, referer, bearer, cookie string
		allowed                                             bool
		verifyBearer                                        bool
	}{
		{
			name:    "safe GET",
			method:  "GET",
			site:    "cross-site",
			allowed: true,
		},
		{
			name:    "safe HEAD",
			method:  "HEAD",
			allowed: true,
		},
		{
			name:    "safe OPTIONS",
			method:  "OPTIONS",
			allowed: true,
		},
		{
			name:    "same origin",
			site:    "same-origin",
			allowed: true,
		},
		{
			name:    "trusted same site",
			site:    "same-site",
			origin:  "https://app.example.com",
			allowed: true,
		},
		{
			name:   "untrusted sibling",
			site:   "same-site",
			origin: "https://other.example.com",
		},
		{
			name:   "cross site",
			site:   "cross-site",
			origin: "https://app.example.com",
		},
		{
			name: "none",
			site: "none",
		},
		{
			name: "missing metadata",
		},
		{
			name: "unknown site",
			site: "future-site",
		},
		{
			name:    "trusted origin fallback",
			origin:  "https://app.example.com",
			allowed: true,
		},
		{
			name:    "trusted referer fallback",
			referer: "https://app.example.com/path",
			allowed: true,
		},
		{
			name:    "untrusted origin blocks trusted referer",
			origin:  "https://evil.example",
			referer: "https://app.example.com/path",
		},
		{
			name:         "verified bearer fallback",
			bearer:       "Bearer valid",
			allowed:      true,
			verifyBearer: true,
		},
		{
			name:   "bearer cannot override cross site",
			site:   "cross-site",
			bearer: "Bearer valid",
		},
		{
			name:   "bearer cannot override none",
			site:   "none",
			bearer: "Bearer valid",
		},
		{
			name:   "bearer cannot override unknown site",
			site:   "future-site",
			bearer: "Bearer valid",
		},
		{
			name:   "bearer cannot override untrusted sibling",
			site:   "same-site",
			origin: "https://other.example.com",
			bearer: "Bearer valid",
		},
		{
			name:   "bearer cannot override untrusted origin",
			origin: "https://evil.example",
			bearer: "Bearer valid",
		},
		{
			name:    "bearer cannot override untrusted referer",
			referer: "https://evil.example/path",
			bearer:  "Bearer valid",
		},
		{
			name:    "same origin does not need bearer fallback",
			site:    "same-origin",
			bearer:  "Bearer valid",
			allowed: true,
		},
		{
			name:    "trusted origin does not need bearer fallback",
			origin:  "https://app.example.com",
			bearer:  "Bearer valid",
			allowed: true,
		},
		{
			name:         "invalid bearer",
			bearer:       "Bearer invalid",
			verifyBearer: true,
		},
		{
			name:   "malformed bearer",
			bearer: "Bearer",
		},
		{
			name:   "cookie cannot bypass",
			cookie: "valid",
		},
		{
			name:         "invalid bearer cannot use cookie to bypass",
			bearer:       "Bearer invalid",
			cookie:       "valid",
			verifyBearer: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			method := test.method
			if method == "" {
				method = http.MethodPost
			}
			router := gin.New()
			issuer := &csrfTestIssuer{}
			router.Use(csrfProtection(conf, issuer))
			reached := false
			router.Handle(method, "/action", func(c *gin.Context) {
				reached = true
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(method, "https://auth.example.com/action", nil)
			for name, value := range map[string]string{"Sec-Fetch-Site": test.site, "Origin": test.origin, "Referer": test.referer, "Authorization": test.bearer} {
				if value != "" {
					request.Header.Set(name, value)
				}
			}
			if test.cookie != "" {
				request.AddCookie(&http.Cookie{Name: auth.AccessTokenCookie, Value: test.cookie})
				request.AddCookie(&http.Cookie{Name: auth.RefreshTokenCookie, Value: test.cookie})
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, test.allowed, reached)
			require.Equal(t, test.verifyBearer, issuer.calls > 0, "bearer verification must only run during fallback")
			require.Empty(t, recorder.Result().Cookies())
			if test.allowed {
				require.Equal(t, http.StatusNoContent, recorder.Code)
				require.Empty(t, recorder.Header().Get(conf.ErrorHeader()))
			} else {
				require.Equal(t, http.StatusForbidden, recorder.Code)
				require.Equal(t, csrf.ErrorRequestRejected, recorder.Header().Get(conf.ErrorHeader()))
			}
		})
	}
}

// Ensures disabling enforcement logs would-be rejections without blocking or setting an error header.
func TestCSRFDisabledLogsRejection(t *testing.T) {
	original := *rlog.Default()
	originalLevel := rlog.Level()
	var logs bytes.Buffer
	rlog.SetDefault(rlog.New(slog.New(slog.NewJSONHandler(&logs, nil))))
	rlog.SetLevel(slog.LevelDebug)
	t.Cleanup(func() {
		rlog.SetDefault(&original)
		rlog.SetLevel(originalLevel)
	})
	router := gin.New()
	conf := config.CSRFConfig{
		Disabled:  true,
		Namespace: "quarterdeck",
	}
	router.Use(csrfProtection(conf, nil))
	router.POST("/action", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Header().Get(conf.ErrorHeader()))
	require.Contains(t, logs.String(), "CSRF request would be rejected")
	require.Contains(t, logs.String(), `"reason":"cross_site"`)
}

// Exercises policy options and ensures bearer fallback cannot override fetch context restrictions.
func TestCSRFPolicyOptions(t *testing.T) {
	tests := []struct {
		name                                    string
		conf                                    config.CSRFConfig
		method, site, mode, destination, bearer string
		allowed                                 bool
	}{
		{
			name: "allow missing",
			conf: config.CSRFConfig{
				AllowMissingMetadata: true,
			},
			allowed: true,
		},
		{
			name: "allow unknown",
			conf: config.CSRFConfig{
				AllowUnknownSite: true,
			},
			site:    "future-site",
			allowed: true,
		},
		{
			name: "allow none",
			conf: config.CSRFConfig{
				AllowSiteNone: true,
			},
			site:    "none",
			allowed: true,
		},
		{
			name: "restrict mode",
			conf: config.CSRFConfig{
				AllowedFetchModes: []string{"cors"},
			},
			site: "same-origin",
			mode: "navigate",
		},
		{
			name: "allowed mode",
			conf: config.CSRFConfig{
				AllowedFetchModes: []string{"cors"},
			},
			site:    "same-origin",
			mode:    "cors",
			allowed: true,
		},
		{
			name: "restrict destination",
			conf: config.CSRFConfig{
				AllowedFetchDestinations: []string{"empty"},
			},
			site:        "same-origin",
			destination: "document",
		},
		{
			name: "allowed destination",
			conf: config.CSRFConfig{
				AllowedFetchDestinations: []string{"empty"},
			},
			site:        "same-origin",
			destination: "empty",
			allowed:     true,
		},
		{
			name: "require mode",
			conf: config.CSRFConfig{
				RequireFetchMode: true,
			},
			site: "same-origin",
		},
		{
			name: "require destination",
			conf: config.CSRFConfig{
				RequireFetchDestination: true,
			},
			site: "same-origin",
		},
		{
			name: "bearer cannot override required context",
			conf: config.CSRFConfig{
				RequireFetchMode:        true,
				RequireFetchDestination: true,
			},
			bearer: "Bearer valid",
		},
		{
			name: "bearer cannot override restricted mode",
			conf: config.CSRFConfig{
				AllowedFetchModes: []string{"cors"},
			},
			mode:   "navigate",
			bearer: "Bearer valid",
		},
		{
			name: "bearer cannot override restricted destination",
			conf: config.CSRFConfig{
				AllowedFetchDestinations: []string{"empty"},
			},
			destination: "document",
			bearer:      "Bearer valid",
		},
		{
			name: "bearer fallback with allowed context",
			conf: config.CSRFConfig{
				RequireFetchMode:         true,
				RequireFetchDestination:  true,
				AllowedFetchModes:        []string{"cors"},
				AllowedFetchDestinations: []string{"empty"},
			},
			mode:        "cors",
			destination: "empty",
			bearer:      "Bearer valid",
			allowed:     true,
		},
		{
			name: "narrow safe methods",
			conf: config.CSRFConfig{
				SafeHTTPMethods: []string{"GET"},
			},
			method: "HEAD",
		},
		{
			name: "no safe methods",
			conf: config.CSRFConfig{
				SafeHTTPMethods: []string{},
			},
			method: "GET",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			method := test.method
			if method == "" {
				method = http.MethodPost
			}
			router := gin.New()
			router.Use(csrfProtection(test.conf, &csrfTestIssuer{}))
			router.Handle(method, "/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(method, "/", nil)
			for name, value := range map[string]string{"Sec-Fetch-Site": test.site, "Sec-Fetch-Mode": test.mode, "Sec-Fetch-Dest": test.destination, "Authorization": test.bearer} {
				if value != "" {
					request.Header.Set(name, value)
				}
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if test.allowed {
				require.Equal(t, http.StatusNoContent, recorder.Code)
			} else {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			}
		})
	}
}

type csrfTestIssuer struct {
	calls int
}

// Accepts only the designated valid test token and rejects all other credentials.
func (issuer *csrfTestIssuer) Verify(token string) (*auth.Claims, error) {
	issuer.calls++
	if token == "valid" {
		return &auth.Claims{}, nil
	}
	return nil, errors.New("invalid or expired token")
}
