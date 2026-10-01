package server

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.rtnl.ai/gimlet/auth"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// Applies global CSRF protection with verified bearer authentication as Gimlet's
// fallback only. Since route-specific authentication runs later, verify tokens
// directly without cookie refresh; protected routes still enforce authentication
// and authorization normally.
func csrfProtection(conf config.CSRFConfig, issuer auth.Authenticator) gin.HandlerFunc {
	// Add fallback to allow verified beareer tokens.
	opts := append(conf.Options(), csrf.WithFallback(func(c *gin.Context) bool {
		if issuer == nil {
			return false
		}
		token, source, err := auth.GetAccessTokenAndSource(c)
		if err != nil || source != auth.AuthenticationSourceBearer {
			return false
		}
		claims, err := issuer.Verify(token)
		if err != nil || claims == nil {
			return false
		}
		traceCSRFEvent(c, "csrf.request.accepted_by_fallback")
		return true
	}))

	// The middleware runs then logs any rejections.
	middleware := csrf.Middleware(opts...)
	return func(c *gin.Context) {
		middleware(c)
		if c.Writer.Header().Get(conf.ErrorHeader()) == csrf.ErrorRequestRejected {
			traceCSRFEvent(c, "csrf.request.rejected")
		}
	}
}

// Adds a CSRF decision event to the active request span when tracing is enabled.
func traceCSRFEvent(c *gin.Context, name string) {
	span := trace.SpanFromContext(c.Request.Context())
	if !span.IsRecording() {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("http.request.method", c.Request.Method),
	}
	if route := c.FullPath(); route != "" {
		attrs = append(attrs, attribute.String("http.route", route))
	}
	span.AddEvent(name, trace.WithAttributes(attrs...))
}
