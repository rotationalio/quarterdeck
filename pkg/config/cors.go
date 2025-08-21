package config

import (
	"time"

	"github.com/gin-contrib/cors"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
)

var (
	allowedHeaders = [14]string{
		"Origin",
		"Accept",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"X-CSRF-TOKEN",
		htmx.HXBoosted,
		htmx.HXCurrentURL,
		htmx.HXHistoryRestoreRequest,
		htmx.HXPrompt,
		htmx.HXRequest,
		htmx.HXTarget,
		htmx.HXTriggerName,
		htmx.HXTrigger,
	}
	exposeHeaders = [14]string{
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
	// Create a CORS config with the configured allowed origins.
	return cors.Config{
		AllowAllOrigins:        false,
		AllowOrigins:           c.AllowOrigins,
		AllowMethods:           []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:           allowedHeaders[:],
		ExposeHeaders:          exposeHeaders[:],
		AllowCredentials:       true,
		AllowWildcard:          false,
		AllowBrowserExtensions: false,
		AllowWebSockets:        false,
		AllowPrivateNetwork:    true,
		MaxAge:                 12 * time.Hour,
		CustomSchemas:          []string{"honu://"},
	}
}
