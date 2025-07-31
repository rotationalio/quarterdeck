package config

import (
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rs/zerolog"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

const Prefix = "QD"

type Config struct {
	Maintenance  bool                `default:"false" desc:"if true, quarterdeck will start in maintenance mode"`
	BindAddr     string              `split_words:"true" default:":8888" desc:"the ip address and port to bind the quarterdeck server on"`
	Mode         string              `default:"release" desc:"specify verbosity of logging and error detail (release, debug, test)"`
	LogLevel     logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog   bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	AllowOrigins []string            `split_words:"true" default:"http://localhost:8000" desc:"a list of allowed origins (domains including port) for CORS requests"`
	RateLimit    ratelimit.Config    `split_words:"true"`
	Database     DatabaseConfig
	Auth         AuthConfig
	Security     SecurityConfig
	processed    bool
}

type DatabaseConfig struct {
	URL      string `default:"sqlite3:////data/db/quarterdeck.db" desc:"the database connection URL, including the driver to use."`
	ReadOnly bool   `split_words:"true" default:"false" desc:"if true, quarterdeck will not write to the database, only read from it"`
}

type AuthConfig struct {
	Keys            map[string]string `required:"false" desc:"a map of keyID to key path for JWT signing and verification; if omitted keys will be generated"`
	Audience        string            `default:"http://localhost:8000" desc:"the audience claim for JWT tokens; used to verify the token is intended for this service"`
	Issuer          string            `default:"http://localhost:8888" desc:"the issuer claim for JWT tokens; used to verify the token is issued by this service"`
	AccessTokenTTL  time.Duration     `split_words:"true" default:"1h" desc:"the duration for which access tokens are valid"`
	RefreshTokenTTL time.Duration     `split_words:"true" default:"2h" desc:"the duration for which refresh tokens are valid"`
	TokenOverlap    time.Duration     `split_words:"true" default:"-15m" desc:"the duration before an access token expires that the refresh token is valid"`
}

type SecurityConfig struct {
	TxtPath string `split_words:"true" required:"false" desc:"path to the security.txt file to serve at /.well-known/security.txt"`
}

func New() (conf Config, err error) {
	if err = confire.Process(Prefix, &conf); err != nil {
		return Config{}, err
	}

	conf.processed = true
	return conf, nil
}

// Returns true if the config has not been correctly processed from the environment.
func (c Config) IsZero() bool {
	return !c.processed
}

// Mark a manually constructed config as processed as long as it is valid.
func (c Config) Mark() (_ Config, err error) {
	if err = c.Validate(); err != nil {
		return c, err
	}
	c.processed = true
	return c, nil
}

func (c Config) Validate() (err error) {
	if c.Mode != gin.ReleaseMode && c.Mode != gin.DebugMode && c.Mode != gin.TestMode {
		err = errors.ConfigError(err, errors.InvalidConfig("", "mode", "%q is not a valid gin mode", c.Mode))
	}
	return err
}

func (c Config) GetLogLevel() zerolog.Level {
	return zerolog.Level(c.LogLevel)
}

// Returns true if the allow origins slice contains one entry that is a "*"
func (c Config) AllowAllOrigins() bool {
	if len(c.AllowOrigins) == 1 && c.AllowOrigins[0] == "*" {
		return true
	}
	return false
}

func (c AuthConfig) Validate() (err error) {
	if c.Audience == "" {
		err = errors.ConfigError(err, errors.RequiredConfig("auth", "audience"))
	}

	if _, perr := url.Parse(c.Audience); perr != nil {
		err = errors.ConfigError(err, errors.ConfigParseError("auth", "audience", perr))
	}

	if c.Issuer == "" {
		err = errors.ConfigError(err, errors.RequiredConfig("auth", "issuer"))
	}

	if _, perr := url.Parse(c.Issuer); perr != nil {
		err = errors.ConfigError(err, errors.ConfigParseError("auth", "issuer", perr))
	}

	if c.AccessTokenTTL <= 0 {
		err = errors.ConfigError(err, errors.RequiredConfig("auth", "accessTokenTTL"))
	}

	if c.RefreshTokenTTL <= 0 {
		err = errors.ConfigError(err, errors.RequiredConfig("auth", "refreshTokenTTL"))
	}

	if (c.TokenOverlap*-1) > c.AccessTokenTTL || c.TokenOverlap > 0 {
		err = errors.ConfigError(err, errors.InvalidConfig("auth", "tokenOverlap", "must be negative and not exceed the access duration"))
	}

	return err
}
