package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/store"
)

const (
	serverStatusOK          = "ok"
	serverStatusNotReady    = "not ready"
	serverStatusUnhealthy   = "unhealthy"
	serverStatusMaintenance = "maintenance"
)

// Status reports the version and uptime of the server
func (s *Server) Status(c *gin.Context) {
	// Reduce logging verbosity for the status endpoint
	c.Set(logger.LogLevelKey, zerolog.DebugLevel)

	var state string
	s.RLock()
	switch {
	case s.conf.Maintenance:
		state = serverStatusMaintenance
	case s.healthy && s.ready:
		state = serverStatusOK
	case s.healthy && !s.ready:
		state = serverStatusNotReady
	case !s.healthy:
		state = serverStatusUnhealthy
	}
	s.RUnlock()

	c.JSON(http.StatusOK, &api.StatusReply{
		Status:  state,
		Version: pkg.Version(false),
		Uptime:  time.Since(s.started).String(),
	})
}

// DBInfo reports the database connection status and information if available,
// otherwise returns a 501 Not Implemented http error.
func (s *Server) DBInfo(c *gin.Context) {
	// Reduce logging verbosity for the dbinfo endpoint
	c.Set(logger.LogLevelKey, zerolog.DebugLevel)

	// Reduce logging verbosity for the db info endpoint
	db, ok := s.store.(store.Stats)
	if !ok {
		c.JSON(http.StatusNotImplemented, api.Error("store does not implement stats"))
		return
	}

	// Render the database statistics
	stats := db.Stats()
	c.JSON(http.StatusOK, &api.DBInfo{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration.String(),
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	})
}

// Healthz is used to alert k8s to the health/liveness status of the server.
func (s *Server) Healthz(c *gin.Context) {
	// Reduce logging verbosity for probe endpoints
	c.Set(logger.LogLevelKey, zerolog.TraceLevel)

	s.RLock()
	healthy := s.healthy
	s.RUnlock()

	if !healthy {
		c.Data(http.StatusServiceUnavailable, "text/plain", []byte(serverStatusUnhealthy))
		return
	}

	c.Data(http.StatusOK, "text/plain", []byte(serverStatusOK))
}

// Readyz is used to alert k8s to the readiness status of the server.
func (s *Server) Readyz(c *gin.Context) {
	// Reduce logging verbosity for probe endpoints
	c.Set(logger.LogLevelKey, zerolog.TraceLevel)

	s.RLock()
	ready := s.ready
	s.RUnlock()

	if !ready {
		c.Data(http.StatusServiceUnavailable, "text/plain", []byte(serverStatusNotReady))
		return
	}

	c.Data(http.StatusOK, "text/plain", []byte(serverStatusOK))
}
