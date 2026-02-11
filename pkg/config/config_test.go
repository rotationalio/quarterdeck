package config_test

import (
	"net/mail"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/commo"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/gimlet/secure"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// The test environment for all config tests, manipulated using curEnv and setEnv
var testEnv = map[string]string{
	"QD_MAINTENANCE":                                "false",
	"QD_BIND_ADDR":                                  ":3636",
	"QD_MODE":                                       gin.TestMode,
	"QD_LOG_LEVEL":                                  "error",
	"QD_CONSOLE_LOG":                                "true",
	"QD_ALLOW_ORIGINS":                              "https://example.com,https://auth.example.com,https://db.example.com",
	"QD_DATABASE_URL":                               "sqlite3:///test.db",
	"QD_DATABASE_READ_ONLY":                         "true",
	"QD_AUTH_KEYS":                                  "01GECSDK5WJ7XWASQ0PMH6K41K:testdata/01GECSDK5WJ7XWASQ0PMH6K41K.pem,01GECSJGDCDN368D0EENX23C7R:testdata/01GECSJGDCDN368D0EENX23C7R.pem",
	"QD_AUTH_AUDIENCE":                              "https://example.com,https://db.example.com",
	"QD_AUTH_ISSUER":                                "https://auth.example.com",
	"QD_AUTH_LOGIN_URL":                             "https://example.com/signin",
	"QD_AUTH_LOGOUT_REDIRECT":                       "https://example.com/signout",
	"QD_AUTH_LOGIN_REDIRECT":                        "https://example.com/dashboard",
	"QD_AUTH_AUTHENTICATE_REDIRECT":                 "https://example.com/dashboard/authenticated",
	"QD_AUTH_REAUTHENTICATE_REDIRECT":               "https://example.com/dashboard/reauthenticated",
	"QD_AUTH_ACCESS_TOKEN_TTL":                      "5m",
	"QD_AUTH_REFRESH_TOKEN_TTL":                     "10m",
	"QD_AUTH_TOKEN_OVERLAP":                         "-2m",
	"QD_CSRF_COOKIE_TTL":                            "20m",
	"QD_CSRF_SECRET":                                "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
	"QD_SECURE_CONTENT_TYPE_NOSNIFF":                "false",
	"QD_SECURE_CROSS_ORIGIN_OPENER_POLICY":          "noopener-allow-popups",
	"QD_SECURE_REFERRER_POLICY":                     "same-origin",
	"QD_SECURE_CONTENT_SECURITY_POLICY_DEFAULT_SRC": "https:",
	"QD_SECURE_CONTENT_SECURITY_POLICY_REPORT_ONLY_SCRIPT_SRC": "'self',*.cloudflare.com",
	"QD_SECURE_CONTENT_SECURITY_POLICY_REPORT_ONLY_REPORT_TO":  "csp-endpoint",
	"QD_SECURE_REPORTING_ENDPOINTS":                            `csp-endpoint://example.com/csp-reports`,
	"QD_SECURE_HSTS_SECONDS":                                   "63244800",
	"QD_SECURE_HSTS_INCLUDE_SUBDOMAINS":                        "true",
	"QD_SECURE_HSTS_PRELOAD":                                   "true",
	"QD_SECURITY_TXT_PATH":                                     "./security.txt",
	"QD_EMAIL_SENDER":                                          "Izuku Midoriya <izuku.midoriya@example.com>",
	"QD_EMAIL_SENDGRID_API_KEY":                                "sendgrid_api_key",
	"QD_EMAIL_BACKOFF_TIMEOUT":                                 "1s",
	"QD_EMAIL_BACKOFF_INITIAL_INTERVAL":                        "1s",
	"QD_EMAIL_BACKOFF_MAX_INTERVAL":                            "1s",
	"QD_EMAIL_BACKOFF_MAX_ELAPSED_TIME":                        "1s",
	"QD_APP_NAME":                                              "AppName",
	"QD_APP_LOGO_URI":                                          "http://localhost:8000/logo.png",
	"QD_APP_BASE_URI":                                          "http://localhost:8000",
	"QD_APP_WELCOME_EMAIL_BODY_TEXT":                           "Email Body Text",
	"QD_APP_WELCOME_EMAIL_BODY_HTML":                           "Email Body HTML",
	"QD_APP_WEBHOOK_URI":                                       "http://localhost:8000/api/v1/users/sync",
	"QD_ORG_NAME":                                              "OrgName",
	"QD_ORG_STREET_ADDRESS":                                    "Org Street Address",
	"QD_ORG_HOMEPAGE_URI":                                      "http://example.com",
	"QD_ORG_SUPPORT_EMAIL":                                     "support@example.com",
	"QD_RATE_LIMIT_TYPE":                                       "ipaddr",
	"QD_RATE_LIMIT_PER_SECOND":                                 "20",
	"QD_RATE_LIMIT_BURST":                                      "100",
	"QD_RATE_LIMIT_CACHE_TTL":                                  "1h",
	"QD_TELEMETRY_ENABLED":                                     "false",
	"OTEL_SERVICE_NAME":                                        "bosun",
	"GIMLET_OTEL_SERVICE_ADDR":                                 "bosun.example.com:8080",
}

func TestConfigImport(t *testing.T) {
	// Set the required environment variables and cleanup after.
	prevEnv := curEnv()
	t.Cleanup(cleanup(prevEnv))
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
	require.Equal(t, 20*time.Minute, conf.CSRF.CookieTTL)
	require.Equal(t, testEnv["QD_CSRF_SECRET"], conf.CSRF.Secret)
	require.False(t, conf.Secure.ContentTypeNosniff)
	require.Equal(t, testEnv["QD_SECURE_CROSS_ORIGIN_OPENER_POLICY"], conf.Secure.CrossOriginOpenerPolicy)
	require.Equal(t, testEnv["QD_SECURE_REFERRER_POLICY"], conf.Secure.ReferrerPolicy)
	require.Equal(t, "default-src https:", conf.Secure.ContentSecurityPolicy.Directive())
	require.Equal(t, "script-src 'self' *.cloudflare.com; report-to csp-endpoint", conf.Secure.ContentSecurityPolicyReportOnly.Directive())
	require.Equal(t, map[string]string{"csp-endpoint": "//example.com/csp-reports"}, conf.Secure.ReportingEndpoints)
	require.Equal(t, 63244800, conf.Secure.HSTS.Seconds)
	require.True(t, conf.Secure.HSTS.IncludeSubdomains)
	require.True(t, conf.Secure.HSTS.Preload)
	require.Equal(t, testEnv["QD_SECURITY_TXT_PATH"], conf.Security.TxtPath)
	require.Equal(t, testEnv["QD_EMAIL_SENDER"], conf.Email.Sender)
	require.Zero(t, conf.Email.SenderName)
	addr, err := mail.ParseAddress(conf.Email.Sender)
	require.NoError(t, err)
	require.Equal(t, addr.Name, conf.Email.GetSenderName())
	dur, err := time.ParseDuration(testEnv["QD_EMAIL_BACKOFF_TIMEOUT"])
	require.NoError(t, err)
	require.Equal(t, dur, conf.Email.Backoff.Timeout)
	dur, err = time.ParseDuration(testEnv["QD_EMAIL_BACKOFF_INITIAL_INTERVAL"])
	require.NoError(t, err)
	require.Equal(t, dur, conf.Email.Backoff.InitialInterval)
	dur, err = time.ParseDuration(testEnv["QD_EMAIL_BACKOFF_MAX_INTERVAL"])
	require.NoError(t, err)
	require.Equal(t, dur, conf.Email.Backoff.MaxInterval)
	dur, err = time.ParseDuration(testEnv["QD_EMAIL_BACKOFF_MAX_ELAPSED_TIME"])
	require.NoError(t, err)
	require.Equal(t, dur, conf.Email.Backoff.MaxElapsedTime)
	require.Equal(t, testEnv["QD_APP_NAME"], "AppName")
	require.Equal(t, testEnv["QD_APP_LOGO_URI"], "http://localhost:8000/logo.png")
	require.Equal(t, testEnv["QD_APP_BASE_URI"], "http://localhost:8000")
	require.Equal(t, testEnv["QD_APP_WELCOME_EMAIL_BODY_TEXT"], "Email Body Text")
	require.Equal(t, testEnv["QD_APP_WELCOME_EMAIL_BODY_HTML"], "Email Body HTML")
	require.Equal(t, testEnv["QD_APP_WEBHOOK_URI"], conf.App.WebhookURI)
	require.Equal(t, testEnv["QD_ORG_NAME"], "OrgName")
	require.Equal(t, testEnv["QD_ORG_STREET_ADDRESS"], "Org Street Address")
	require.Equal(t, testEnv["QD_ORG_HOMEPAGE_URI"], "http://example.com")
	require.Equal(t, testEnv["QD_ORG_SUPPORT_EMAIL"], "support@example.com")
	require.Equal(t, testEnv["QD_RATE_LIMIT_TYPE"], conf.RateLimit.Type)
	require.Equal(t, 20.00, conf.RateLimit.PerSecond)
	require.Equal(t, 100, conf.RateLimit.Burst)
	require.Equal(t, 60*time.Minute, conf.RateLimit.CacheTTL)
	require.False(t, conf.Telemetry.Enabled)
	require.Equal(t, testEnv["OTEL_SERVICE_NAME"], conf.Telemetry.ServiceName)
	require.Equal(t, testEnv["GIMLET_OTEL_SERVICE_ADDR"], conf.Telemetry.ServiceAddr)
}

func TestGlobal(t *testing.T) {
	// Set the required environment variables and cleanup after.// Set the required environment variables and cleanup after.
	prevEnv := curEnv()
	t.Cleanup(cleanup(prevEnv))
	setEnv()

	t.Run("Get", func(t *testing.T) {
		t.Cleanup(func() {
			config.Reset()
			setEnv()
		})

		require.Nil(t, config.GlobalConf(), "expected config to be nil before test")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil before test")

		conf, err := config.Get()
		require.NoError(t, err, "could not get the config")
		require.False(t, conf.IsZero(), "config should be processed")

		// Changing the environment should not change the config created above.
		os.Setenv("QD_BIND_ADDR", ":4444")

		require.NotNil(t, config.GlobalConf(), "expected config to be non-nil after the Get loaded it")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil since the config is valid")

		conf2, err := config.Get()
		require.NoError(t, err, "could not get the config")
		require.Equal(t, conf, conf2, "expected config to be the same because of sync.Once")
	})

	t.Run("GetError", func(t *testing.T) {
		t.Cleanup(func() {
			config.Reset()
			setEnv()
		})

		require.Nil(t, config.GlobalConf(), "expected config to be nil before test")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil before test")

		// Add a bad value to the environment for an invalid configuration.
		os.Setenv("QD_MODE", "invalid")

		conf, err := config.Get()
		require.True(t, conf.IsZero())
		require.Error(t, err, "expecting an invalid configuration")

		// Get should return the same error even if the environment is changed.
		os.Setenv("QD_MODE", gin.TestMode)

		require.Nil(t, config.GlobalConf(), "expected config to be nil after getting invalid configuration")
		require.Error(t, config.GlobalConfErr(), "expected error to be the invalid configuration error")

		conf, err2 := config.Get()
		require.Equal(t, err, err2, "expected error to be the same because of sync.Once")
		require.True(t, conf.IsZero())
	})

	t.Run("Set", func(t *testing.T) {
		t.Cleanup(func() {
			config.Reset()
			setEnv()
		})

		require.Nil(t, config.GlobalConf(), "expected config to be nil before test")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil before test")

		conf, err := config.New()
		require.NoError(t, err, "could not create a new config")

		require.Nil(t, config.GlobalConf(), "global config should not be modified by New()")
		require.Nil(t, config.GlobalConfErr(), "global config error should not be modified by New()")

		// Modify the config to ensure it is not set by New()
		conf.BindAddr = ":4444"
		conf.DocsName = "bad bunny superbowl"

		// Set the config globally
		config.Set(conf)

		require.NotNil(t, config.GlobalConf(), "expected config to be non-nil after being Set()")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil after being Set()")

		// Once Set, the config should be the same as the one that was Set()
		conf2, err := config.Get()
		require.NoError(t, err, "could not get the config")
		require.Equal(t, conf, conf2, "expected config to be the same because of sync.Once")
	})

	t.Run("SetZero", func(t *testing.T) {
		t.Cleanup(func() {
			config.Reset()
			setEnv()
		})

		require.Nil(t, config.GlobalConf(), "expected config to be nil before test")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil before test")

		conf := config.Config{}
		config.Set(conf)

		require.Nil(t, config.GlobalConf(), "expected config to be nil after being Set()")
		require.Error(t, config.GlobalConfErr(), "expected error to be the invalid configuration error")

		conf2, err := config.Get()
		require.Error(t, err, "expected error to be the same because of sync.Once")
		require.True(t, conf2.IsZero())
	})

	t.Run("SetInvalid", func(t *testing.T) {
		t.Cleanup(func() {
			config.Reset()
			setEnv()
		})

		require.Nil(t, config.GlobalConf(), "expected config to be nil before test")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil before test")

		conf, err := config.New()
		require.NoError(t, err, "could not create a new config")

		// Modify the config to ensure it is invalid
		conf.Mode = "invalid"

		// Set the config globally
		config.Set(conf)

		require.Nil(t, config.GlobalConf(), "expected config to be nil after being Set()")
		require.Error(t, config.GlobalConfErr(), "expected error to be the invalid configuration error")

		conf2, err := config.Get()
		require.EqualError(t, err, "invalid configuration: mode \"invalid\" is not a valid gin mode", "expected validation error, not is zero error")
		require.True(t, conf2.IsZero())
	})

	t.Run("Reload", func(t *testing.T) {
		t.Cleanup(func() {
			config.Reset()
			setEnv()
		})

		require.Nil(t, config.GlobalConf(), "expected config to be nil before test")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil before test")

		conf, err := config.Get()
		require.NoError(t, err, "could not get the config")
		require.False(t, conf.IsZero(), "config should be processed")

		// Modify the environment to change the config when reloaded
		os.Setenv("QD_BIND_ADDR", ":4444")

		// Reload the config
		config.Reload()

		require.NotNil(t, config.GlobalConf(), "expected config to be non-nil after being Set()")
		require.Nil(t, config.GlobalConfErr(), "expected error to be nil after being Set()")

		// Once Reloaded, the config should contain the new values
		conf2, err := config.Get()
		require.NoError(t, err, "could not get the config")
		require.NotEqual(t, conf, conf2, "expected config to be modified by Reload()")
		require.Equal(t, ":4444", conf2.BindAddr, "expected bind addr to be the same")
	})
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
				RateLimit:   ratelimit.DefaultConfig,
				Auth: config.AuthConfig{
					Keys: map[string]string{
						"01GECSDK5WJ7XWASQ0PMH6K41K": "testdata/01GECSDK5WJ7XWASQ0PMH6K41K.pem",
						"01GECSJGDCDN368D0EENX23C7R": "testdata/01GECSJGDCDN368D0EENX23C7R.pem",
					},
					Audience:               []string{"https://example.com", "https://db.example.com"},
					Issuer:                 "https://auth.example.com",
					LoginURL:               "https://example.com/signin",
					ResetPasswordURL:       "https://example.com/signout",
					LogoutRedirect:         "https://example.com/dashboard",
					AuthenticateRedirect:   "https://example.com/dashboard/authenticated",
					ReauthenticateRedirect: "https://example.com/dashboard/reauthenticated",
					LoginRedirect:          "https://example.com/login",
					AccessTokenTTL:         24 * time.Hour,
					RefreshTokenTTL:        48 * time.Hour,
					TokenOverlap:           -12 * time.Hour,
				},
				CSRF: config.CSRFConfig{
					CookieTTL: 5 * time.Minute,
					Secret:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
				},
				Secure: secure.Config{
					ContentTypeNosniff:              false,
					CrossOriginOpenerPolicy:         "noopener-allow-popups",
					ReferrerPolicy:                  "same-origin",
					ContentSecurityPolicy:           secure.CSPDirectives{DefaultSrc: []string{"https:"}},
					ContentSecurityPolicyReportOnly: secure.CSPDirectives{ScriptSrc: []string{"'self'", "*.cloudflare.com"}, ReportTo: "csp-endpoint"},
					ReportingEndpoints:              map[string]string{"csp-endpoint": "//example.com/csp-reports"},
				},
				Email: commo.Config{
					Testing: false,
					Backoff: commo.BackoffConfig{
						Timeout:         1 * time.Second,
						InitialInterval: 1 * time.Second,
						MaxInterval:     1 * time.Second,
						MaxElapsedTime:  1 * time.Second,
					},
				},
				App: config.AppConfig{
					LogoURI:    "https://www.example.com/logo.png",
					BaseURI:    "https://www.example.com",
					WebhookURI: "https://www.example.com/api/v1/sync",
				},
				Org: config.OrgConfig{
					HomepageURI: "https://www.example.com",
				},
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

func cleanup(prevEnv map[string]string) func() {
	return func() {
		for key, val := range prevEnv {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}
