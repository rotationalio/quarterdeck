package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
	c.Set(logger.LogLevelKey, slog.LevelDebug)

	healthy := s.IsHealthy()
	ready := s.IsReady()

	var state string
	switch {
	case s.conf.Maintenance:
		state = serverStatusMaintenance
	case healthy && ready:
		state = serverStatusOK
	case healthy && !ready:
		state = serverStatusNotReady
	case !healthy:
		state = serverStatusUnhealthy
	}

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
	c.Set(logger.LogLevelKey, slog.LevelDebug)

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
