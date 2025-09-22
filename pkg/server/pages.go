package server

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
)

//===========================================================================
// Authentication Pages
//===========================================================================

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

//===========================================================================
// Workspace Pages
//===========================================================================

func (s *Server) Dashboard(c *gin.Context) {
	// Render the dashboard page with the scene
	// TODO: redirect to user dashboard if HTML request or to API docs if JSON request.
	c.HTML(http.StatusOK, "pages/home/index.html", scene.New(c).ForPage("dashboard"))
}

func (s *Server) WorkspaceSettingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/settings/index.html", scene.New(c).ForPage("settings"))
}

func (s *Server) GovernancePage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/governance/index.html", scene.New(c).ForPage("governance"))
}

func (s *Server) ActivityPage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/activity/index.html", scene.New(c).ForPage("activity"))
}

//===========================================================================
// Profile Pages
//===========================================================================

func (s *Server) ProfilePage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/profile/index.html", scene.New(c))
}

func (s *Server) ProfileSettingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/profile/account.html", scene.New(c))
}

func (s *Server) ProfileDeletePage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/profile/delete.html", scene.New(c))
}

//===========================================================================
// Access Management Pages
//===========================================================================

func (s *Server) APIKeyListPage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/apikeys/list.html", scene.New(c).ForPage("apikeys"))
}
