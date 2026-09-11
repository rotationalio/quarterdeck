package server

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/gimlet/cache"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/ratelimit"
	"go.rtnl.ai/gimlet/secure"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth/permissions"
	"go.rtnl.ai/quarterdeck/pkg/docs"
	"go.rtnl.ai/quarterdeck/pkg/telemetry"
	"go.rtnl.ai/quarterdeck/pkg/web"
)

func (s *Server) setupRoutes() (err error) {
	// Setup HTML template renderer
	if s.router.HTMLRender, err = web.HTMLRender(web.Templates()); err != nil {
		return err
	}

	// Create observability middleware
	var observability gin.HandlerFunc
	if observability, err = telemetry.Middleware(); err != nil {
		return err
	}

	// Create rate limiting middleware
	var throttle gin.HandlerFunc
	if throttle, err = ratelimit.RateLimit(&s.conf.RateLimit); err != nil {
		return err
	}

	// Instantiate CSRF middleware before assembling the application middleware
	// chain. It skips safe methods internally and protects every unsafe route.
	csrfMiddleware, csrfPage := s.csrfHandlers()

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// o11y should be on the outside so we can record the correct latency of requests
		// NOTE: o11y panics will not recover due to middleware ordering.
		observability,

		// Panic recovery middleware
		gin.Recovery(),

		// Optional logging middleware
		logger.Logger(ServiceName, pkg.Version(true)),

		// Security middleware sets security policy headers
		secure.Secure(&s.conf.Secure),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(s.conf.CORS()),

		// Rate limiting middleware to prevent abuse of the API
		throttle,

		// CSRF protection is global; safe methods are intentionally ignored.
		csrfMiddleware,
	}

	// Kubernetes liveness probes are intentionally outside application
	// middleware, including CSRF protection.
	s.router.GET("/healthz", gin.WrapF(s.Healthz))
	s.router.GET("/livez", gin.WrapF(s.Healthz))
	s.router.GET("/readyz", gin.WrapF(s.Readyz))

	// Add the middleware to the router before registering application routes.
	// This ensures the unauthenticated CSRF bootstrap uses the same CORS policy
	// as the API routes without applying CSRF middleware to the probes.
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

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Error routes for HTMX redirect handling
	s.router.GET("/not-found", s.NotFound)
	s.router.GET("/not-allowed", s.NotAllowed)
	s.router.GET("/error", s.InternalError)

	// Static Files
	s.router.StaticFS("/static", web.Static())

	// CSRF bootstrap for direct browser clients. This endpoint is intentionally
	// unauthenticated and is protected by the global exact-origin CORS policy.
	s.router.GET("/csrf", s.CSRFToken)

	// Web UI Routes (Unauthenticated)
	uio := s.router.Group("")
	{
		uio.GET("/login", s.LoginPage)
		uio.POST("/logout", s.Logout)

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
	uia := s.router.Group("", authenticate, csrfPage)
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
		v1o.POST("/login", s.Login)
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
			users.POST("", s.CreateUser)
			users.GET("/:userID", s.UserDetail)
			users.PUT("/:userID", s.UpdateUser)
			users.DELETE("/:userID", s.DeleteUser)
			users.POST("/:userID/password", s.ChangePassword)
		}

		// API Key Management
		apikeys := v1a.Group("/apikeys")
		{
			apikeys.GET("", s.ListAPIKeys)
			apikeys.POST("", s.CreateAPIKey)
			apikeys.GET("/:keyID", s.APIKeyDetail)
			apikeys.PUT("/:keyID", s.UpdateAPIKey)
			apikeys.DELETE("/:keyID", s.DeleteAPIKey)
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
				oidcclients.POST("", s.CreateOIDCClient)
				oidcclients.GET("/:id", s.OIDCClientDetail)
				oidcclients.PUT("/:id", s.UpdateOIDCClient)
				oidcclients.DELETE("/:id", s.DeleteOIDCClient)
			}
		}
	}

	return nil
}
