package server

import (
	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/auth"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// Applies global CSRF protection with verified bearer authentication as Gimlet's
// fallback only. Since route-specific authentication runs later, verify tokens
// directly without cookie refresh; protected routes still enforce authentication
// and authorization normally.
func csrfProtection(conf config.CSRFConfig, issuer auth.Authenticator) gin.HandlerFunc {
	opts := append(conf.Options(), csrf.WithFallback(func(c *gin.Context) bool {
		if issuer == nil {
			return false
		}
		token, source, err := auth.GetAccessTokenAndSource(c)
		if err != nil || source != auth.AuthenticationSourceBearer {
			return false
		}
		claims, err := issuer.Verify(token)
		return err == nil && claims != nil
	}))
	return csrf.Middleware(opts...)
}
