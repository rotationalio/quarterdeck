package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
)

// Logs the error with c.Error and negotiates the response. If HTML is requested by the
// Accept header, then a 500 error page is displayed. If JSON is requested, then the
// error is rendered as a JSON response. If a non error is passed as err then no error
// is logged to the context and it is treated as a message to the user.
func (s *Server) Error(c *gin.Context, err error) {
	if err != nil {
		c.Error(err)
	}

	c.Negotiate(http.StatusInternalServerError, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/500.html",
		HTMLData: scene.New(c).Error(err),
		JSONData: api.Error(err),
	})
}

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

// Renders the "internal server error page"
// TODO: handle htmx error redirects with error message in the context.
func (s *Server) InternalError(c *gin.Context) {
	c.Negotiate(http.StatusInternalServerError, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/500.html",
		HTMLData: scene.New(c).Error(nil),
		JSONData: api.InternalError,
	})
}
