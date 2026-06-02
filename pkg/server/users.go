package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.rtnl.ai/commo"
	gimauth "go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/cursor"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/randstr"
	"go.rtnl.ai/x/rlog"
	"go.rtnl.ai/x/vero"
)

// usersTracer names OpenTelemetry spans for user lifecycle operations.
var usersTracer = otel.Tracer("go.rtnl.ai/quarterdeck/pkg/server")

// ============================================================================
// User resource handlers
// ============================================================================

// ListUsers returns a page of users, optionally filtered by role.
func (s *Server) ListUsers(c *gin.Context) {
	var (
		err        error
		in         *api.UserPageQuery
		userModels cursor.Cursor[*models.User]
		out        *api.UserList
	)

	// Parse the URL parameters from the input request
	in = &api.UserPageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("invalid query parameters"))
		return
	}

	// TODO: implement pagination

	// List users
	if userModels, err = s.store.ListUsers(c.Request.Context(), nil); err != nil {
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

// CreateUser creates a user or idempotently updates one with the same email.
// Unverified users receive a welcome email; verified users do not. The user
// must complete the password-reset flow from that email before they can log in.
func (s *Server) CreateUser(c *gin.Context) {
	ctx, span := usersTracer.Start(c.Request.Context(), "users.create")
	defer span.End()

	var (
		user             *api.User
		err              error
		model            *models.User
		welcomeAttempted bool
		welcomeErr       error
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

	// Create the user, or upsert when the email already exists.
	if err = s.store.CreateUser(ctx, model); err != nil {
		if errors.Is(err, errors.ErrAlreadyExists) {
			if model, err = s.upsertExistingUser(ctx, user); err != nil {
				c.Error(errors.Join(err, errors.New("could not upsert existing user")))
				c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
				return
			}
		} else {
			c.Error(errors.Join(err, errors.New("could not create user")))
			c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
			return
		}
	}

	span.SetAttributes(attribute.String("user.id", model.ID.String()))

	if !model.EmailVerified {
		welcomeAttempted = true
		welcomeErr = s.sendWelcomeEmail(ctx, model)
		if welcomeErr != nil {
			span.RecordError(welcomeErr)
			span.SetStatus(codes.Error, "welcome email failed")
			rlog.ErrorAttrs(ctx, "could not send user a welcome email",
				slog.Any("err", welcomeErr), slog.String("user_id", model.ID.String()))
		}
	}

	// Reload so roles and associations are current for the API response.
	if model, err = s.store.RetrieveUser(ctx, model.ID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	if user, err = api.NewUser(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create user request"))
		return
	}

	s.syncUserPost(c, user, nil, true)

	if welcomeAttempted && welcomeErr != nil {
		htmx.SetTrigger(c, htmx.EventUserCreated, htmx.EventInviteWelcomeEmailFailed)
	} else {
		htmx.SetTrigger(c, htmx.EventUserCreated)
	}

	c.JSON(http.StatusOK, user)
}

// UserDetail returns the full user record for a user ID.
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

// UpdateUser updates applicable user fields and syncs the record to endeavor.
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
	s.syncUserPost(c, user, nil, true)

	// TODO: negotiate HTMX response when UI pages are implemented for users
	c.JSON(http.StatusOK, user)
}

// DeleteUser removes a user and notifies the endeavor webhook.
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

// ============================================================================
// Password and profile handlers
// ============================================================================

// ChangePassword updates the password when the user knows their current password.
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

// ForgotPassword emails a reset link when the address is known; always redirects to success.
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
		if err = s.sendResetPasswordEmail(c, in.Email); err != nil {
			// If the user is not found, then still redirect to the success page because
			// we don't want to leak information about whether the email address is valid.
			// If the error is ErrTooSoon, then we want to rate limit the user without
			// leaking information so also redirect to the success page.
			if !errors.Is(err, errors.ErrNotFound) && !errors.Is(err, errors.ErrTooSoon) {
				c.Error(err)
				s.Error(c, errors.New("could not complete reset password request"))
				return
			}

			rlog.WarnAttrs(c.Request.Context(), "non-user email address provided for reset password request",
				slog.Any("err", err), slog.String("email", in.Email))
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

// ResetPassword verifies the emailed link and sets a new password.
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
		rlog.ErrorAttrs(c.Request.Context(), "could not delete reset password link record",
			slog.Any("err", err), slog.String("link_id", veroToken.ID.String()))
	}
	auth.ClearResetPasswordTokenCookie(c, s.conf.Auth.GetResetPasswordURL().Hostname())

	// Complete the transaction
	tx.Commit()

	// Signal to HTMX that the password has been changed successfully
	c.HTML(http.StatusOK, "auth/reset/success.html", scene.New(c))
}

// ============================================================================
// Password reset email
// ============================================================================

// resetPasswordTokenTTL is how long a forgot-password link remains valid.
const resetPasswordTokenTTL = 15 * time.Minute

// sendResetPasswordEmail creates a vero token and emails a password-reset link.
func (s *Server) sendResetPasswordEmail(c *gin.Context, emailOrUserID any) (err error) {
	ctx := c.Request.Context()

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
	resetURL := s.conf.Auth.GetResetPasswordURL()
	resetURL.Host = s.conf.App.BaseURL().Host
	emailData := emails.ResetPasswordEmailData{
		ContactName:         user.Name.String,
		PasswordLinkBaseURL: resetURL,
		EmailBaseData: emails.EmailBaseData{
			AppName:        s.conf.App.Name,
			AppLogoURL:     s.conf.App.LogoURL(),
			OrgName:        s.conf.Org.Name,
			OrgHomepageURL: s.conf.Org.HomepageURL(),
			SupportEmail:   s.conf.Org.SupportEmail,
		},
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

// ============================================================================
// Vero token verification
// ============================================================================

// verifyVeroToken loads and cryptographically validates an emailed verification link.
func (s *Server) verifyVeroToken(ctx context.Context, verification *api.URLVerification) (token *models.VeroToken, err error) {
	// Validate the verification token
	if err = verification.Validate(); err != nil {
		return nil, err
	}

	// Get the VeroToken record from the database
	if token, err = s.store.RetrieveVeroToken(ctx, verification.RecordULID()); err != nil {
		rlog.DebugAttrs(ctx, "could not retrieve vero token record",
			slog.Any("err", err), slog.String("vero_token_id", verification.RecordULID().String()))
		return nil, err
	}

	// Check that the token is valid
	if secure, err := token.Signature.Verify(verification.VerificationToken()); err != nil || !secure {
		// If the token is not secure or verifiable, be freaked out and warn admins
		rlog.WarnAttrs(ctx, "a vero token request hmac verification failed",
			slog.Any("err", err), slog.String("vero_token_id", token.ID.String()), slog.Bool("secure", secure))
		return nil, errors.ErrNotAllowed
	}

	// Check that the token and link have both not expired
	if token.Signature.Token.IsExpired() || token.IsExpired() {
		rlog.DebugAttrs(ctx, "received a request with an expired verification token",
			slog.String("vero_token_id", token.ID.String()))
		return nil, errors.ErrExpiredToken
	}

	return token, nil
}

// SlowDown adds a random delay to slow brute-force or enumeration attempts.
func SlowDown() {
	delay := time.Duration(rand.Int64N(2000)+2500) * time.Millisecond
	time.Sleep(delay)
}

// ============================================================================
// Endeavor webhook sync
// ============================================================================

// syncUserPost POSTs the user JSON to the endeavor sync webhook.
// When async is true the request runs in a background goroutine.
func (s *Server) syncUserPost(c *gin.Context, user *api.User, accessToken *string, async bool) {
	bearer := syncBearerToken(c, accessToken)
	ctx := c.Request.Context()
	if async {
		ctx = context.WithoutCancel(ctx)
		u := *user
		go s.postUserSync(ctx, &u, bearer)
		return
	}
	s.postUserSync(ctx, user, bearer)
}

// syncBearerToken returns the explicit token or the bearer token from the gin context.
func syncBearerToken(c *gin.Context, accessToken *string) string {
	if accessToken != nil && *accessToken != "" {
		return *accessToken
	}
	token, _ := gimauth.GetAccessToken(c)
	return token
}

// postUserSync performs the endeavor user sync HTTP POST.
func (s *Server) postUserSync(ctx context.Context, user *api.User, bearer string) {
	if bearer == "" {
		rlog.WarnAttrs(ctx, "user sync post: missing access token",
			slog.String("user_id", user.ID.String()))
		return
	}

	u := s.conf.App.WebhookURL()
	bodyBytes, err := json.Marshal(user)
	if err != nil {
		rlog.WarnAttrs(ctx, "user sync post: could not marshal user to json",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", user.ID.String()))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		rlog.WarnAttrs(ctx, "user sync post: could not create new post request",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", user.ID.String()))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		rlog.WarnAttrs(ctx, "user sync post: could not complete http post request",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", user.ID.String()))
		return
	}
	resp.Body.Close()

	rlog.DebugAttrs(ctx, "user sync post successful",
		slog.String("endpoint_url", u.String()), slog.String("user_id", user.ID.String()))
}

// syncUserDelete notifies endeavor that a user was deleted.
func (s *Server) syncUserDelete(c *gin.Context, userID ulid.ULID) {
	var (
		req   *http.Request
		idURL string
		token string
		resp  *http.Response
		err   error
	)

	u := s.conf.App.WebhookURL()

	// Create the URL by appending the userID onto the sync webhook url path
	if idURL, err = url.JoinPath(u.String(), userID.String()); err != nil {
		rlog.WarnAttrs(c.Request.Context(), "user sync delete: could not create sync url",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", userID.String()))
		return
	}

	// Create a DELETE request
	if req, err = http.NewRequestWithContext(c.Request.Context(), http.MethodDelete, idURL, nil); err != nil {
		rlog.WarnAttrs(c.Request.Context(), "user sync delete: could not create new delete request",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", userID.String()))
		return
	}

	// Add authorization token
	if token, err = gimauth.GetAccessToken(c); err != nil || token == "" {
		rlog.WarnAttrs(c.Request.Context(), "user sync delete: could not attain an access token from context",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", userID.String()))
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	// Do request
	if resp, err = http.DefaultClient.Do(req); err != nil {
		rlog.WarnAttrs(c.Request.Context(), "user sync delete: could not complete http post request",
			slog.Any("err", err), slog.String("endpoint_url", u.String()), slog.String("user_id", userID.String()))
		return
	}
	resp.Body.Close()

	rlog.DebugAttrs(c.Request.Context(), "user sync delete successful",
		slog.String("endpoint_url", u.String()), slog.String("user_id", userID.String()))
}

// ============================================================================
// Team invite and welcome email
// ============================================================================

// welcomeEmailTokenTTL is how long a team-invite password link remains valid.
const welcomeEmailTokenTTL = 48 * time.Hour

// welcomeEmailResendCooldown is the minimum time between welcome email sends.
const welcomeEmailResendCooldown = 15 * time.Minute

// upsertExistingUser updates an existing user when CreateUser is retried.
func (s *Server) upsertExistingUser(ctx context.Context, in *api.User) (*models.User, error) {
	existing, err := s.store.RetrieveUser(ctx, in.Email)
	if err != nil {
		return nil, err
	}

	// Only update the name for now.
	existing.Name = sql.NullString{Valid: in.Name != "", String: in.Name}
	if err = s.store.UpdateUser(ctx, existing); err != nil {
		return nil, err
	}

	return s.store.RetrieveUser(ctx, existing.ID)
}

// teamInviteTokenValid reports whether a stored team-invite token can still be used.
func teamInviteTokenValid(record *models.VeroToken) bool {
	if record == nil || record.ID.IsZero() || record.IsExpired() {
		// The token is invalid if it is nil, has no ID, or is expired.
		return false
	}
	if record.Signature == nil {
		// The token is invalid if it has no signature.
		return false
	}
	// The token is valid if the signature is not expired.
	return !record.Signature.Token.IsExpired()
}

// welcomeEmailRateLimited reports whether a welcome email was sent too recently.
func welcomeEmailRateLimited(record *models.VeroToken) bool {
	if record == nil || !record.SentOn.Valid {
		return false
	}
	return time.Since(record.SentOn.Time) < welcomeEmailResendCooldown
}

// sendWelcomeEmail creates a team-invite token and emails the welcome message.
// Verified users are skipped. An existing valid invite may be resent after
// [welcomeEmailResendCooldown].
func (s *Server) sendWelcomeEmail(ctx context.Context, user *models.User) (err error) {
	if user.EmailVerified {
		return nil
	}

	ctx, span := usersTracer.Start(ctx, "users.welcome_email")
	defer span.End()
	span.SetAttributes(attribute.String("user.id", user.ID.String()))

	var tx txn.Txn
	if tx, err = s.store.Begin(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	// Create a new token or reuse an existing one.
	record := &models.VeroToken{
		TokenType:  enum.TokenTypeTeamInvite,
		ResourceID: ulid.NullULID{Valid: true, ULID: user.ID},
		Email:      user.Email,
		Expiration: time.Now().Add(welcomeEmailTokenTTL),
	}
	if err = tx.CreateTeamInviteVeroToken(record); err != nil {
		if !errors.Is(err, errors.ErrTooSoon) {
			return err
		}

		existing, err := tx.RetrieveTeamInviteVeroToken(user.ID)
		if err != nil {
			return err
		}

		switch {
		case teamInviteTokenValid(existing):
			// If the existing token is valid, check if it was sent too
			// recently. If it was, do not send a new email.
			if welcomeEmailRateLimited(existing) {
				return tx.Commit()
			}
			record = existing
		default:
			// If the existing token is invalid, delete it and create a new
			// token.
			if err = tx.DeleteVeroToken(existing.ID); err != nil {
				return err
			}
			if err = tx.CreateTeamInviteVeroToken(record); err != nil {
				return err
			}
		}
	}

	resetURL := *s.conf.Auth.GetResetPasswordURL()
	resetURL.Host = s.conf.App.BaseURL().Host
	emailData := emails.WelcomeUserEmailData{
		ContactName:          user.Name.String,
		Role:                 emails.RoleTitle(user),
		PasswordResetURL:     &resetURL,
		WelcomeEmailBodyText: s.conf.App.WelcomeEmail.TextContent(),
		WelcomeEmailBodyHTML: s.conf.App.WelcomeEmail.HTMLContent(),
		EmailBaseData: emails.EmailBaseData{
			AppName:        s.conf.App.Name,
			AppLogoURL:     s.conf.App.LogoURL(),
			OrgName:        s.conf.Org.Name,
			OrgHomepageURL: s.conf.Org.HomepageURL(),
			SupportEmail:   s.conf.Org.SupportEmail,
		},
	}

	verification, err := vero.New(record.ID[:], record.Expiration)
	if err != nil {
		return err
	}

	if emailData.Token, record.Signature, err = verification.Sign(); err != nil {
		return err
	}

	if err = tx.UpdateVeroToken(record); err != nil {
		return err
	}

	if err = emails.ValidateWelcomeUserEmail(emailData); err != nil {
		return err
	}

	email, err := emails.NewWelcomeUserEmail(user.Email, emailData)
	if err != nil {
		return err
	}

	if err = email.Send(); err != nil {
		return err
	}

	record.SentOn = sql.NullTime{Valid: true, Time: time.Now()}
	if err = tx.UpdateVeroToken(record); err != nil {
		return err
	}

	return tx.Commit()
}
