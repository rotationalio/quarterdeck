package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/telemetry"
	"go.rtnl.ai/quarterdeck/pkg/web"
)

// Setup the server to run in maintenance mode.
func (s *Server) Maintenance() (err error) {
	// Setup the gin router in maintenance mode.
	// In maintenance mode, there is no logging, though there is still telemetry.
	gin.SetMode(s.conf.Mode)
	s.router = gin.New()
	s.router.RedirectTrailingSlash = true
	s.router.RedirectFixedPath = false
	s.router.HandleMethodNotAllowed = true
	s.router.ForwardedByClientIP = true
	s.router.UseRawPath = false
	s.router.UnescapePathValues = false

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", gin.WrapF(s.Healthz))
	s.router.GET("/livez", gin.WrapF(s.Healthz))
	s.router.GET("/readyz", gin.WrapF(s.Readyz))

	// Create observability middleware
	var observability gin.HandlerFunc
	if observability, err = telemetry.Middleware(); err != nil {
		return err
	}

	// No logging, just telemetry.
	if observability != nil {
		s.router.Use(observability)
	}

	// Recovery middleware to prevent panics from crashing the server.
	s.router.Use(gin.Recovery())

	// Setup maintenance mode HTML template renderer
	if s.router.HTMLRender, err = web.Maintenance(); err != nil {
		return err
	}

	// Status endpoints are available in maintenance mode.
	s.router.GET("/v1/status", s.Status)

	// All other routes return the maintenance handler.
	s.router.NoRoute(ServiceNotAvailable)

	// Create the HTTP server to serve the maintenance mode router.
	s.srv = &http.Server{
		Addr:              s.conf.BindAddr,
		Handler:           s.router,
		ErrorLog:          nil,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}

	return nil
}

func ServiceNotAvailable(c *gin.Context) {
	c.Negotiate(http.StatusServiceUnavailable, gin.Negotiate{
		Offered: []string{binding.MIMEJSON, binding.MIMEHTML},
		Data: &api.StatusReply{
			Status:  serverStatusMaintenance,
			Version: pkg.Version(true),
		},
		HTMLName: "maintenance/index.html",
	})
}
