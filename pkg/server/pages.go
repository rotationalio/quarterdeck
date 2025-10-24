package server

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth"
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
// Forgot/Reset Password Pages
//===========================================================================

// ForgotPasswordPage displays the reset password form for the UI so that the user can
// enter their email address and receive a password reset link.
func (s *Server) ForgotPasswordPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/reset/forgot.html", scene.New(c))
}

// ForgotPasswordSentPage displays the success page for the reset password
// request. Rather than using an HTMX partial, we redirect the user to this page to
// ensure they close the window (e.g. if they were logged in) and to prevent a conflict
// when cookies are reset during the password reset process.
func (s *Server) ForgotPasswordSentPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/reset/sent.html", scene.New(c))
}

// ResetPasswordPage allows the user to enter a new password if the reset password link
// is verified and change their password as necessary.
func (s *Server) ResetPasswordPage(c *gin.Context) {
	// Read the token string from the URL parameters.
	in := &api.URLVerification{}
	if err := c.BindQuery(in); err != nil {
		// Debug an error here but don't worry about erroring; the token will be
		// blank and will cause a validation error when the form is submitted.
		log.Debug().Err(err).Msg("could not parse query string")
	}

	// Set the token into a cookie so that it can be parsed when the form is submitted.
	// A cookie is more secure than using a hidden form because it cannot be accessed
	// by XSS attacks (though it could be fetched by the window.location object).
	// NOTE: no verification is performed here, just on reset-password.
	auth.SetResetPasswordTokenCookie(c, in.Token)

	// Render the verify and change page
	c.HTML(http.StatusOK, "auth/reset/password.html", scene.New(c))
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
	// Use a scene that has the "forgot password" page URL
	c.HTML(http.StatusOK, "pages/profile/account.html", scene.New(c).WithForgotPasswordURL())
}

func (s *Server) ProfileDeletePage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/profile/delete.html", scene.New(c))
}

//===========================================================================
// Access Management Pages
//===========================================================================

func (s *Server) APIKeyListPage(c *gin.Context) {
	// Set CSRF cookies for the api key management forms.
	if err := s.csrf.SetDoubleCookieToken(c); err != nil {
		s.Error(c, err)
		return
	}
	c.HTML(http.StatusOK, "pages/apikeys/list.html", scene.New(c).ForPage("apikeys"))
}
