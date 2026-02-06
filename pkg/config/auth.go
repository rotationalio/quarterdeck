package config

import (
	"net/url"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/redirect"
)

const (
	LoginPath         = "/login"
	ResetPasswordPath = "/reset-password"
	LoginRedirectPath = "/"
)

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
