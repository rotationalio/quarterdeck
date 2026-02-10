package config

import (
	"net/url"
	"sync"

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

var (
	conf    *Config
	confErr error
	once    sync.Once
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

// Get the configuration being used globally by the codebase. In normal operation, the
// config is set by [server.New] and then used by other modules and packages.
func Get() (Config, error) {
	once.Do(func() {
		var config Config
		if config, confErr = New(); confErr != nil {
			return
		}

		if confErr = config.Validate(); confErr != nil {
			return
		}

		conf = &config
	})

	if conf == nil {
		return Config{}, confErr
	}
	return *conf, confErr
}

// Set the configuration being used globally by the codebase. In normal operation, the
// config is set by [server.New] but it can be set manually by testing code as well.
func Set(config Config) {
	// Mark the once block as done
	once.Do(func() {})

	// Do not allow setting a zero-valued config.
	if config.IsZero() {
		confErr = errors.ConfigError(confErr, errors.InvalidConfig("", "config", "cannot set a zero-valued config"))
		return
	}

	// If you try to set an invalid config then the config will not be set and the error
	// will be returned when you try to get the config.
	if confErr = config.Validate(); confErr != nil {
		return
	}

	// Set the config and error
	conf = &config
	confErr = nil
}

// Force a reload of the configuration from the environment.
func Reload() (conf Config, err error) {
	once = sync.Once{}
	return Get()
}

// Resets the config module (only used by tests).
func reset() {
	once = sync.Once{}
	conf = nil
	confErr = nil
}

// New processes the configuration from the environment and marks it as ready for use.
// It is used by external callers to create a new config to pass to the server. Internal
// code should use [Get] to ensure the config is properly initialized.
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
