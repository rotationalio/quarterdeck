package server

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
)

func (s *Server) Home(c *gin.Context) {
	// Render the home page with the scene
	// TODO: redirect to user dashboard if HTML request or to API docs if JSON request.
	c.HTML(http.StatusOK, "pages/home/index.html", scene.New(c))
}

func (s *Server) LoginPage(c *gin.Context) {
	prepareURL := &url.URL{Path: "/v1/login"}
	if next := c.Query("next"); next != "" {
		params := url.Values{}
		params.Set("next", next)
		prepareURL.RawQuery = params.Encode()
	}

	ctx := scene.New(c)
	ctx["PrepareLoginURL"] = prepareURL.String()
	c.HTML(http.StatusOK, "auth/login/login.html", ctx)
}
