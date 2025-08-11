package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/gimlet/csrf"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/o11y"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth/permissions"
	"go.rtnl.ai/quarterdeck/pkg/web"
)

func (s *Server) setupRoutes() (err error) {
	// Setup HTML template renderer
	if s.router.HTMLRender, err = web.HTMLRender(web.Templates()); err != nil {
		return err
	}

	// Create rate limiting middleware
	var throttle gin.HandlerFunc
	if throttle, err = ratelimit.RateLimit(&s.conf.RateLimit); err != nil {
		return err
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.Logger(ServiceName, pkg.Version(true), true),

		// Panic recovery middleware
		gin.Recovery(),

		// Maintenance mode middleware to make system unavailable while running
		s.Maintenance(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(s.conf.CORS()),

		// Rate limiting middleware to prevent abuse of the API
		throttle,
	}

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

	// Prometheus metrics handler added before middleware.
	// Note metrics will be served at /metrics
	o11y.Routes(s.router)

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// Instantiate per-route middleware
	var authenticate gin.HandlerFunc
	if authenticate, err = auth.Authenticate(s.issuer); err != nil {
		return err
	}

	// CSRF protection middleware
	csrf := csrf.DoubleCookie(s.csrf)

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Error routes for HTMX redirect handling
	s.router.GET("/not-found", s.NotFound)
	s.router.GET("/not-allowed", s.NotAllowed)
	s.router.GET("/error", s.InternalError)

	// Static Files
	s.router.StaticFS("/static", web.Static())

	// Web UI Routes (Unauthenticated)
	uio := s.router.Group("")
	{
		uio.GET("/login", s.LoginPage)
		uio.GET("/logout", s.Logout)
	}

	// Web UI Routes (Authenticated)
	uia := s.router.Group("")
	{
		uia.GET("/", s.Home)
	}

	// The "well known" routes expose client security information and credentials.
	wk := s.router.Group("/.well-known")
	{
		wk.GET("/jwks.json", s.JWKS)
		wk.GET("/security.txt", s.SecurityTxt)
		wk.GET("/openid-configuration", s.OpenIDConfiguration)
	}

	// API Routes (Including Content Negotiated Partials)
	v1 := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1.GET("/status", s.Status)

		// Database Statistics
		v1.GET("/dbinfo", authenticate, auth.Authorize(permissions.ConfigView), s.DBInfo)

		// Authentication endpoints
		v1.GET("/login", s.PrepareLogin)
		v1.POST("/login", csrf, s.Login)
	}

	return nil
}
