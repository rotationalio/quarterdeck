package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
)

func (s *Server) Home(c *gin.Context) {
	// Render the home page with the scene
	// TODO: redirect to user dashboard if HTML request or to API docs if JSON request.
	c.HTML(http.StatusOK, "pages/home/index.html", scene.New(c))
}
