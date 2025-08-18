package config_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// The test environment for all config tests, manipulated using curEnv and setEnv
var testEnv = map[string]string{
	"QD_MAINTENANCE":                  "false",
	"QD_BIND_ADDR":                    ":3636",
	"QD_MODE":                         gin.TestMode,
	"QD_LOG_LEVEL":                    "error",
	"QD_CONSOLE_LOG":                  "true",
	"QD_ALLOW_ORIGINS":                "https://example.com,https://auth.example.com,https://db.example.com",
	"QD_RATE_LIMIT_TYPE":              "ipaddr",
	"QD_RATE_LIMIT_PER_SECOND":        "20",
	"QD_RATE_LIMIT_BURST":             "100",
	"QD_RATE_LIMIT_CACHE_TTL":         "1h",
	"QD_DATABASE_URL":                 "sqlite3:///test.db",
	"QD_DATABASE_READ_ONLY":           "true",
	"QD_AUTH_KEYS":                    "01GECSDK5WJ7XWASQ0PMH6K41K:testdata/01GECSDK5WJ7XWASQ0PMH6K41K.pem,01GECSJGDCDN368D0EENX23C7R:testdata/01GECSJGDCDN368D0EENX23C7R.pem",
	"QD_AUTH_AUDIENCE":                "https://example.com,https://db.example.com",
	"QD_AUTH_ISSUER":                  "https://auth.example.com",
	"QD_AUTH_LOGIN_URL":               "https://example.com/signin",
	"QD_AUTH_LOGOUT_REDIRECT":         "https://example.com/signout",
	"QD_AUTH_LOGIN_REDIRECT":          "https://example.com/dashboard",
	"QD_AUTH_AUTHENTICATE_REDIRECT":   "https://example.com/dashboard/authenticated",
	"QD_AUTH_REAUTHENTICATE_REDIRECT": "https://example.com/dashboard/reauthenticated",
	"QD_AUTH_ACCESS_TOKEN_TTL":        "5m",
	"QD_AUTH_REFRESH_TOKEN_TTL":       "10m",
	"QD_AUTH_TOKEN_OVERLAP":           "-2m",
	"QD_CSRF_COOKIE_TTL":              "20m",
	"QD_CSRF_SECRET":                  "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
	"QD_SECURITY_TXT_PATH":            "./security.txt",
}

func TestConfig(t *testing.T) {
	// Set the required environment variables and cleanup after.
	prevEnv := curEnv()
	t.Cleanup(func() {
		for key, val := range prevEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	})
	setEnv()

	// At this point in the test, the environment should contain testEnv
	conf, err := config.New()
	require.NoError(t, err, "could not create a default config")
	require.False(t, conf.IsZero(), "default config should be processed")

	// Test the configuration
	require.False(t, conf.Maintenance)
	require.Equal(t, testEnv["QD_BIND_ADDR"], conf.BindAddr)
	require.Equal(t, testEnv["QD_MODE"], conf.Mode)
	require.Equal(t, zerolog.ErrorLevel, conf.GetLogLevel())
	require.True(t, conf.ConsoleLog)
	require.Equal(t, []string{"https://example.com", "https://auth.example.com", "https://db.example.com"}, conf.AllowOrigins)
	require.Equal(t, testEnv["QD_DATABASE_URL"], conf.Database.URL)
	require.True(t, conf.Database.ReadOnly)
	require.Len(t, conf.Auth.Keys, 2)
	require.Equal(t, []string{"https://example.com", "https://db.example.com"}, conf.Auth.Audience)
	require.Equal(t, testEnv["QD_AUTH_ISSUER"], conf.Auth.Issuer)
	require.Equal(t, testEnv["QD_AUTH_LOGIN_URL"], conf.Auth.LoginURL)
	require.Equal(t, testEnv["QD_AUTH_LOGOUT_REDIRECT"], conf.Auth.LogoutRedirect)
	require.Equal(t, testEnv["QD_AUTH_LOGIN_REDIRECT"], conf.Auth.LoginRedirect)
	require.Equal(t, testEnv["QD_AUTH_AUTHENTICATE_REDIRECT"], conf.Auth.AuthenticateRedirect)
	require.Equal(t, testEnv["QD_AUTH_REAUTHENTICATE_REDIRECT"], conf.Auth.ReauthenticateRedirect)
	require.Equal(t, 5*time.Minute, conf.Auth.AccessTokenTTL)
	require.Equal(t, 10*time.Minute, conf.Auth.RefreshTokenTTL)
	require.Equal(t, -2*time.Minute, conf.Auth.TokenOverlap)
	require.Equal(t, testEnv["QD_RATE_LIMIT_TYPE"], conf.RateLimit.Type)
	require.Equal(t, 20.00, conf.RateLimit.PerSecond)
	require.Equal(t, 100, conf.RateLimit.Burst)
	require.Equal(t, 60*time.Minute, conf.RateLimit.CacheTTL)
	require.Equal(t, 20*time.Minute, conf.CSRF.CookieTTL)
	require.Equal(t, testEnv["QD_CSRF_SECRET"], conf.CSRF.Secret)
	require.Equal(t, testEnv["QD_SECURITY_TXT_PATH"], conf.Security.TxtPath)
}

func TestValidation(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		// Ensure the default config is valid
		conf, err := config.New()
		require.NoError(t, err, "could not create default config")
		require.NoError(t, conf.Validate(), "expected default config to be valid")
	})

	t.Run("Valid", func(t *testing.T) {
		tests := []config.Config{
			{
				Maintenance:  false,
				BindAddr:     ":3333",
				LogLevel:     logger.LevelDecoder(zerolog.InfoLevel),
				Mode:         gin.ReleaseMode,
				ConsoleLog:   true,
				AllowOrigins: []string{"*"},
			},
			{
				Maintenance:  false,
				BindAddr:     ":3333",
				LogLevel:     logger.LevelDecoder(zerolog.InfoLevel),
				Mode:         gin.DebugMode,
				ConsoleLog:   true,
				AllowOrigins: []string{"*"},
			},
			{
				Maintenance:  false,
				BindAddr:     ":3333",
				LogLevel:     logger.LevelDecoder(zerolog.InfoLevel),
				Mode:         gin.TestMode,
				ConsoleLog:   true,
				AllowOrigins: []string{"*"},
			},
		}

		for i, conf := range tests {
			require.NoError(t, conf.Validate(), "expected config validation to pass on test case %d", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			conf config.Config
			errs string
		}{
			{
				conf: config.Config{
					Maintenance:  false,
					BindAddr:     ":3333",
					LogLevel:     logger.LevelDecoder(zerolog.InfoLevel),
					Mode:         "invalid",
					ConsoleLog:   true,
					AllowOrigins: []string{"*"},
				},
				errs: `invalid configuration: mode "invalid" is not a valid gin mode`,
			},
		}

		for i, test := range tests {
			err := test.conf.Validate()
			require.EqualError(t, err, test.errs, "expected config validation error on test case %d", i)
		}
	})
}

func TestIsZero(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		// An empty config should always return IsZero
		require.True(t, config.Config{}.IsZero(), "an empty config should always be zero valued")
	})

	t.Run("Processed", func(t *testing.T) {
		// A processed config should not be zero valued
		conf, err := config.New()
		require.NoError(t, err, "should have been able to load the config")
		require.False(t, conf.IsZero(), "expected a processed config to be non-zero valued")
	})

	t.Run("Unprocessed", func(t *testing.T) {
		// Custom config not processed
		conf := config.Config{
			Maintenance: false,
			BindAddr:    "127.0.0.1:0",
			LogLevel:    logger.LevelDecoder(zerolog.TraceLevel),
			Mode:        "invalid",
		}
		require.True(t, conf.IsZero(), "a non-empty config that isn't marked will be zero valued")
	})

	t.Run("Mark", func(t *testing.T) {
		t.Run("Valid", func(t *testing.T) {
			conf := config.Config{
				Maintenance: false,
				BindAddr:    "127.0.0.1:0",
				LogLevel:    logger.LevelDecoder(zerolog.TraceLevel),
				Mode:        gin.ReleaseMode,
			}

			conf, err := conf.Mark()
			require.NoError(t, err, "should be able to mark a valid config")
			require.False(t, conf.IsZero(), "a marked config should not be zero-valued")
		})

		t.Run("Invalid", func(t *testing.T) {
			conf := config.Config{
				Maintenance: false,
				BindAddr:    "127.0.0.1:0",
				LogLevel:    logger.LevelDecoder(zerolog.TraceLevel),
				Mode:        "invalid",
			}

			// Should not be able to mark a custom config that is invalid
			conf, err := conf.Mark()
			require.EqualError(t, err, `invalid configuration: mode "invalid" is not a valid gin mode`, "expected gin mode validation error")
			require.True(t, conf.IsZero(), "an invalid config when marked should be zero-valued")
		})
	})
}

func TestCookieDomains(t *testing.T) {
	testCases := []struct {
		conf     config.Config
		expected []string
	}{
		{
			conf: config.Config{
				AllowOrigins: []string{"http://localhost:8000"},
			},
			expected: []string{"localhost"},
		},
		{
			conf: config.Config{
				AllowOrigins: []string{"http://example.com:8080"},
			},
			expected: []string{"example.com"},
		},
		{
			conf: config.Config{
				AllowOrigins: []string{"https://example.com"},
			},
			expected: []string{"example.com"},
		},
		{
			conf: config.Config{
				AllowOrigins: []string{"https://example.com", "https://auth.example.com", "https://db.example.com"},
			},
			expected: []string{"example.com", "auth.example.com", "db.example.com"},
		},
		{
			conf: config.Config{
				AllowOrigins: []string{"http://localhost:8000", "http://localhost:8888", "http://localhost:4444"},
			},
			expected: []string{"localhost"},
		},
	}

	for i, tc := range testCases {
		cookieDomains := tc.conf.CookieDomains()
		for _, domain := range tc.expected {
			require.Contains(t, cookieDomains, domain, "expected cookie domains to contain %q for test case %d", domain, i)
		}
	}
}

func TestAuthConfigValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []config.AuthConfig{
			{
				Audience:        []string{"https://example.com"},
				Issuer:          "https://auth.example.com",
				AccessTokenTTL:  5 * time.Minute,
				RefreshTokenTTL: 10 * time.Minute,
				TokenOverlap:    -2 * time.Minute,
			},
			{
				Keys:            map[string]string{},
				Audience:        []string{"https://example.com"},
				Issuer:          "https://example.com",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 48 * time.Hour,
				TokenOverlap:    -12 * time.Hour,
			},
			{
				Keys:            map[string]string{},
				Audience:        []string{"https://example.com", "https://sub.example.com"},
				Issuer:          "https://example.com",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 48 * time.Hour,
				TokenOverlap:    -12 * time.Hour,
			},
			{
				Keys:            map[string]string{},
				Audience:        []string{"https://example.com", "https://sub.example.com"},
				Issuer:          "https://example.com",
				LoginURL:        "https://sub.example.com/login",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 48 * time.Hour,
				TokenOverlap:    -12 * time.Hour,
			},
			{
				Keys:            map[string]string{},
				Audience:        []string{"https://example.com", "https://sub.example.com"},
				Issuer:          "https://example.com",
				LogoutRedirect:  "https://sub.example.com/logout",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 48 * time.Hour,
				TokenOverlap:    -12 * time.Hour,
			},
			{
				Keys:                   map[string]string{},
				Audience:               []string{"https://example.com", "https://sub.example.com"},
				Issuer:                 "https://example.com",
				LoginRedirect:          "https://sub.example.com/login",
				AuthenticateRedirect:   "https://sub.example.com/authenticate",
				ReauthenticateRedirect: "https://sub.example.com/reauthenticate",
				AccessTokenTTL:         24 * time.Hour,
				RefreshTokenTTL:        48 * time.Hour,
				TokenOverlap:           -12 * time.Hour,
			},
		}

		for i, conf := range tests {
			require.NoError(t, conf.Validate(), "expected auth config validation to pass on test case %d", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			conf config.AuthConfig
			errs string
		}{
			{
				conf: config.AuthConfig{
					Audience:        nil,
					Issuer:          "https://auth.example.com",
					LoginURL:        "https://auth.example.com/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.audience is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"\x00"},
					Issuer:          "https://auth.example.com",
					LoginURL:        "https://auth.example.com/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.audience could not parse audience: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "",
					LoginURL:        "https://auth.example.com/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.issuer is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "\x00",
					LoginURL:        "https://auth.example.com/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.issuer cannot be used as a login url: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "//auth.example.com",
					LoginURL:        "https://auth.example.com/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.issuer cannot be used as a login url: invalid login url: \"//auth.example.com\"",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://",
					LoginURL:        "https://auth.example.com/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.issuer cannot be used as a login url: invalid login url: \"https://\"",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://example.com",
					LoginURL:        "\x00",
					LogoutRedirect:  "/logout",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.loginURL could not parse loginURL: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://example.com",
					LoginURL:        "/login",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.loginURL could not parse loginURL: invalid login url: \"/login\"",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://example.com",
					LogoutRedirect:  "\x00",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.logoutRedirect could not parse logoutRedirect: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://example.com",
					LoginRedirect:   "\x00",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.loginRedirect could not parse loginRedirect: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:             []string{"https://example.com"},
					Issuer:               "https://example.com",
					AuthenticateRedirect: "\x00",
					AccessTokenTTL:       5 * time.Minute,
					RefreshTokenTTL:      10 * time.Minute,
					TokenOverlap:         -2 * time.Minute,
				},
				errs: "invalid configuration: auth.authenticateRedirect could not parse authenticateRedirect: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:               []string{"https://example.com"},
					Issuer:                 "https://example.com",
					ReauthenticateRedirect: "\x00",
					AccessTokenTTL:         5 * time.Minute,
					RefreshTokenTTL:        10 * time.Minute,
					TokenOverlap:           -2 * time.Minute,
				},
				errs: "invalid configuration: auth.reauthenticateRedirect could not parse reauthenticateRedirect: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  0 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    0 * time.Minute,
				},
				errs: "invalid configuration: auth.accessTokenTTL is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  20 * time.Minute,
					RefreshTokenTTL: -10 * time.Minute,
					TokenOverlap:    -5 * time.Minute,
				},
				errs: "invalid configuration: auth.refreshTokenTTL is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  20 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    2 * time.Minute,
				},
				errs: "invalid configuration: auth.tokenOverlap must be negative and not exceed the access duration",
			},
			{
				conf: config.AuthConfig{
					Audience:        []string{"https://example.com"},
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  20 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -24 * time.Minute,
				},
				errs: "invalid configuration: auth.tokenOverlap must be negative and not exceed the access duration",
			},
		}

		for i, test := range tests {
			err := test.conf.Validate()
			require.EqualError(t, err, test.errs, "expected auth config validation error on test case %d", i)
		}
	})
}

func TestAuthConfigDefaults(t *testing.T) {
	config := config.AuthConfig{
		Audience:        []string{"https://example.com"},
		Issuer:          "https://example.com",
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 10 * time.Minute,
		TokenOverlap:    -2 * time.Minute,
	}

	resetConfig := func() {
		config.Issuer = "https://example.com"
		config.LoginURL = ""
		config.LogoutRedirect = ""
		config.LoginRedirect = ""
		config.AuthenticateRedirect = ""
		config.ReauthenticateRedirect = ""
	}

	t.Run("Defaults", func(t *testing.T) {
		t.Run("WithoutTrailingSlash", func(t *testing.T) {
			defer resetConfig()

			config.Issuer = "https://example.com"
			require.Empty(t, config.LoginURL, "expected login URL to be empty before validation")
			require.Empty(t, config.LogoutRedirect, "expected logout URL to be empty before validation")
			require.Empty(t, config.LoginRedirect, "expected login redirect to be empty before validation")
			require.Empty(t, config.AuthenticateRedirect, "expected authenticate redirect to be empty before validation")
			require.Empty(t, config.ReauthenticateRedirect, "expected reauthenticate redirect to be empty before validation")

			// Validation sets the defaults
			require.NoError(t, config.Validate(), "expected auth config validation to pass with default values")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://example.com/login", config.LoginURL)
			require.Equal(t, "https://example.com/login", config.LogoutRedirect)
			require.Equal(t, "https://example.com/", config.LoginRedirect)
			require.Equal(t, "https://example.com/", config.AuthenticateRedirect)
			require.Equal(t, "https://example.com/", config.ReauthenticateRedirect)
		})

		t.Run("WithTrailingSlash", func(t *testing.T) {
			defer resetConfig()

			config.Issuer = "https://example.com/"
			require.Empty(t, config.LoginURL, "expected login URL to be empty before validation")
			require.Empty(t, config.LogoutRedirect, "expected logout URL to be empty before validation")
			require.Empty(t, config.LoginRedirect, "expected login redirect to be empty before validation")

			// Validation sets the defaults
			require.NoError(t, config.Validate(), "expected auth config validation to pass with default values")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://example.com/login", config.LoginURL)
			require.Equal(t, "https://example.com/login", config.LogoutRedirect)
			require.Equal(t, "https://example.com/", config.LoginRedirect)
			require.Equal(t, "https://example.com/", config.AuthenticateRedirect)
			require.Equal(t, "https://example.com/", config.ReauthenticateRedirect)
		})
	})

	t.Run("Override", func(t *testing.T) {
		t.Run("All", func(t *testing.T) {
			defer resetConfig()

			config.LoginURL = "https://testing.com/login"
			config.LogoutRedirect = "https://testing.com/logout"
			config.LoginRedirect = "https://testing.com/"
			config.AuthenticateRedirect = "https://testing.com/authenticate"
			config.ReauthenticateRedirect = "https://testing.com/reauthenticate"

			require.NoError(t, config.Validate(), "expected auth config validation to pass with overridden values")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://testing.com/login", config.LoginURL)
			require.Equal(t, "https://testing.com/logout", config.LogoutRedirect)
			require.Equal(t, "https://testing.com/", config.LoginRedirect)
			require.Equal(t, "https://testing.com/authenticate", config.AuthenticateRedirect)
			require.Equal(t, "https://testing.com/reauthenticate", config.ReauthenticateRedirect)
		})

		t.Run("LoginURL", func(t *testing.T) {
			defer resetConfig()

			config.LoginURL = "https://testing.com/login"
			require.NoError(t, config.Validate(), "expected auth config validation to pass with overridden login URL")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://testing.com/login", config.LoginURL)
			require.Equal(t, "https://testing.com/login", config.LogoutRedirect)
			require.Equal(t, "https://example.com/", config.LoginRedirect)
			require.Equal(t, "https://example.com/", config.AuthenticateRedirect)
			require.Equal(t, "https://example.com/", config.ReauthenticateRedirect)
		})

		t.Run("LogoutURL", func(t *testing.T) {
			defer resetConfig()

			config.LogoutRedirect = "https://testing.com/logout"
			require.NoError(t, config.Validate(), "expected auth config validation to pass with overridden logout URL")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://example.com/login", config.LoginURL)
			require.Equal(t, "https://testing.com/logout", config.LogoutRedirect)
			require.Equal(t, "https://example.com/", config.LoginRedirect)
			require.Equal(t, "https://example.com/", config.AuthenticateRedirect)
			require.Equal(t, "https://example.com/", config.ReauthenticateRedirect)
		})

		t.Run("LoginRedirect", func(t *testing.T) {
			defer resetConfig()

			config.LoginRedirect = "https://testing.com/"
			require.NoError(t, config.Validate(), "expected auth config validation to pass with overridden login redirect")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://example.com/login", config.LoginURL)
			require.Equal(t, "https://example.com/login", config.LogoutRedirect)
			require.Equal(t, "https://testing.com/", config.LoginRedirect)
			require.Equal(t, "https://example.com/", config.AuthenticateRedirect)
			require.Equal(t, "https://example.com/", config.ReauthenticateRedirect)
		})

		t.Run("Some", func(t *testing.T) {
			defer resetConfig()

			config.LoginURL = "https://testing.com/login"
			config.LoginRedirect = "https://testing.com/"
			require.NoError(t, config.Validate(), "expected auth config validation to pass with overridden login redirect")

			require.Equal(t, "https://example.com", config.Issuer, "expected issuer to be set without trailing slash")
			require.Equal(t, "https://testing.com/login", config.LoginURL)
			require.Equal(t, "https://testing.com/login", config.LogoutRedirect)
			require.Equal(t, "https://testing.com/", config.LoginRedirect)
			require.Equal(t, "https://example.com/", config.AuthenticateRedirect)
			require.Equal(t, "https://example.com/", config.ReauthenticateRedirect)
		})
	})

	t.Run("AudienceStripSlashes", func(t *testing.T) {
		defer resetConfig()
		config.Audience = []string{
			"https://example.com/",
			"https://sub.example.com",
			"https://another.example.com/",
			"https://specific.example.com/endpoint",
			"https://specific.example.com/endpoint/slash/",
		}

		require.NoError(t, config.Validate(), "expected auth config validation to pass with overridden login redirect")
		require.Equal(t, []string{
			"https://example.com",
			"https://sub.example.com",
			"https://another.example.com",
			"https://specific.example.com/endpoint",
			"https://specific.example.com/endpoint/slash/",
		}, config.Audience)

	})
}

func TestCSRFConfigValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []config.CSRFConfig{
			{
				CookieTTL: 20 * time.Minute,
				Secret:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			},
			{
				CookieTTL: 30 * time.Minute,
				Secret:    "",
			},
		}

		for i, conf := range tests {
			require.NoError(t, conf.Validate(), "expected csrf config validation to pass on test case %d", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			conf config.CSRFConfig
			errs string
		}{
			{
				conf: config.CSRFConfig{
					CookieTTL: 0,
					Secret:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
				},
				errs: "invalid configuration: csrf.cookieTTL is required but not set",
			},
			{
				conf: config.CSRFConfig{
					CookieTTL: 20 * time.Minute,
					Secret:    "invalidhexstring",
				},
				errs: "invalid configuration: csrf.secret could not parse secret: encoding/hex: invalid byte: U+0069 'i'",
			},
		}

		for i, test := range tests {
			err := test.conf.Validate()
			require.EqualError(t, err, test.errs, "expected csrf config validation error on test case %d", i)
		}
	})
}

func TestCSRFGetSecret(t *testing.T) {
	t.Run("WithSecret", func(t *testing.T) {
		conf := config.CSRFConfig{
			CookieTTL: 20 * time.Minute,
			Secret:    "000102030405060708090a0b0c0d0e0f",
		}
		require.NoError(t, conf.Validate(), "should be able to validate the csrf config with a secret")
		require.Equal(t, []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, conf.GetSecret())
	})

	t.Run("WithoutSecret", func(t *testing.T) {
		conf := config.CSRFConfig{
			CookieTTL: 20 * time.Minute,
			Secret:    "",
		}
		require.NoError(t, conf.Validate(), "should be able to validate the csrf config without a secret")

		// Require secret is 65 characters that aren't all zeros
		secret := conf.GetSecret()
		require.Len(t, secret, 65)
		require.NotEqual(t, bytes.Repeat([]byte{0}, 65), secret)

		// Require generating a new secret doesn't return the original
		require.NotEqual(t, secret, conf.GetSecret())
	})
}

// Returns the current environment for the specified keys, or if no keys are specified
// then returns the current environment for all keys in testEnv.
func curEnv(keys ...string) map[string]string {
	env := make(map[string]string)

	if len(keys) > 0 {
		// Process the keys passed in by the user
		for _, key := range keys {
			if val, ok := os.LookupEnv(key); ok {
				env[key] = val
			}
		}
	} else {
		// Process all the keys in testEnv
		for key := range testEnv {
			env[key] = os.Getenv(key)
		}
	}

	return env
}

// Sets the environment variables from the testEnv, if no keys are specified then sets
// all environment variables that are specified in the testEnv.
func setEnv(keys ...string) {
	if len(keys) > 0 {
		for _, key := range keys {
			if val, ok := testEnv[key]; ok {
				os.Setenv(key, val)
			}
		}
	} else {
		for key, val := range testEnv {
			os.Setenv(key, val)
		}
	}
}
