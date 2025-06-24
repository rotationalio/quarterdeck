package server

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/scene"
)

var (
	ErrNotAccepted = errors.New("the accepted formats are not offered by the server")
	ErrMissingID   = errors.New("id required for this resource")
	ErrIDMismatch  = errors.New("resource id does not match target")
	ErrNotFound    = errors.New("resource not found")
	ErrNotAllowed  = errors.New("the requested action is not allowed")
)

// Renders the "not found page"
func (s *Server) NotFound(c *gin.Context) {
	c.Negotiate(http.StatusNotFound, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/404.html",
		HTMLData: scene.New(c).Error(ErrNotFound),
		JSONData: api.NotFound,
	})
}

// Renders the "invalid action page"
func (s *Server) NotAllowed(c *gin.Context) {
	c.Negotiate(http.StatusMethodNotAllowed, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/405.html",
		HTMLData: scene.New(c).Error(ErrNotAllowed),
		JSONData: api.NotAllowed,
	})
}
