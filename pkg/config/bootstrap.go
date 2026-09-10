package config

import (
	"net/mail"

	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

type BootstrapConfig struct {
	Enabled   bool            `split_words:"true" default:"false" desc:"if true, quarterdeck will bootstrap itself on startup"`
	Superuser SuperuserConfig `split_words:"true" default:"" desc:"on bootstrap, ensure that the superuser specified is created and verified"`
}

type SuperuserConfig struct {
	Name          string `split_words:"true" default:"" desc:"if email is set and boostrap is enabled, will set the name of the superuser to this field"`
	Email         string `split_words:"true" default:"" desc:"if set and boostrap is enabled, will ensure this user exists with the admin role"`
	Password      string `split_words:"true" default:"" desc:"if email is set and boostrap is enabled, will verify or set the password of the superuser"`
	ForcePassword bool   `split_words:"true" default:"false" desc:"if true, will update the password of the superuser if it already exists instead of verifying it"`
}

func (c BootstrapConfig) Validate() (err error) {
	if c.Enabled {
		// Check if the superuser has an email set.
		if c.Superuser.Enabled() {
			// Validate the email address
			if _, perr := mail.ParseAddress(c.Superuser.Email); perr != nil {
				err = errors.ConfigError(err, errors.ConfigParseError("bootstrap", "superuser.email", perr))
			}

			// Password is required and must have minimum strength level.
			if c.Superuser.Password == "" {
				err = errors.ConfigError(err, errors.RequiredConfig("bootstrap", "superuser.password"))
			} else {
				if _, perr := passwords.Strength(c.Superuser.Password); perr != nil {
					err = errors.ConfigError(err, errors.ConfigParseError("bootstrap", "superuser.password", perr))
				}
			}
		}
	}
	return err
}

// Returns true if the email is set (meaning the superuser job is to be performed).
// NOTE: this does not necessarily mean the bootstrap process is enabled.
func (c SuperuserConfig) Enabled() bool {
	return c.Email != ""
}
