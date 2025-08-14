package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/golang-jwt/jwt/v5"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/gimlet/cache"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
)

//===========================================================================
// Authentication Interface
//===========================================================================

// Make sure that the issuer implements the Authenticator interface
var _ auth.Authenticator = (*Issuer)(nil)

func (tm *Issuer) Verify(tks string) (claims *auth.Claims, err error) {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		jwt.WithAudience(tm.conf.Audience...),
		jwt.WithIssuer(tm.conf.Issuer),
	}

	var token *jwt.Token
	if token, err = jwt.ParseWithClaims(tks, &auth.Claims{}, tm.GetKey, opts...); err != nil {
		return nil, err
	}

	var ok bool
	if claims, ok = token.Claims.(*auth.Claims); ok && token.Valid {
		// TODO: add claims specific validation here if needed.
		return claims, nil
	}

	// I haven't figured out a test that will allow us to reach this case; if you pass
	// in a token with a different type of claims, it will return an empty auth.Claims.
	return nil, errors.ErrUnparsableClaims
}

// Make sure that Issuer implements the Unauthenticated interface
var _ auth.Unauthenticator = (*Issuer)(nil)

func (tm *Issuer) NotAuthorized(c *gin.Context) error {
	var loginURL string
	if loginURL = tm.loginURL.Location(c); loginURL == "" {
		return errors.ErrNoLoginURL
	}

	if htmx.IsHTMXRequest(c) {
		htmx.Redirect(c, http.StatusSeeOther, loginURL)
		c.Abort()
		return nil
	}

	// Content Negotiation
	switch accept := c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML); accept {
	case binding.MIMEJSON:
		c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error(errors.ErrAuthRequired))
	case binding.MIMEHTML:
		c.Redirect(http.StatusSeeOther, loginURL)
		c.Abort()
	default:
		return fmt.Errorf("unhandled negotiated content type %q", accept)
	}

	return nil
}

//===========================================================================
// Cache-Control for JWKS (Passthrough)
//===========================================================================

var _ cache.ETagger = (*Issuer)(nil)
var _ cache.Expirer = (*Issuer)(nil)
var _ cache.CacheController = (*Issuer)(nil)

func (tm *Issuer) ETag() string {
	return tm.publicKeys.ETag()
}

func (tm *Issuer) ComputeETag(data []byte) {
	tm.publicKeys.ComputeETag(data)
}

func (tm *Issuer) SetETag(s string) {
	tm.publicKeys.SetETag(s)
}

func (tm *Issuer) LastModified() time.Time {
	return tm.publicKeys.LastModified()
}

func (tm *Issuer) Expires() time.Time {
	return tm.publicKeys.Expires()
}

func (tm *Issuer) Modified(t time.Time, d any) {
	tm.publicKeys.Modified(t, d)
}

func (tm *Issuer) Directives() string {
	return tm.publicKeys.Directives()
}

func (tm *Issuer) SetMaxAge(v any) {
	tm.publicKeys.SetMaxAge(v)
}

func (tm *Issuer) SetSMaxAge(v any) {
	tm.publicKeys.SetSMaxAge(v)
}
