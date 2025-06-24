package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
)

// If the server is in maintenance mode, aborts the current request and renders the
// maintenance mode page instead. Returns nil if not in maintenance mode.
func (s *Server) Maintenance() gin.HandlerFunc {
	if s.conf.Maintenance {
		return func(c *gin.Context) {
			c.Negotiate(http.StatusServiceUnavailable, gin.Negotiate{
				Offered: []string{binding.MIMEJSON, binding.MIMEHTML},
				Data: &api.StatusReply{
					Status:  "maintenance",
					Version: pkg.Version(false),
					Uptime:  time.Since(s.started).String(),
				},
				HTMLName: "maintenance.html",
			})
			c.Abort()
		}
	}
	return nil
}
