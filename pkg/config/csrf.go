package config

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

type CSRFConfig struct {
	CookieTTL time.Duration `split_words:"true" default:"15m" desc:"the duration for which CSRF tokens are valid"`
	Secret    string        `required:"false" desc:"a hexadecimal secret key for signing CSRF tokens; if omitted a random key will be generated"`
}

func (c CSRFConfig) Validate() (err error) {
	if c.CookieTTL <= 0 {
		err = errors.ConfigError(err, errors.RequiredConfig("csrf", "cookieTTL"))
	}

	if c.Secret != "" {
		if _, perr := hex.DecodeString(c.Secret); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("csrf", "secret", perr))
		}
	}

	return err
}

func (c CSRFConfig) GetSecret() []byte {
	var secret []byte
	if c.Secret != "" {
		secret, _ = hex.DecodeString(c.Secret)
	} else {
		secret = make([]byte, 65)
		rand.Read(secret)
	}
	return secret
}
