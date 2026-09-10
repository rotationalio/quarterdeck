package config_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

func TestBootstrapConfigValidate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []config.BootstrapConfig{
			{},
			{
				Enabled: false,
				Superuser: config.SuperuserConfig{
					Email:    "test@example.com",
					Password: "",
				},
			},
			{
				Enabled: true,
			},
			{
				Enabled: true,
				Superuser: config.SuperuserConfig{
					Email:    "test@example.com",
					Password: "theEagleFliesAtHalfPast12",
				},
			},
			{
				Enabled: true,
				Superuser: config.SuperuserConfig{
					Name:     "Rotational Tester",
					Email:    "test@example.com",
					Password: "theEagleFliesAtHalfPast12",
				},
			},
			{
				Enabled: true,
				Superuser: config.SuperuserConfig{
					Name:          "Rotational Tester",
					Email:         "test@example.com",
					Password:      "theEagleFliesAtHalfPast12",
					ForcePassword: true,
				},
			},
			{
				Enabled: true,
				Superuser: config.SuperuserConfig{
					Name:          "Rotational Tester",
					Email:         "test@example.com",
					Password:      "theEagleFliesAtHalfPast12",
					ForcePassword: true,
				},
			},
		}

		for i, conf := range tests {
			require.NoError(t, conf.Validate(), "expected bootstrap config validation to pass on test case %d", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			conf config.BootstrapConfig
			errs string
		}{
			{
				conf: config.BootstrapConfig{
					Enabled: true,
					Superuser: config.SuperuserConfig{
						Email: "test@example.com",
					},
				},
				errs: "invalid configuration: bootstrap.superuser.password is required but not set",
			},
			{
				conf: config.BootstrapConfig{
					Enabled: true,
					Superuser: config.SuperuserConfig{
						Email:    "test@example.com",
						Password: "password",
					},
				},
				errs: "invalid configuration: bootstrap.superuser.password could not parse superuser.password: password must contain uppercase letters, lowercase letters, numbers, and special characters",
			},
		}

		for i, test := range tests {
			err := test.conf.Validate()
			require.EqualError(t, err, test.errs, "expected bootstrap config validation error on test case %d", i)
		}
	})
}
