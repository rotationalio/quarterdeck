package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/logger"
)

// The test environment for all config tests, manipulated using curEnv and setEnv
var testEnv = map[string]string{
	"QD_MAINTENANCE":            "false",
	"QD_BIND_ADDR":              ":3636",
	"QD_MODE":                   gin.TestMode,
	"QD_LOG_LEVEL":              "error",
	"QD_CONSOLE_LOG":            "true",
	"QD_ALLOW_ORIGINS":          "http://localhost:8888,http://localhost:8080",
	"QD_RATE_LIMIT_ENABLED":     "true",
	"QD_RATE_LIMIT_PER_SECOND":  "20",
	"QD_RATE_LIMIT_BURST":       "100",
	"QD_RATE_LIMIT_TTL":         "1h",
	"QD_DATABASE_URL":           "sqlite3:///test.db",
	"QD_DATABASE_READ_ONLY":     "true",
	"QD_AUTH_KEYS":              "01GECSDK5WJ7XWASQ0PMH6K41K:testdata/01GECSDK5WJ7XWASQ0PMH6K41K.pem,01GECSJGDCDN368D0EENX23C7R:testdata/01GECSJGDCDN368D0EENX23C7R.pem",
	"QD_AUTH_AUDIENCE":          "http://localhost:8888",
	"QD_AUTH_ISSUER":            "http://localhost:1025",
	"QD_AUTH_ACCESS_TOKEN_TTL":  "5m",
	"QD_AUTH_REFRESH_TOKEN_TTL": "10m",
	"QD_AUTH_TOKEN_OVERLAP":     "-2m",
	"QD_SECURITY_TXT_PATH":      "./security.txt",
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
	require.Len(t, conf.AllowOrigins, 2)
	require.Equal(t, testEnv["QD_DATABASE_URL"], conf.Database.URL)
	require.True(t, conf.Database.ReadOnly)
	require.Len(t, conf.Auth.Keys, 2)
	require.Equal(t, testEnv["QD_AUTH_AUDIENCE"], conf.Auth.Audience)
	require.Equal(t, testEnv["QD_AUTH_ISSUER"], conf.Auth.Issuer)
	require.Equal(t, 5*time.Minute, conf.Auth.AccessTokenTTL)
	require.Equal(t, 10*time.Minute, conf.Auth.RefreshTokenTTL)
	require.Equal(t, -2*time.Minute, conf.Auth.TokenOverlap)
	require.True(t, conf.RateLimit.Enabled)
	require.Equal(t, 20.00, conf.RateLimit.PerSecond)
	require.Equal(t, 100, conf.RateLimit.Burst)
	require.Equal(t, 60*time.Minute, conf.RateLimit.TTL)
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

func TestAllowAllOrigins(t *testing.T) {
	conf, err := config.New()
	require.NoError(t, err, "could not create default configuration")
	require.Equal(t, []string{"http://localhost:8000"}, conf.AllowOrigins, "allow origins should be localhost by default")
	require.False(t, conf.AllowAllOrigins(), "expected allow all origins to be false by default")

	conf.AllowOrigins = []string{"https://ensign.rotational.dev", "https://ensign.io"}
	require.False(t, conf.AllowAllOrigins(), "expected allow all origins to be false when allow origins is set")

	conf.AllowOrigins = []string{}
	require.False(t, conf.AllowAllOrigins(), "expected allow all origins to be false when allow origins is empty")

	conf.AllowOrigins = []string{"*"}
	require.True(t, conf.AllowAllOrigins(), "expect allow all origins to be true when * is set")
}

func TestRateLimitConfigValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []config.RateLimitConfig{
			{Enabled: false},
			{
				Enabled:   true,
				Burst:     120,
				PerSecond: 20.00,
				TTL:       5 * time.Minute,
			},
		}

		for i, conf := range tests {
			require.NoError(t, conf.Validate(), "expected rate limit config validation to pass on test case %d", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			conf config.RateLimitConfig
			errs string
		}{
			{
				conf: config.RateLimitConfig{
					Enabled:   true,
					Burst:     0,
					PerSecond: 20.00,
					TTL:       5 * time.Minute,
				},
				errs: "invalid configuration: rateLimit.burst is required but not set",
			},
			{
				conf: config.RateLimitConfig{
					Enabled:   true,
					Burst:     120,
					PerSecond: 0.00,
					TTL:       5 * time.Minute,
				},
				errs: "invalid configuration: rateLimit.perSecond is required but not set",
			},
			{
				conf: config.RateLimitConfig{
					Enabled:   true,
					Burst:     120,
					PerSecond: 20.00,
					TTL:       0,
				},
				errs: "invalid configuration: rateLimit.ttl is required but not set",
			},
		}

		for i, test := range tests {
			err := test.conf.Validate()
			require.EqualError(t, err, test.errs, "expected rate limit config validation error on test case %d", i)
		}
	})
}

func TestAuthConfigValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []config.AuthConfig{
			{
				Audience:        "https://example.com",
				Issuer:          "https://auth.example.com",
				AccessTokenTTL:  5 * time.Minute,
				RefreshTokenTTL: 10 * time.Minute,
				TokenOverlap:    -2 * time.Minute,
			},
			{
				Keys:            map[string]string{},
				Audience:        "https://example.com",
				Issuer:          "https://example.com",
				AccessTokenTTL:  24 * time.Hour,
				RefreshTokenTTL: 48 * time.Hour,
				TokenOverlap:    -12 * time.Hour,
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
					Audience:        "",
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.audience is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        "\x00",
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.audience could not parse audience: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        "https://example.com",
					Issuer:          "",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.issuer is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        "https://example.com",
					Issuer:          "\x00",
					AccessTokenTTL:  5 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    -2 * time.Minute,
				},
				errs: "invalid configuration: auth.issuer could not parse issuer: parse \"\\x00\": net/url: invalid control character in URL",
			},
			{
				conf: config.AuthConfig{
					Audience:        "https://example.com",
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  0 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    0 * time.Minute,
				},
				errs: "invalid configuration: auth.accessTokenTTL is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        "https://example.com",
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  20 * time.Minute,
					RefreshTokenTTL: -10 * time.Minute,
					TokenOverlap:    -5 * time.Minute,
				},
				errs: "invalid configuration: auth.refreshTokenTTL is required but not set",
			},
			{
				conf: config.AuthConfig{
					Audience:        "https://example.com",
					Issuer:          "https://auth.example.com",
					AccessTokenTTL:  20 * time.Minute,
					RefreshTokenTTL: 10 * time.Minute,
					TokenOverlap:    2 * time.Minute,
				},
				errs: "invalid configuration: auth.tokenOverlap must be negative and not exceed the access duration",
			},
			{
				conf: config.AuthConfig{
					Audience:        "https://example.com",
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
