package config_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

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
