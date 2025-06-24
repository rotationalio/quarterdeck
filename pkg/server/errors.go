package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/scene"
)

// Renders the "not found page"
func (s *Server) NotFound(c *gin.Context) {
	c.Negotiate(http.StatusNotFound, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/404.html",
		HTMLData: scene.New(c).Error(errors.ErrNotFound),
		JSONData: api.NotFound,
	})
}

// Renders the "invalid action page"
func (s *Server) NotAllowed(c *gin.Context) {
	c.Negotiate(http.StatusMethodNotAllowed, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/405.html",
		HTMLData: scene.New(c).Error(errors.ErrNotAllowed),
		JSONData: api.NotAllowed,
	})
}
