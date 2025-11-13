package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/commo"
	gimauth "go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/randstr"
	"go.rtnl.ai/x/vero"
)

// List all users or the users for a specific role. Returns summary information
// for the users only.
func (s *Server) ListUsers(c *gin.Context) {
	var (
		err        error
		in         *api.UserPageQuery
		page       *models.UserPage
		userModels *models.UserList
		out        *api.UserList
	)

	// Parse the URL parameters from the input request
	in = &api.UserPageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("invalid query parameters"))
		return
	}

	// Query page to model page
	if page, err = in.UserPage().Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("invalid query parameters"))
		return
	}

	// List users
	if userModels, err = s.store.ListUsers(c.Request.Context(), page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process users list request"))
		return
	}

	// Convert the database model to an API output
	if out, err = api.NewUserList(userModels); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process users list request"))
		return
	}

	c.JSON(http.StatusOK, out)
}

// Create a user via the API.
// NOTE: Does not set a user password; the user must perform the "forgot password"
// flow to reset their password via email.
func (s *Server) CreateUser(c *gin.Context) {
	var (
		user  *api.User
		err   error
		model *models.User
	)

	// Parse the model from the POST request
	user = &api.User{}
	if err = c.BindJSON(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse user data"))
		return
	}

	// Validate the user to be created
	if err = user.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the API model to a database model
	if model, err = user.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process user data"))
		return
	}

	// Set an unguessable random password for the new user (they will need to
	// reset their password via an email verification link to login)
	if model.Password, err = passwords.CreateDerivedKey(randstr.Password(24)); err != nil {
		s.Error(c, err)
		return
	}

	// Create the user
	if err = s.store.CreateUser(c.Request.Context(), model); err != nil {
		c.Error(errors.Fmt("could not create user: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	// TODO: send verification email (for now, the user can verify themselves by performing "forgot/reset" password)

	// Convert the model back to an API response
	if user, err = api.NewUser(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	// Sync user
	s.syncUserPost(c, user)

	// TODO: negotiate HTMX response when UI pages are implemented for users
	c.JSON(http.StatusOK, user)
}

// Return the full user model for a specific user.
func (s *Server) UserDetail(c *gin.Context) {
	var (
		err    error
		userID ulid.ULID
		user   *models.User
		out    *api.User
	)

	// Parse the user ID from the URL parameter
	if userID, err = ulid.Parse(c.Param("userID")); err != nil {
		c.Error(err)
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	// Retreive the user from DB
	if user, err = s.store.RetrieveUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process user detail request"))
		return
	}

	// Convert the user to an API response
	if out, err = api.NewUser(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process user detail request"))
		return
	}

	// TODO: negotiate HTMX response when UI pages are implemented for users
	c.JSON(http.StatusOK, out)
}

// Updates applicable user fields in the database.
func (s *Server) UpdateUser(c *gin.Context) {
	var (
		user   *api.User
		userID ulid.ULID
		err    error
		model  *models.User
	)

	// Parse the model from the POST request
	user = &api.User{}
	if err = c.BindJSON(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse user data"))
		return
	}

	// Parse the user ID from the URL parameter
	if userID, err = ulid.Parse(c.Param("userID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("user id not found"))
		return
	}

	// Validate the user to be updated
	if err = user.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Set the user ID only after validation
	user.ID = userID

	// Convert the API model to a database model
	if model, err = user.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process user data"))
		return
	}

	// Update the user
	if err = s.store.UpdateUser(c.Request.Context(), model); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update user request"))
		return
	}

	// Convert the model back to an API response
	if user, err = api.NewUser(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	// Sync user
	s.syncUserPost(c, user)

	// TODO: negotiate HTMX response when UI pages are implemented for users
	c.JSON(http.StatusOK, user)
}

func (s *Server) DeleteUser(c *gin.Context) {
	var (
		err    error
		userID ulid.ULID
	)

	// Parse the user ID from the URL parameter
	if userID, err = ulid.Parse(c.Param("userID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	// Delete the user from the database
	// TODO: for audit purposes we may simply want to move the user to an 'inactive' or 'deleted' status.
	if err = s.store.DeleteUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("user not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process delete user request"))
		return
	}

	// Sync user
	s.syncUserDelete(c, userID)

	// TODO: negotiate HTMX response when UI pages are implemented for users
	c.JSON(http.StatusOK, api.Reply{Success: true})
}

// Allows a user to change their password if they know their current one.
func (s *Server) ChangePassword(c *gin.Context) {
	var (
		err        error
		in         *api.ProfilePassword
		userID     ulid.ULID
		user       *models.User
		derivedKey string
		template   = "partials/profile/changePassword.html"
	)

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
	if userID, err = ulid.Parse(c.Param("userID")); err != nil {
		c.HTML(http.StatusBadRequest, template, gin.H{"Error": "could not change password"})
		return
	}

	// Fetch the model from the database
	if user, err = s.store.RetrieveUser(c.Request.Context(), userID); err != nil {
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

// Looks up a user by email and sends that user a link/token to reset their password.
func (s *Server) ForgotPassword(c *gin.Context) {
	var (
		err error
		in  *api.ResetPasswordRequest
	)

	// We do not allow JSON API requests to this endpoint.
	// Technically someone could automate requests with an Accept: text/html header
	// so it's also important to rate limit reset password requests. But returning a
	// 406 error here is for the legitimate API users.
	if !htmx.IsWebRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint unavailable for API calls"))
		return
	}

	in = &api.ResetPasswordRequest{}
	if err = c.BindJSON(in); err != nil {
		s.Error(c, errors.New("could not parse reset password request"))
		return
	}

	// Send the email, also creating a verification token; if no email was provided
	// simply redirect them to the success page to avoid leaking information.
	if in.Email != "" {
		ctx := c.Request.Context()
		if err = s.sendResetPasswordEmail(ctx, in.Email); err != nil {
			// If the user is not found, then still redirect to the success page because
			// we don't want to leak information about whether the email address is valid.
			// If the error is ErrTooSoon, then we want to rate limit the user without
			// leaking information so also redirect to the success page.
			if !errors.Is(err, errors.ErrNotFound) && !errors.Is(err, errors.ErrTooSoon) {
				c.Error(err)
				s.Error(c, errors.New("could not complete reset password request"))
				return
			}

			log.Warn().Err(err).Str("email", in.Email).Msg("non-user email address provided for reset password request")
		}
	}

	// Make sure the user is logged out to prevent session hijacking
	auth.ClearAuthCookies(c, s.conf.Auth.Audience)

	// Redirect to reset-password success page (note do not use an HTMX partial here
	// because the forgot password request can come from a logged in user on their
	// profile page or a non-logged in user on the login page); a full redirect is
	// necessary so they can close this window and follow the flow from their email.
	htmx.Redirect(c, http.StatusSeeOther, "/forgot-password/sent")
}

// Verifies an incoming password change requested via a verification link, then changes
// the user's password according to the password form submitted.
func (s *Server) ResetPassword(c *gin.Context) {
	var (
		derivedKey string
		err        error
		in         *api.ResetPasswordChangeRequest
		veroToken  *models.VeroToken
	)

	// We do not allow JSON API requests to this endpoint. Returning a 406 error
	// here is for the legitimate API users who need to not use this endpoint.
	if !htmx.IsWebRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint unavailable for API calls"))
		return
	}

	// Read the token string from the query parameters
	in = &api.ResetPasswordChangeRequest{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse reset password request"))
		return
	}

	// Get the verification token from the cookie
	if in.Token, err = c.Cookie(auth.ResetPasswordTokenCookie); err != nil {
		// If no cookie is submitted, then slow down the request and send back a 403.
		// The slow down prevents spamming the reset password endpoint.
		SlowDown()
		c.JSON(http.StatusForbidden, api.Error("unable to process reset password request"))
		return
	}

	// Validate the change password input
	if err = in.Validate(); err != nil {
		// If the token is invalid or missing, return a 422.
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Verify the VeroToken token
	if veroToken, err = s.verifyVeroToken(c.Request.Context(), &in.URLVerification); err != nil {
		switch {
		case errors.Is(err, errors.ErrNotFound), errors.Is(err, errors.ErrExpiredToken):
			// If the link is not found or expired, the user needs to try to reset their password again
			c.JSON(http.StatusBadRequest, api.Error("your reset password link is invalid or expired, please submit a new forgot password request"))
			return
		case errors.Is(err, errors.ErrNotAllowed):
			// If the link is not verified or secure, then slow down the request and send back a 403.
			// The slow down prevents brute force attacks on the change password endpoint.
			SlowDown()
			c.JSON(http.StatusForbidden, api.Error("unable to process reset password request"))
			return
		default:
			s.Error(c, err)
			return
		}
	}

	// Create derived key from requested password reset
	if derivedKey, err = passwords.CreateDerivedKey(in.Password); err != nil {
		s.Error(c, err)
		return
	}

	// Begin a transaction
	var tx txn.Txn
	if tx, err = s.store.Begin(c.Request.Context(), &sql.TxOptions{ReadOnly: false}); err != nil {
		s.Error(c, err)
		return
	}
	defer tx.Rollback()

	// Set the password for the specified user
	if err = tx.UpdatePassword(veroToken.ResourceID.ULID, derivedKey); err != nil {
		s.Error(c, err)
		return
	}

	// Because we contacted the user via email to reset their password, this
	// can count as an email verification if they are not yet verified
	if err = tx.VerifyEmail(veroToken.ResourceID.ULID); err != nil {
		s.Error(c, err)
		return
	}

	// Now that the password has been changed, delete the VeroToken record and
	// clear its cookie
	if err = tx.DeleteVeroToken(veroToken.ID); err != nil {
		// Do not return an error if we could not delete the record, just log it.
		log.Error().Err(err).Str("link_id", veroToken.ID.String()).Msg("could not delete reset password link record")
	}
	auth.ClearResetPasswordTokenCookie(c, s.conf.Auth.GetResetPasswordURL().Hostname())

	// Complete the transaction
	tx.Commit()

	// Signal to HTMX that the password has been changed successfully
	c.HTML(http.StatusOK, "auth/reset/success.html", scene.New(c))
}

// The default amount of time that a reset password token will expire after.
const resetPasswordTokenTTL = 15 * time.Minute

// Send a reset password email to the user, also creating a verification token.
func (s *Server) sendResetPasswordEmail(ctx context.Context, emailOrUserID any) (err error) {
	// Begin a read-write transaction
	var tx txn.Txn
	if tx, err = s.store.Begin(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	// Lookup the user
	var user *models.User
	if user, err = tx.RetrieveUser(emailOrUserID); err != nil {
		return err
	}

	// Create a VeroToken record for database storage
	record := &models.VeroToken{
		TokenType:  enum.TokenTypeResetPassword,
		ResourceID: ulid.NullULID{Valid: true, ULID: user.ID},
		Email:      user.Email,
		Expiration: time.Now().Add(resetPasswordTokenTTL),
	}

	// Create the ID in the database of the VeroToken record.
	// NOTE: the CreateVeroToken function will return ErrTooSoon if the record
	// already exists and is not expired; otherwise it will delete any existing
	// (expired) record for the user and create a new one. ErrTooSoon will
	// enable rate limiting to make sure the user cannot spam reset password
	// requests.
	if err = tx.CreateResetPasswordVeroToken(record); err != nil {
		return err
	}

	// Create the ResetPasswordEmailData for the email builder
	emailData := emails.ResetPasswordEmailData{
		ContactName:  user.Name.String,
		BaseURL:      s.conf.Auth.GetResetPasswordURL(),
		SupportEmail: s.conf.SupportEmail,
	}

	// Create the HMAC verification token for the VeroToken
	var verification *vero.Token
	if verification, err = vero.New(record.ID[:], record.Expiration); err != nil {
		return err
	}

	// Sign the verification token
	if emailData.Token, record.Signature, err = verification.Sign(); err != nil {
		return err
	}

	// Update the VeroToken record in the database with the token
	if err = tx.UpdateVeroToken(record); err != nil {
		return err
	}

	// Build the email
	var email *commo.Email
	if email, err = emails.NewResetPasswordEmail(user.Email, emailData); err != nil {
		return err
	}

	// Send the email to the user
	if err = email.Send(); err != nil {
		return err
	}

	// Update the VeroToken record in the database with a SentOn timestamp
	record.SentOn = sql.NullTime{Valid: true, Time: time.Now()}
	if err = tx.UpdateVeroToken(record); err != nil {
		return err
	}

	// Commit the successful transaction
	tx.Commit()

	return nil
}

// Verifies a VeroToken token and returns the VeroToken object.
func (s *Server) verifyVeroToken(ctx context.Context, verification *api.URLVerification) (token *models.VeroToken, err error) {
	// Validate the verification token
	if err = verification.Validate(); err != nil {
		return nil, err
	}

	// Get the VeroToken record from the database
	if token, err = s.store.RetrieveVeroToken(ctx, verification.RecordULID()); err != nil {
		log.Debug().Err(err).Str("vero_token_id", verification.RecordULID().String()).Msg("could not retrieve vero token record")
		return nil, err
	}

	// Check that the token is valid
	if secure, err := token.Signature.Verify(verification.VerificationToken()); err != nil || !secure {
		// If the token is not secure or verifiable, be freaked out and warn admins
		log.Warn().Err(err).Str("vero_token_id", token.ID.String()).Bool("secure", secure).Msg("a vero token request hmac verification failed")
		return nil, errors.ErrNotAllowed
	}

	// Check that the token and link have both not expired
	if token.Signature.Token.IsExpired() || token.IsExpired() {
		log.Debug().Str("vero_token_id", token.ID.String()).Msg("received a request with an expired verification token")
		return nil, errors.ErrExpiredToken
	}

	return token, nil
}

// Slow down sleeps the request for a random amount of time between 250ms and 2500ms
func SlowDown() {
	delay := time.Duration(rand.Int64N(2000)+2500) * time.Millisecond
	time.Sleep(delay)
}

// Syncs user create/update events with each configured webhook endpoint using a
// HTTP POST request with the created/modified [api.User] as the JSON body. This
// reuses the access token in the [gin.Context] to authenticate the request to
// the endpoint. It will log all errors, but will not handle the errors.
func (s *Server) syncUserPost(c *gin.Context, user *api.User) {
	for _, u := range s.conf.UserSync.WebhookURLs() {
		var (
			req       *http.Request
			bodyBytes []byte
			token     string
			resp      *http.Response
			err       error
		)

		// Marshal the user into JSON bytes
		if bodyBytes, err = json.Marshal(user); err != nil {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", user.ID.String()).Msg("user sync post: could not marshal user to json")
			return
		}

		// Create a POST request for JSON
		if req, err = http.NewRequestWithContext(c.Request.Context(), http.MethodPost, u.String(), bytes.NewReader(bodyBytes)); err != nil {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", user.ID.String()).Msg("user sync post: could not create new post request")
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Add authorization token
		if token, err = gimauth.GetAccessToken(c); err != nil || token == "" {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", user.ID.String()).Msg("user sync post: could not attain an access token from context")
			return
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

		// Do request
		if resp, err = http.DefaultClient.Do(req); err != nil {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", user.ID.String()).Str("response_status", resp.Status).Msg("user sync post: could not complete http post request")
			return
		}
	}
}

// Syncs user delete events with each configured webhook endpoint using a
// HTTP DELETE request with the deleted user's ID as a URL param. This reuses
// the access token in the [gin.Context] to authenticate the request to the
// endpoint. It will log all errors, but will not handle the errors.
func (s *Server) syncUserDelete(c *gin.Context, userID ulid.ULID) {
	for _, u := range s.conf.UserSync.WebhookURLs() {
		var (
			req   *http.Request
			idURL string
			token string
			resp  *http.Response
			err   error
		)

		// Create the URL by appending the userID onto the sync webhook url path
		if idURL, err = url.JoinPath(u.String(), userID.String()); err != nil {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", userID.String()).Msg("user sync delete: could not create sync url")
			return
		}

		// Create a DELETE request
		if req, err = http.NewRequestWithContext(c.Request.Context(), http.MethodDelete, idURL, nil); err != nil {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", userID.String()).Msg("user sync delete: could not create new delete request")
			return
		}

		// Add authorization token
		if token, err = gimauth.GetAccessToken(c); err != nil || token == "" {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", userID.String()).Msg("user sync delete: could not attain an access token from context")
			return
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

		// Do request
		if resp, err = http.DefaultClient.Do(req); err != nil {
			log.Warn().Err(err).Str("endpoint_url", u.String()).Str("user_id", userID.String()).Str("response_status", resp.Status).Msg("user sync delete: could not complete http post request")
			return
		}
	}

}
