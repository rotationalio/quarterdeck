package auth

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
)

// Make sure that the issuer implements the Authenticator interface
var _ auth.Authenticator = (*Issuer)(nil)

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
