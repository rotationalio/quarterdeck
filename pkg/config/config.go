package config

import (
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rotationalio/confire"
	"github.com/rs/zerolog"
	"go.rtnl.ai/commo"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/gimlet/secure"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/redirect"
)

const (
	Prefix            = "QD"
	LoginPath         = "/login"
	ResetPasswordPath = "/reset-password"
	LoginRedirectPath = "/"
)

type Config struct {
	Maintenance  bool                `default:"false" desc:"if true, quarterdeck will start in maintenance mode"`
	BindAddr     string              `split_words:"true" default:":8888" desc:"the ip address and port to bind the quarterdeck server on"`
	Mode         string              `default:"release" desc:"specify verbosity of logging and error detail (release, debug, test)"`
	LogLevel     logger.LevelDecoder `split_words:"true" default:"info" desc:"specify the verbosity of logging (trace, debug, info, warn, error, fatal panic)"`
	ConsoleLog   bool                `split_words:"true" default:"false" desc:"if true logs colorized human readable output instead of json"`
	AllowOrigins []string            `split_words:"true" default:"http://localhost:8000" desc:"a list of allowed origins (domains including port) for CORS requests"`
	DocsName     string              `split_words:"true" default:"quarterdeck" desc:"the display name for the API docs server in the Swagger app"`
	SupportEmail string              `split_words:"true" default:"" desc:"an email address that a user may email for technical support, by default support@ISSUER.TLD"`
	ServiceName  string              `split_words:"true" env:"OTEL_SERVICE_NAME" desc:"override the default name of the service, used for logging and telemetry"`
	ServiceAddr  string              `split_words:"true" env:"GIMLET_OTEL_SERVICE_ADDR" desc:"the primary server name if it is known. E.g. the server name directive in an Nginx config. Should include host identifier and port if it is used; empty if not known."`
	RateLimit    ratelimit.Config    `split_words:"true"`
	Database     DatabaseConfig      `split_words:"true"`
	Auth         AuthConfig          `split_words:"true"`
	CSRF         CSRFConfig          `split_words:"true"`
	Secure       secure.Config       `split_words:"true"`
	Security     SecurityConfig      `split_words:"true"`
	Email        commo.Config        `split_words:"true"`
	UserSync     UserSyncConfig      `split_words:"true"`
	processed    bool
}

type DatabaseConfig struct {
	URL      string `default:"sqlite3:////data/db/quarterdeck.db" desc:"the database connection URL, including the driver to use."`
	ReadOnly bool   `split_words:"true" default:"false" desc:"if true, quarterdeck will not write to the database, only read from it"`
}

type AuthConfig struct {
	Keys                   map[string]string `required:"false" desc:"a map of keyID to key path for JWT signing and verification; if omitted keys will be generated"`
	Audience               []string          `default:"http://localhost:8000" desc:"the audience claim for JWT tokens; used to verify the token is intended for this service"`
	Issuer                 string            `default:"http://localhost:8888" desc:"the issuer claim for JWT tokens; used to verify the token is issued by this service"`
	LoginURL               string            `split_words:"true" default:"" desc:"specify an alternate login URL, by default it is the issuer + /login"`
	ResetPasswordURL       string            `split_words:"true" default:"" desc:"specify an alternate reset-pasword URL, by default it is the issuer + /reset-password"`
	LogoutRedirect         string            `split_words:"true" default:"" desc:"specify an alternate URL to redirect the user to after logout, by default it is the login url"`
	AuthenticateRedirect   string            `split_words:"true" default:"/" desc:"specify a location to redirect the user to after successful authentication"`
	ReauthenticateRedirect string            `split_words:"true" default:"/" desc:"specify a location to redirect the user to after successful re-authentication"`
	LoginRedirect          string            `split_words:"true" default:"/" desc:"specify a location to redirect the user to after successful login"`
	AccessTokenTTL         time.Duration     `split_words:"true" default:"1h" desc:"the duration for which access tokens are valid"`
	RefreshTokenTTL        time.Duration     `split_words:"true" default:"2h" desc:"the duration for which refresh tokens are valid"`
	TokenOverlap           time.Duration     `split_words:"true" default:"-15m" desc:"the duration before an access token expires that the refresh token is valid"`
}

type CSRFConfig struct {
	CookieTTL time.Duration `split_words:"true" default:"15m" desc:"the duration for which CSRF tokens are valid"`
	Secret    string        `required:"false" desc:"a hexadecimal secret key for signing CSRF tokens; if omitted a random key will be generated"`
}

type SecurityConfig struct {
	TxtPath string `split_words:"true" required:"false" desc:"path to the security.txt file to serve at /.well-known/security.txt"`
}

// Configures user syncing. Quarterdeck will attempt to post any new or modified
// users to each of the endpoints provided. A create/update for a user will be
// sent via an HTTP POST with the [api.User] JSON to the endpoints, and when a
// user is deleted the user's ID will be appended to each endpoint as a path
// parameter and sent via an HTTP DELETE. This is a "best effort" functionality,
// so failures will be logged but not handled at that time.
type UserSyncConfig struct {
	WebhookEndpoints []string `split_words:"true" required:"false" desc:"webhook endpoints for the applications that wish to recieve user sync events"`
}

func New() (conf Config, err error) {
	if err = confire.Process(Prefix, &conf); err != nil {
		return Config{}, err
	}

	// If the support email is not set, set it to support@ISSUER.TLD
	if conf.SupportEmail == "" {
		issuerURL, _ := url.Parse(conf.Auth.Issuer)
		conf.SupportEmail = "support@" + issuerURL.Hostname()
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

	if err = c.RateLimit.Validate(); err != nil {
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

	if err = c.UserSync.Validate(); err != nil {
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

func (c *AuthConfig) Validate() (err error) {
	if len(c.Audience) == 0 {
		err = errors.ConfigError(err, errors.RequiredConfig("auth", "audience"))
	}

	for i, aud := range c.Audience {
		if audURL, perr := url.Parse(aud); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("auth", "audience", perr))
		} else {
			if audURL.Path == "/" {
				audURL.Path = ""
				c.Audience[i] = audURL.String()
			}
		}
	}

	if perr := c.validateIssuer(); perr != nil {
		err = errors.ConfigError(err, perr)
	} else {
		// We know the issuer URL is valid so create the URL to resolve references
		issuerURL, _ := url.Parse(c.Issuer)
		origin := redirect.MustNew(c.Issuer)

		// Remove trailing spaces from issuer
		if issuerURL.Path == "/" {
			issuerURL.Path = ""
			c.Issuer = issuerURL.String()
		}

		// LoginURL must be an absolute URL with the scheme and host, even if it matches
		// the issuer URL scheme and host. If empty, it is derived from the issuer URL.
		if c.LoginURL == "" {
			c.LoginURL = issuerURL.ResolveReference(&url.URL{Path: LoginPath}).String()
		}

		// Ensure the login URL can be used for login redirects.
		if _, perr := redirect.Login(c.LoginURL); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("auth", "loginURL", perr))
		}

		// ResetPasswordURL must be an absolute URL with the scheme and host, even if it matches
		// the issuer URL scheme and host. If empty, it is derived from the issuer URL.
		if c.ResetPasswordURL == "" {
			c.ResetPasswordURL = issuerURL.ResolveReference(&url.URL{Path: ResetPasswordPath}).String()
		}

		// If the LogoutRedirect is not set, use the LoginURL.
		if c.LogoutRedirect == "" {
			c.LogoutRedirect = c.LoginURL
		}

		// Normalize the LogoutRedirect with respect to the origin
		if _, perr := origin.Location(c.LogoutRedirect); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("auth", "logoutRedirect", perr))
		}

		// If LoginRedirect is not set, use the default login redirect path
		if c.LoginRedirect == "" {
			c.LoginRedirect = issuerURL.ResolveReference(&url.URL{Path: LoginRedirectPath}).String()
		}

		// Normalize the LoginRedirect with respect to the origin
		if _, perr := origin.Location(c.LoginRedirect); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("auth", "loginRedirect", perr))
		}

		// If AuthenticateRedirect is not set, use the default login redirect path
		if c.AuthenticateRedirect == "" {
			c.AuthenticateRedirect = issuerURL.ResolveReference(&url.URL{Path: LoginRedirectPath}).String()
		}

		// Normalize the AuthenticateRedirect with respect to the origin
		if _, perr := origin.Location(c.AuthenticateRedirect); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("auth", "authenticateRedirect", perr))
		}

		// If ReauthenticateRedirect is not set, use the default login redirect path
		if c.ReauthenticateRedirect == "" {
			c.ReauthenticateRedirect = issuerURL.ResolveReference(&url.URL{Path: LoginRedirectPath}).String()
		}

		// Normalize the ReauthenticateRedirect with respect to the origin
		if _, perr := origin.Location(c.ReauthenticateRedirect); perr != nil {
			err = errors.ConfigError(err, errors.ConfigParseError("auth", "reauthenticateRedirect", perr))
		}
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

func (c AuthConfig) validateIssuer() *errors.InvalidConfiguration {
	if c.Issuer == "" {
		return errors.RequiredConfig("auth", "issuer")
	}

	if _, err := redirect.Login(c.Issuer); err != nil {
		return errors.InvalidConfig("auth", "issuer", "cannot be used as a login url: %s", err)
	}

	return nil
}

// Returns the ResetPasswordURL as a [url.URL].
func (c AuthConfig) GetResetPasswordURL() *url.URL {
	u, _ := url.Parse(c.Issuer)
	u.Path = ResetPasswordPath
	return u
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

func (c UserSyncConfig) Validate() (err error) {
	for idx, ep := range c.WebhookEndpoints {
		if _, err := url.Parse(ep); err != nil {
			return errors.ConfigError(err, errors.InvalidConfig("userSync", "webhookEndpoints", "url index %d is unparseable", idx))
		}
	}

	return nil
}

// Returns the webhook endpoints as [url.URL]s.
func (c UserSyncConfig) WebhookURLs() (urls []*url.URL) {
	urls = make([]*url.URL, 0)
	for _, ep := range c.WebhookEndpoints {
		// We can ignore the error here because we have previously validated
		// these URLs as parseable
		u, _ := url.Parse(ep)
		urls = append(urls, u)
	}

	return urls
}
