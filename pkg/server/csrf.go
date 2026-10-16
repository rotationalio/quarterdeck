package server

import (
	"mime"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.rtnl.ai/gimlet/auth"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

const HeaderCookie = "Cookie"

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
		// Allow token exchange requests from non-browser clients.
		if isStatelessTokenExchangeRequest(c) {
			traceCSRFEvent(c, "csrf.request.bypassed_for_token_exchange")
			c.Next()
			return
		}

		// Otherwise, perform the CSRF check.
		middleware(c)
		if c.Writer.Header().Get(conf.ErrorHeader()) == csrf.ErrorRequestRejected {
			traceCSRFEvent(c, "csrf.request.rejected")
		}
	}
}

// Token exchange is safe to bypass browser CSRF checks only for explicit JSON API
// requests with no browser-origin or ambient-cookie signals. These requests receive
// tokens in their JSON response instead of authentication cookies.
func isStatelessTokenExchangeRequest(c *gin.Context) bool {
	// Only for POST token exchange requests.
	if c.Request.Method != http.MethodPost {
		return false
	}

	// Only allow token exchange request endpoints.
	switch c.FullPath() {
	case "/v1/authenticate", "/v1/reauthenticate":
	default:
		return false
	}

	// Only allow JSON API requests with JSON responses accepted.
	contentType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || contentType != binding.MIMEJSON {
		return false
	}
	if c.GetHeader("Accept") == "" || c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) != binding.MIMEJSON {
		return false
	}

	// Only allow requests with no browser-origin, sec-fetch, or cookies headers.
	for _, header := range []string{
		csrf.HeaderSecFetchSite,
		csrf.HeaderSecFetchDest,
		csrf.HeaderSecFetchMode,
		csrf.HeaderOrigin,
		csrf.HeaderReferer,
		HeaderCookie,
	} {
		if c.GetHeader(header) != "" {
			return false
		}
	}

	// Bypass CSRF is allowed.
	return true
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
