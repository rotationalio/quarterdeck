package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
	"go.rtnl.ai/ulid"
)

func (s *Server) ListAccounts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) CreateAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) AccountDetail(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) UpdateAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) DeleteAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) ChangePassword(c *gin.Context) {
	var (
		err        error
		in         *api.ProfilePassword
		accountID  ulid.ULID
		user       *models.User
		derivedKey string
		template   = "partials/profile/changePassword.html"
	)

	// Profile requests are only available for logged in users and therefore are UI
	// only requests (Accept: text/html). JSON requests return a 406 error.
	if !htmx.IsWebRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint unavailable for API calls"))
		return
	}

	in = &api.ProfilePassword{}
	if err = c.BindJSON(in); err != nil {
		c.HTML(http.StatusBadRequest, template, gin.H{"Error": "could not parse password change request"})
		return
	}

	if err = in.Validate(); err != nil {
		var out interface{}
		if verr, ok := err.(api.ValidationErrors); ok {
			out = gin.H{"FieldErrors": verr.Map()}
		} else {
			out = gin.H{"Error": err.Error()}
		}

		c.HTML(http.StatusBadRequest, template, out)
		return
	}

	// Retrieve the user's ID from the path parameter
	if accountID, err = ulid.Parse(c.Param("accountID")); err != nil {
		c.HTML(http.StatusBadRequest, template, gin.H{"Error": "could not change password"})
		return
	}

	// Fetch the model from the database
	if user, err = s.store.RetrieveUser(c.Request.Context(), accountID); err != nil {
		// By default in change password we'll return 400 to display the error alert.
		// Only if something is really bad we will redirect to error page.
		switch {
		case errors.Is(err, errors.ErrNotFound):
			c.HTML(http.StatusBadRequest, template, gin.H{"Error": "could not change password"})
		default:
			c.Error(err)
			c.HTML(http.StatusInternalServerError, template, gin.H{"Error": "could not change password"})
		}
		return
	}

	// Confirm the current password is correct
	if verified, err := passwords.VerifyDerivedKey(user.Password, in.Current); err != nil || !verified {
		c.HTML(http.StatusBadRequest, template, gin.H{"FieldErrors": map[string]string{"current": "password is incorrect"}})
		return
	}

	// Create derived key from requested password reset
	if derivedKey, err = passwords.CreateDerivedKey(in.Password); err != nil {
		c.Error(err)
		c.HTML(http.StatusInternalServerError, template, gin.H{"Error": "could not change password"})
		return
	}

	// Set the password for the specified user
	if err = s.store.UpdatePassword(c.Request.Context(), user.ID, derivedKey); err != nil {
		c.Error(err)
		c.HTML(http.StatusInternalServerError, template, gin.H{"Error": "could not change password"})
		return
	}

	// Success! Log the user out and redirect to the login page.
	auth.ClearAuthCookies(c, s.conf.Auth.Audience)

	// Send the user to the login page if this is an HTMX request
	if htmx.IsHTMXRequest(c) {
		htmx.Redirect(c, http.StatusSeeOther, "/login")
		return
	}

	// Otherwise respond with a JSON 200 message
	c.JSON(http.StatusOK, &api.Reply{Success: true})
}
