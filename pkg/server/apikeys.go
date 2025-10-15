package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
	"go.rtnl.ai/quarterdeck/pkg/web/scene"
	"go.rtnl.ai/ulid"
)

func (s *Server) ListAPIKeys(c *gin.Context) {
	var (
		err    error
		in     *api.PageQuery
		page   *models.Page
		models *models.APIKeyList
		out    *api.APIKeyList
	)

	// PArse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("invalid query parameters"))
		return
	}

	// TODO: manage pagination mechanism

	if models, err = s.store.ListAPIKeys(c.Request.Context(), page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikeys list request"))
		return
	}

	// Convert the database model to an API output
	if out, err = api.NewAPIKeyList(models); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikeys list request"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) CreateAPIKey(c *gin.Context) {
	var (
		tx          txn.Txn
		err         error
		in          *api.APIKey
		key         *models.APIKey
		secret      string
		claims      *auth.Claims
		subjectID   ulid.ULID
		subjectType auth.SubjectType
		out         *api.APIKey
	)

	// Get the claims of the authenticated user creating the API key
	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnauthorized, api.Error("could not get user claims"))
		return
	}

	// Get the subject ID and subject type from the claims
	if subjectType, subjectID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	// Parse the model from the POST request
	in = &api.APIKey{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse apikey data"))
		return
	}

	// Validate the API key to be created
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the API model to a database model
	if key, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikey data"))
		return
	}

	// Create a client ID for the API key
	key.ClientID = passwords.ClientID()

	// Create a secret and the derived key of that secret
	secret = passwords.ClientSecret()
	if key.Secret, err = passwords.CreateDerivedKey(secret); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	// Create a transaction to handle the API key creation
	if tx, err = s.store.Begin(c.Request.Context(), nil); err != nil {
		c.Error(errors.Fmt("could not start write transaction for apikey: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}
	defer tx.Rollback()

	// Set the owner of the API key
	if subjectType == auth.SubjectUser {
		key.CreatedBy = subjectID
	} else if subjectType == auth.SubjectAPIKey {
		// Lookup the key being used in the database and set the created by to
		// the owner of that key (e.g. the user that created that key).
		var parent *models.APIKey
		if parent, err = tx.RetrieveAPIKey(subjectID); err != nil {
			c.Error(errors.Fmt("could not lookup parent API key: %w", err))
			c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
			return
		}
		key.CreatedBy = parent.CreatedBy
	} else {
		c.JSON(http.StatusForbidden, api.Error("only users and api keys can create api keys"))
		return
	}

	if err = tx.CreateAPIKey(key); err != nil {
		c.Error(errors.Fmt("could not create api key: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	if err = tx.Commit(); err != nil {
		c.Error(errors.Fmt("could not commit create api key transaction: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAPIKey(key); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	// Ensure the created apikey secret is returned to the user
	out.Secret = secret

	// Add HTMX Trigger to reload the API Key List
	c.Header(htmx.HXTriggerAfterSwap, htmx.APIKeysUpdated)

	// Content negotiation
	c.Negotiate(http.StatusCreated, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/apikeys/created.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) APIKeyDetail(c *gin.Context) {
	var (
		err   error
		keyID ulid.ULID
		key   *models.APIKey
		out   *api.APIKey
	)

	if keyID, err = ulid.Parse(c.Param("keyID")); err != nil {
		c.Error(err)
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	if key, err = s.store.RetrieveAPIKey(c.Request.Context(), keyID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikey detail request"))
		return
	}

	if out, err = api.NewAPIKey(key); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikey detail request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/apikeys/detail.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) UpdateAPIKeyPreview(c *gin.Context) {
	var (
		err    error
		keyID  ulid.ULID
		apikey *models.APIKey
		out    *api.APIKey
	)

	// Preview requests target a UI only audience and therefore only accept text/html
	// requests (Accept: text/html). JSON requests return a 406 error. The endpoint
	// still may return JSON errors for AJAX handling on the front-end.
	if !htmx.IsWebRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint unavailable for API calls"))
		return
	}

	// Parse the keyID from the URL
	if keyID, err = ulid.Parse(c.Param("keyID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	// Fetch the model from the database
	if apikey, err = s.store.RetrieveAPIKey(c.Request.Context(), keyID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process apikey detail request"))
		return
	}

	if out, err = api.NewAPIKey(apikey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process apikey detail request"))
		return
	}

	// Render the edit form for the API key
	c.HTML(http.StatusOK, "partials/apikeys/edit.html", scene.New(c).WithAPIData(out))
}

func (s *Server) UpdateAPIKey(c *gin.Context) {
	var (
		err   error
		keyID ulid.ULID
		key   *models.APIKey
		in    *api.APIKey
		out   *api.APIKey
	)

	// Parse the key ID from the URL parameter
	if keyID, err = ulid.Parse(c.Param("keyID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	// Parse the apikey data from the request
	in = &api.APIKey{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse apikey data"))
		return
	}

	// Validate the API key to be updated
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Set the key ID only after validation
	in.ID = keyID

	// Create the model to be updated
	if key, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update apikey request"))
		return
	}

	// Update the API key in the database
	if err = s.store.UpdateAPIKey(c.Request.Context(), key); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update apikey request"))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAPIKey(key); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update apikey request"))
		return
	}

	// Return successful JSON response or 204 with htmx trigger depending on the content negotiation
	switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
	case binding.MIMEJSON:
		c.JSON(http.StatusOK, out)
	case binding.MIMEHTML:
		htmx.Trigger(c, htmx.APIKeysUpdated)
	}
}

func (s *Server) DeleteAPIKey(c *gin.Context) {
	var (
		err   error
		keyID ulid.ULID
	)

	// Parse the key ID from the URL parameter
	if keyID, err = ulid.Parse(c.Param("keyID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	// Delete the API key from the database
	// TODO: for audit purposes we may simply want to move the API key to a revoked table.
	if err = s.store.DeleteAPIKey(c.Request.Context(), keyID); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process delete apikey request"))
		return
	}

	if htmx.IsHTMXRequest(c) {
		htmx.Trigger(c, htmx.APIKeysUpdated)
		return
	}

	c.JSON(http.StatusOK, api.Reply{Success: true})
}
