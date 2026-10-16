package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.rtnl.ai/gimlet/auth"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	quarterdeckauth "go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/store/v1/mock"
	"go.rtnl.ai/quarterdeck/pkg/store/v1/models"
	"go.rtnl.ai/ulid"
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
			name:    "untrusted origin blocks trusted referer before bearer fallback",
			origin:  "https://evil.example",
			referer: "https://app.example.com/path",
			bearer:  "Bearer valid",
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
			name:         "invalid bearer fallback rejects request",
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
			for name, value := range map[string]string{
				csrf.HeaderSecFetchSite: test.site,
				csrf.HeaderOrigin:       test.origin,
				csrf.HeaderReferer:      test.referer,
				"Authorization":         test.bearer,
			} {
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

// Verifies the token-exchange CSRF exception is limited to non-browser JSON
// token exchanges and leaves requests with browser context, cookies, or other
// routes under normal CSRF checks.
func TestStatelessTokenExchangeCSRFBypass(t *testing.T) {
	conf := config.CSRFConfig{
		Namespace:       "quarterdeck",
		ExpectedOrigins: []string{"https://app.example.com"},
	}
	tests := []struct {
		name     string
		path     string
		headers  map[string]string
		allowed  bool
		bypassed bool
	}{
		{
			name: "authenticate SDK request",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			allowed:  true,
			bypassed: true,
		},
		{
			name: "reauthenticate SDK request",
			path: "/v1/reauthenticate",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
			allowed:  true,
			bypassed: true,
		},
		{
			name: "other endpoint remains protected",
			path: "/v1/login",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
			},
		},
		{
			name: "non-JSON request remains protected",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type": "text/plain",
				"Accept":       "application/json",
			},
		},
		{
			name: "HTML request remains protected",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "text/html",
			},
		},
		{
			name: "explicit cross-site metadata remains protected",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type":          "application/json",
				"Accept":                "application/json",
				csrf.HeaderSecFetchSite: "cross-site",
			},
		},
		{
			name: "untrusted origin remains protected",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type":    "application/json",
				"Accept":          "application/json",
				csrf.HeaderOrigin: "https://attacker.example",
			},
		},
		{
			name: "referer remains protected",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type":     "application/json",
				"Accept":           "application/json",
				csrf.HeaderReferer: "https://attacker.example/page",
			},
		},
		{
			name: "fetch metadata remains protected",
			path: "/v1/reauthenticate",
			headers: map[string]string{
				"Content-Type":          "application/json",
				"Accept":                "application/json",
				csrf.HeaderSecFetchMode: "cors",
			},
		},
		{
			name: "ambient cookies remain protected",
			path: "/v1/reauthenticate",
			headers: map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/json",
				"Cookie":       "session=existing",
			},
		},
		{
			name: "trusted browser origin uses normal CSRF policy",
			path: "/v1/authenticate",
			headers: map[string]string{
				"Content-Type":    "application/json",
				"Accept":          "application/json",
				csrf.HeaderOrigin: "https://app.example.com",
			},
			allowed: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(csrfProtection(conf, nil))
			reached, bypassed := false, false
			router.POST(test.path, func(c *gin.Context) {
				reached = true
				bypassed = isStatelessTokenExchangeRequest(c)
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodPost, "https://auth.example.com"+test.path, nil)
			for name, value := range test.headers {
				request.Header.Set(name, value)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			require.Equal(t, test.allowed, reached)
			require.Equal(t, test.bypassed, bypassed)
			if test.allowed {
				require.Equal(t, http.StatusNoContent, recorder.Code)
			} else {
				require.Equal(t, http.StatusForbidden, recorder.Code)
				require.Equal(t, csrf.ErrorRequestRejected, recorder.Header().Get(conf.ErrorHeader()))
			}
		})
	}
}

// Exercises both real token-exchange handlers through CSRF middleware and verifies that
// SDK-style requests receive JSON credentials without creating ambient auth cookies.
func TestStatelessTokenExchangeDoesNotSetAuthCookies(t *testing.T) {
	srv, apiKey, clientSecret := newTokenExchangeTestServer(t)

	t.Run("authenticate", func(t *testing.T) {
		// Authenticate with API-key credentials, as the SDK does on its first request.
		body, err := json.Marshal(api.AuthenticateRequest{
			ClientID:     apiKey.ClientID,
			ClientSecret: clientSecret,
		})
		require.NoError(t, err)

		// Exercise the real handler behind the CSRF middleware.
		router := gin.New()
		router.Use(csrfProtection(config.CSRFConfig{Namespace: "quarterdeck"}, srv.issuer))
		router.POST("/v1/authenticate", srv.Authenticate)

		recorder := performTokenExchangeRequest(t, router, "/v1/authenticate", body)
		assertTokenExchangeResponse(t, recorder)
	})

	t.Run("reauthenticate", func(t *testing.T) {
		// Create a refresh token for the API key and submit it through the real handler.
		_, refreshToken, err := srv.issuer.CreateTokens(apiKey.Claims())
		require.NoError(t, err)
		body, err := json.Marshal(api.ReauthenticateRequest{RefreshToken: refreshToken})
		require.NoError(t, err)

		router := gin.New()
		router.Use(csrfProtection(config.CSRFConfig{Namespace: "quarterdeck"}, srv.issuer))
		router.POST("/v1/reauthenticate", srv.Reauthenticate)

		recorder := performTokenExchangeRequest(t, router, "/v1/reauthenticate", body)
		assertTokenExchangeResponse(t, recorder)
	})
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
	request.Header.Set(csrf.HeaderSecFetchSite, "cross-site")
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
			for name, value := range map[string]string{
				csrf.HeaderSecFetchSite: test.site,
				csrf.HeaderSecFetchMode: test.mode,
				csrf.HeaderSecFetchDest: test.destination,
				"Authorization":         test.bearer,
			} {
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

// Records CSRF rejections and fallback-only acceptances on the active request span.
func TestCSRFTraceEvents(t *testing.T) {
	tests := []struct {
		name      string
		site      string
		bearer    string
		wantEvent string
		wantCode  int
	}{
		{
			name:      "rejected request",
			site:      "cross-site",
			wantEvent: "csrf.request.rejected",
			wantCode:  http.StatusForbidden,
		},
		{
			name:      "accepted by verified bearer fallback",
			bearer:    "Bearer valid",
			wantEvent: "csrf.request.accepted_by_fallback",
			wantCode:  http.StatusNoContent,
		},
		{
			name:     "accepted by same-origin policy",
			site:     "same-origin",
			wantCode: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spanRecorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
			ctx, span := provider.Tracer("csrf-test").Start(context.Background(), "request")

			router := gin.New()
			router.Use(csrfProtection(config.CSRFConfig{Namespace: "quarterdeck"}, &csrfTestIssuer{}))
			router.POST("/action", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			request := httptest.NewRequest(http.MethodPost, "/action", nil).WithContext(ctx)
			if test.site != "" {
				request.Header.Set(csrf.HeaderSecFetchSite, test.site)
			}
			if test.bearer != "" {
				request.Header.Set("Authorization", test.bearer)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			span.End()

			require.Equal(t, test.wantCode, recorder.Code)
			endedSpans := spanRecorder.Ended()
			require.Len(t, endedSpans, 1)
			events := endedSpans[0].Events()
			if test.wantEvent == "" {
				require.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			require.Equal(t, test.wantEvent, events[0].Name)
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

// Builds a server with a mock API-key store and signing issuer for handler tests.
func newTokenExchangeTestServer(t *testing.T) (*Server, *models.APIKey, string) {
	t.Helper()

	clientSecret := "token-exchange-test-secret"
	derivedSecret, err := passwords.CreateDerivedKey(clientSecret)
	require.NoError(t, err)

	mockStore, err := mock.Open(nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, mockStore.Close()) })

	apiKey := &models.APIKey{
		Model:    models.Model{ID: ulid.MakeSecure()},
		ClientID: "token-exchange-test-client",
		Secret:   derivedSecret,
	}
	mockStore.OnRetrieveAPIKey = func(_ context.Context, _ any) (*models.APIKey, error) {
		return apiKey, nil
	}
	mockStore.OnUpdateLastSeen = func(_ context.Context, _ ulid.ULID, _ time.Time) error {
		return nil
	}

	issuer, err := quarterdeckauth.NewIssuer(config.AuthConfig{
		Issuer:          "https://auth.example.com",
		Audience:        []string{"https://app.example.com"},
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -time.Hour,
	})
	require.NoError(t, err)

	return &Server{store: mockStore, issuer: issuer}, apiKey, clientSecret
}

// Sends the token-exchange request with the same JSON headers as the Guidelight
// Python SDK.
func performTokenExchangeRequest(t *testing.T, router http.Handler, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "https://auth.example.com"+path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// Verifies token exchange succeeds and returns credentials without setting cookies.
func assertTokenExchangeResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Empty(t, recorder.Result().Cookies())

	var reply api.LoginReply
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &reply))
	require.NotEmpty(t, reply.AccessToken)
	require.NotEmpty(t, reply.RefreshToken)
}
