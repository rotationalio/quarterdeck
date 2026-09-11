package config

import (
	"time"

	"github.com/gin-contrib/cors"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
)

// CSRFRetryHeader is the header used to request a CSRF token retry.
// TODO: Use the Gimlet namespace retry header when Gimlet supports it.
const CSRFRetryHeader = "X-Quarterdeck-CSRF-Retry"

var (
	allowedHeaders = []string{
		"Origin",
		"Accept",
		"Content-Length",
		"Content-Type",
		"Authorization",
		htmx.HXCurrentURL,
		htmx.HXRequest,
		htmx.HXTarget,
		htmx.HXTrigger,
	}
	exposeHeaders = []string{
		"Content-Length",
		"Content-Type",
		"Access-Control-Allow-Origin",
		htmx.HXLocation,
		htmx.HXPushURL,
		htmx.HXRedirect,
		htmx.HXRefresh,
		htmx.HXReplaceURL,
		htmx.HXReswap,
		htmx.HXRetarget,
		htmx.HXReselect,
		htmx.HXTriggerAfterSettle,
		htmx.HXTriggerAfterSwap,
		htmx.HXTrigger,
	}
)

func (c Config) CORS() cors.Config {
	// Derive the request and error header names from Gimlet's namespace helper.
	names := c.CSRF.Names()
	requestHeaders := append([]string{}, allowedHeaders...)
	requestHeaders = append(requestHeaders, names.Header, CSRFRetryHeader)
	responseHeaders := append([]string{}, exposeHeaders...)
	responseHeaders = append(responseHeaders, names.ErrorHeader)

	// Create a credentialed CORS config with exact configured origins.
	return cors.Config{
		AllowAllOrigins:        false,
		AllowOrigins:           c.AllowOrigins,
		AllowMethods:           []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:           requestHeaders,
		ExposeHeaders:          responseHeaders,
		AllowCredentials:       true,
		AllowWildcard:          false,
		AllowBrowserExtensions: false,
		AllowWebSockets:        false,
		AllowPrivateNetwork:    false,
		MaxAge:                 12 * time.Hour,
		CustomSchemas:          []string{"honu://"},
	}
}
