package server

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/logger"
	"go.rtnl.ai/quarterdeck/pkg/metrics"
	"go.rtnl.ai/quarterdeck/pkg/web"
)

func (s *Server) setupRoutes() (err error) {
	// Setup HTML template renderer
	if s.router.HTMLRender, err = web.HTMLRender(web.Templates()); err != nil {
		return err
	}

	// Create CORS configuration
	corsConf := cors.Config{
		AllowMethods:     []string{"GET", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowOrigins:     s.conf.AllowOrigins,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger(ServiceName),

		// Panic recovery middleware
		gin.Recovery(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(corsConf),

		// Maintenance mode middleware to return unavailable
		s.Maintenance(),
	}

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

	// Prometheus metrics handler added before middleware.
	// Note metrics will be served at /metrics
	metrics.Routes(s.router)

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Error routes for HTMX redirect handling
	s.router.GET("/not-found", s.NotFound)
	s.router.GET("/not-allowed", s.NotAllowed)
	s.router.GET("/error", s.InternalError)

	// Static Files
	s.router.StaticFS("/static", web.Static())

	// TODO: authentication middleware

	// TODO: authorization middleware

	// Web UI Routes (Unauthenticated)
	ui := s.router.Group("")
	{
		ui.GET("/", s.Home)
	}

	// Web UI Routes (Authenticated)

	// API Routes (Including Content Negotiated Partials)
	v1 := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1.GET("/status", s.Status)

		// Database Statistics
		// TODO: ensure this is only available when authenticated
		v1.GET("/dbinfo", s.DBInfo)
		// v1.GET("/dbinfo", authenticate, authorize(permiss.ConfigView), s.DBInfo)
	}

	// The "well known" routes expose client security information and credentials.
	wk := s.router.Group("/.well-known")
	{
		wk.GET("/jwks.json", s.JWKS)
		wk.GET("/security.txt", s.SecurityTxt)
		wk.GET("/openid-configuration", s.OpenIDConfiguration)
	}

	return nil
}
