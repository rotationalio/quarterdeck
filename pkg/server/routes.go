package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/gimlet/cache"
	"go.rtnl.ai/gimlet/csrf"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/o11y"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/gimlet/secure"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth/permissions"
	"go.rtnl.ai/quarterdeck/pkg/docs"
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

		// Security middleware sets security policy headers
		secure.Secure(&s.conf.Secure),

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

		// UI for forgot/reset password
		uio.GET("/forgot-password", s.ForgotPasswordPage)
		uio.GET("/forgot-password/sent", s.ForgotPasswordSentPage)
		uio.GET("/reset-password", s.ResetPasswordPage)

		// The "well known" routes expose client security information and credentials.
		wk := uio.Group("/.well-known")
		{
			wk.GET("/jwks.json", cache.Control(s.issuer), s.JWKS)
			wk.GET("/security.txt", s.SecurityTxt)
			wk.GET("/openid-configuration", s.OpenIDConfiguration)
		}
	}

	// Web UI Routes (Authenticated)
	uia := s.router.Group("", authenticate)
	{
		uia.GET("/", s.Dashboard)
		uia.GET("/settings", s.WorkspaceSettingsPage)
		uia.GET("/governance", s.GovernancePage)
		uia.GET("/activity", s.ActivityPage)

		uia.GET("/apikeys", s.APIKeyListPage)

		profile := uia.Group("/profile")
		{
			profile.GET("", s.ProfilePage)
			profile.GET("/account", s.ProfileSettingsPage)
			profile.GET("/delete", s.ProfileDeletePage)
		}

		// Add documentation routes
		docs.Routes(uia.Group("/docs"))
	}

	// Unauthenticated API Routes (Including Content Negotiated Partials)
	v1o := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1o.GET("/status", s.Status)

		// Documentation routes
		v1o.GET("/docs/openapi.:ext", s.OpenAPI())
		v1o.GET("/docs", s.APIDocs)

		// Authentication endpoints
		v1o.GET("/login", s.PrepareLogin)
		v1o.POST("/login", csrf, s.Login)
		v1o.POST("/authenticate", s.Authenticate)
		v1o.POST("/reauthenticate", s.Reauthenticate)

		// API endpoints for forgot/reset password
		v1o.POST("/forgot-password", s.ForgotPassword)
		v1o.POST("/reset-password", s.ResetPassword)
	}

	// Authenticated API Routes (Including Content Negotiated Partials)
	v1a := s.router.Group("/v1", authenticate)
	{
		// Database Statistics
		v1a.GET("/dbinfo", auth.Authorize(permissions.ConfigView), s.DBInfo)

		// User account Management
		users := v1a.Group("/users")
		{
			users.GET("", s.ListUsers)
			users.POST("", csrf, s.CreateUser)
			users.GET("/:userID", s.UserDetail)
			users.PUT("/:userID", csrf, s.UpdateUser)
			users.DELETE("/:userID", csrf, s.DeleteUser)
			users.POST("/:userID/password", csrf, s.ChangePassword)
		}

		// API Key Management
		apikeys := v1a.Group("/apikeys")
		{
			apikeys.GET("", s.ListAPIKeys)
			apikeys.POST("", csrf, s.CreateAPIKey)
			apikeys.GET("/:keyID", s.APIKeyDetail)
			apikeys.PUT("/:keyID", csrf, s.UpdateAPIKey)
			apikeys.DELETE("/:keyID", csrf, s.DeleteAPIKey)
			apikeys.GET("/:keyID/edit", s.UpdateAPIKeyPreview)
		}

		// OIDC Endpoints
		oidc := v1a.Group("oidc")
		{
			// OIDC UserInfo endpoint.
			// See: https://openid.net/specs/openid-connect-core-1_0.html#UserInfo
			// NOTE: Requires both POST and GET per the spec
			oidc.GET("/userinfo", s.UserInfo)
			oidc.POST("/userinfo", s.UserInfo)

			// OIDC Client Management (Dynamic Client Registration)
			oidcclients := oidc.Group("oidcclients")
			{
				oidcclients.GET("", s.ListOIDCClients)
				oidcclients.POST("", csrf, s.CreateOIDCClient)
				oidcclients.GET("/:id", s.OIDCClientDetail)
				oidcclients.PUT("/:id", csrf, s.UpdateOIDCClient)
				oidcclients.POST("/:id/revoke", csrf, s.RevokeOIDCClient)
				oidcclients.DELETE("/:id", csrf, s.DeleteOIDCClient)
			}
		}
	}

	return nil
}
