package config

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rs/zerolog"
	"go.rtnl.ai/commo"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/gimlet/secure"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

const (
	Prefix = "QD"
)

type Config struct {
	Maintenance  bool                `default:"false" desc:"if true, quarterdeck will start in maintenance mode"`
	BindAddr     string              `split_words:"true" default:":8888" desc:"the ip address and port to bind the quarterdeck server on"`
	Mode         string              `default:"release" desc:"specify verbosity of logging and error detail (release, debug, test)"`
	LogLevel     logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog   bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	AllowOrigins []string            `split_words:"true" default:"http://localhost:8000" desc:"a list of allowed origins (domains including port) for CORS requests"`
	RateLimit    ratelimit.Config    `split_words:"true"`
	DocsName     string              `split_words:"true" default:"quarterdeck" desc:"the display name for the API docs server in the Swagger app"`
	Org          OrgConfig           `split_words:"true"`
	App          AppConfig           `split_words:"true"`
	Database     DatabaseConfig      `split_words:"true"`
	Auth         AuthConfig          `split_words:"true"`
	CSRF         CSRFConfig          `split_words:"true"`
	Secure       secure.Config       `split_words:"true"`
	Security     SecurityConfig      `split_words:"true"`
	Email        commo.Config        `split_words:"true"`
	processed    bool
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
// Validates each sub-config.
func (c Config) Mark() (_ Config, err error) {
	if err = c.Validate(); err != nil {
		return c, err
	}

	if err = c.RateLimit.Validate(); err != nil {
		return c, err
	}

	if err = c.Org.Validate(); err != nil {
		return c, err
	}

	if err = c.App.Validate(); err != nil {
		return c, err
	}

	if err = c.Auth.Validate(); err != nil {
		return c, err
	}

	if err = c.CSRF.Validate(); err != nil {
		return c, err
	}

	if err = c.Secure.Validate(); err != nil {
		return c, err
	}

	if err = c.Email.Validate(); err != nil {
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

func (c Config) CookieDomains() []string {
	// Strip scheme and port from domains and de-duplicate (in the case of multiple ports)
	domains := map[string]struct{}{}
	for _, origin := range c.AllowOrigins {
		if u, err := url.Parse(origin); err == nil {
			domains[u.Hostname()] = struct{}{}
		}
	}

	// Return just the cookie domains
	out := make([]string, 0, len(domains))
	for domain := range domains {
		out = append(out, domain)
	}
	return out
}
