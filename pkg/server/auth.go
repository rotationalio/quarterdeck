package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	gimlet "go.rtnl.ai/gimlet/auth"

	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
)

// PrepareLogin sets CSRF cookies to protect the login form and renders a login form
// if the user requests HTML (otherwise it returns a 204 with just the cookies set).
func (s *Server) PrepareLogin(c *gin.Context) {
	// Set CSRF cookies for the login form
	if err := s.csrf.SetDoubleCookieToken(c); err != nil {
		s.Error(c, err)
		return
	}

	// Render the login page if this is an html/htmx request.
	// NOTE: the scene does a lot of work to fetch URL information for the login form.
	if htmx.IsWebRequest(c) {
		ctx := scene.New(c).Login(c)
		c.HTML(http.StatusOK, "partials/auth/login.html", ctx)
		return
	}

	// Render a 204 No Content response with CSRF cookies set
	// NOTE: c.Status(http.StatusNoContent) doesn't work, so we have to use c.Data or c.JSON
	c.JSON(http.StatusNoContent, nil)
}

// Login is oriented toward human users who use their email and password for
// authentication (whereas Authenticate is used for machine access using API keys)
// Login verifies the password submitted by the user is correct by looking up the user
// in the database via email and using the argon2 derived key to verify the password.
// Upon authentication an access and refresh token with the authorized claims of the
// user are returned. The user can use the access token to authenticate to all systems
// specified by the audience and the claims can dictate to those systems what operations
// the user is allowed to perform. The refresh token can be used to reauthenticate the
// user without resubmitting the password, but it is only valid for a limited time.
func (s *Server) Login(c *gin.Context) {
	var (
		err  error
		user *models.User
		in   *api.LoginRequest
		out  *api.LoginReply
	)

	if err = c.BindJSON(&in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(errors.ErrBindJSON))
		return
	}

	// Ensure this is a valid login request
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Retrieve the user by email
	if user, err = s.store.RetrieveUser(c.Request.Context(), in.Email); err != nil {
		// Do not indicate whether or not the user exists to prevent enumeration attacks
		// Simply indicate that the authentication failed.
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusUnauthorized, api.Error(errors.ErrFailedAuthentication))
			return
		}

		// Otherwise something very bad is happening with the database
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(errors.ErrInternal))
		return
	}

	// User must be verified before they can log in.
	// TODO: redirect to an email verification page where they can request a new verification email
	if !user.EmailVerified {
		c.JSON(http.StatusUnauthorized, api.Error(errors.ErrEmailNotVerified))
		return
	}

	// Check that the password supplied by the user is correct.
	var verified bool
	if verified, err = passwords.VerifyDerivedKey(user.Password, in.Password); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(errors.ErrInternal))
		return
	}

	if !verified {
		// If the password is incorrect, return a failed authentication error.
		c.JSON(http.StatusUnauthorized, api.Error(errors.ErrFailedAuthentication))
		return
	}

	// Update the user's last login time after successful authentication.
	if err = s.store.UpdateLastLogin(c.Request.Context(), user.ID, time.Now()); err != nil {
		// If we cannot update the last login time, still return the access tokens but
		// log the error. This is not critical to the authentication process.
		c.Error(err)
	}

	// Prepare the login reply now that the user has been authenticated
	out = &api.LoginReply{}
	if user.LastLogin.Valid {
		out.LastLogin = user.LastLogin.Time
	}

	// Create the access and refresh tokens for the user.
	var claims *gimlet.Claims
	if claims, err = user.Claims(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(errors.ErrInternal))
		return
	}

	if out.AccessToken, out.RefreshToken, err = s.issuer.CreateTokens(claims); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(errors.ErrInternal))
		return
	}

	// Set tokens as cookies to the frontend, if configured to do so.
	if err = auth.SetAuthCookies(c, out.AccessToken, out.RefreshToken); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(errors.ErrInternal))
		return
	}

	// Content negotiation and redirection if required.
	switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
	case binding.MIMEJSON:
		c.JSON(http.StatusOK, out)
	case binding.MIMEHTML:
		location := in.Next
		if location == "" {
			location = s.conf.Auth.LoginRedirect
		}
		htmx.Redirect(c, http.StatusSeeOther, location)
	default:
		c.AbortWithError(http.StatusNotAcceptable, errors.ErrNotAccepted)
	}
}

func (s *Server) Logout(c *gin.Context) {
	// Clear the authentication cookies to log out the user.
	auth.ClearAuthCookies(c, s.conf.Auth.Audience)

	// Redirect to the login page after logging out.
	htmx.Redirect(c, http.StatusSeeOther, s.conf.Auth.LoginURL)
}
