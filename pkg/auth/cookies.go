package auth

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
	CookieMaxAgeBuffer = 600 * time.Second
)

const (
	localhost = "localhost"
	localTLD  = ".local"
)

// SetAuthCookies is a helper function to set authentication cookies on a gin request.
// The access token cookie (access_token) is an http only cookie that expires when the
// access token expires. The refresh token cookie is not an http only cookie (it can be
// accessed by client-side scripts) and it expires when the refresh token expires. Both
// cookies require https and will not be set (silently) over http connections.
//
// The cookie domains are set based on the access token audience (the refresh token
// audience is the issuer so must duplicate the access token audience).
func SetAuthCookies(c *gin.Context, accessToken, refreshToken string) (err error) {
	// Parse the acccess token to get the audience and expiration time.
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(accessToken); err != nil {
		return errors.Fmt("could not parse access token: %w", err)
	}

	// Get the cookie domains from the audience of the access token.
	cookieDomains := make([]string, 0, len(claims.Audience))
	for _, audience := range claims.Audience {
		url, err := url.Parse(audience)
		if err != nil {
			return errors.Fmt("could not parse audience %s: %w", audience, err)
		}
		cookieDomains = append(cookieDomains, url.Hostname())
	}

	// Compute the access token max age based on the expiration time of the access token
	// The access token cannot be accessed by javascript so it is set as an http only cookie.
	accessMaxAge := int(time.Until(claims.ExpiresAt.Time.Add(CookieMaxAgeBuffer)).Seconds())
	for _, domain := range cookieDomains {
		SetSecureCookie(c, AccessTokenCookie, accessToken, accessMaxAge, domain, true)
	}

	// Parse the refresh token to get the expiration time.
	var refreshExpires time.Time
	if refreshExpires, err = ExpiresAt(refreshToken); err != nil {
		return errors.Fmt("could not parse refresh token: %w", err)
	}

	// Set the refresh token cookie; httpOnly is false it can be accessed by javascript.
	refreshMaxAge := int(time.Until(refreshExpires.Add(CookieMaxAgeBuffer)).Seconds())
	for _, domain := range cookieDomains {
		SetSecureCookie(c, RefreshTokenCookie, refreshToken, refreshMaxAge, domain, false)
	}

	return nil
}

func SetSecureCookie(c *gin.Context, name, value string, maxAge int, domain string, httpOnly bool) {
	secure := !IsLocalhost(domain) // Secure is true unless the domain is localhost or ends in .local
	c.SetCookie(name, value, maxAge, "/", domain, secure, httpOnly)
}

// ClearAuthCookies is a helper function to clear authentication cookies on a gin
// request to effectively log out a user.
func ClearAuthCookies(c *gin.Context, audience []string) {
	// TODO: the cookie domains could probably be cached since they won't change.
	cookieDomains := make([]string, 0, len(audience))
	for _, audience := range audience {
		// NOTE: ignoring errors here since those cookies wouldn't have been set anyway
		if url, err := url.Parse(audience); err == nil {
			cookieDomains = append(cookieDomains, url.Hostname())
		}
	}

	for _, domain := range cookieDomains {
		ClearSecureCookie(c, AccessTokenCookie, domain, true)
		ClearSecureCookie(c, RefreshTokenCookie, domain, false)
	}
}

func ClearSecureCookie(c *gin.Context, name, domain string, httpOnly bool) {
	// Secure is true unless the domain is localhost or ends in .local
	secure := !IsLocalhost(domain)

	// Clear the cookie by setting its expiration to one second ago
	c.SetCookie(name, "", -1, "/", domain, secure, httpOnly)
}

func IsLocalhost(domain string) bool {
	return domain == localhost || strings.HasSuffix(domain, localTLD)
}
