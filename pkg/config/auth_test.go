package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

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
