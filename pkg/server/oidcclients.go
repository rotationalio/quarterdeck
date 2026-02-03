package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *Server) ListOIDCClients(c *gin.Context) {
	var (
		err  error
		in   *api.PageQuery
		list *models.OIDCClientList
		out  *api.OIDCClientList
	)

	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("invalid query parameters"))
		return
	}

	list, err = s.store.ListOIDCClients(c.Request.Context(), in.PageModel())
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process oidc clients list request"))
		return
	}

	if out, err = api.NewOIDCClientList(list); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process oidc clients list request"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) CreateOIDCClient(c *gin.Context) {
	var (
		err         error
		in          *api.OIDCClient
		client      *models.OIDCClient
		secret      string
		claims      *auth.Claims
		subjectID   ulid.ULID
		subjectType auth.SubjectType
		out         *api.OIDCClient
	)

	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnauthorized, api.Error("could not get user claims"))
		return
	}

	if subjectType, subjectID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create oidc client request"))
		return
	}

	in = &api.OIDCClient{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse oidc client data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	if client, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process oidc client data"))
		return
	}

	client.ClientID = passwords.ClientID()
	secret = passwords.ClientSecret()
	if client.Secret, err = passwords.CreateDerivedKey(secret); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create oidc client request"))
		return
	}

	switch subjectType {
	case auth.SubjectUser:
		client.CreatedBy = subjectID
	case auth.SubjectAPIKey:
		var parent *models.APIKey
		if parent, err = s.store.RetrieveAPIKey(c.Request.Context(), subjectID); err != nil {
			c.Error(errors.Fmt("could not lookup parent API key: %w", err))
			c.JSON(http.StatusInternalServerError, api.Error("could not process create oidc client request"))
			return
		}
		client.CreatedBy = parent.CreatedBy
	default:
		c.JSON(http.StatusForbidden, api.Error("only users and api keys can create oidc clients"))
		return
	}

	if err = s.store.CreateOIDCClient(c.Request.Context(), client); err != nil {
		c.Error(errors.Fmt("could not create oidc client: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not process create oidc client request"))
		return
	}

	if out, err = api.NewOIDCClient(client); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create oidc client request"))
		return
	}
	out.Secret = secret

	c.JSON(http.StatusCreated, out)
}

func (s *Server) OIDCClientDetail(c *gin.Context) {
	var (
		err    error
		id     ulid.ULID
		client *models.OIDCClient
		out    *api.OIDCClient
	)

	if id, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
		return
	}

	client, err = s.store.RetrieveOIDCClient(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process oidc client detail request"))
		return
	}

	if out, err = api.NewOIDCClient(client); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process oidc client detail request"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) UpdateOIDCClient(c *gin.Context) {
	var (
		err      error
		id       ulid.ULID
		existing *models.OIDCClient
		in       *api.OIDCClient
		client   *models.OIDCClient
		out      *api.OIDCClient
	)

	if id, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
		return
	}

	existing, err = s.store.RetrieveOIDCClient(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update oidc client request"))
		return
	}

	in = &api.OIDCClient{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse oidc client data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	in.ID = existing.ID

	if client, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update oidc client request"))
		return
	}

	if err = s.store.UpdateOIDCClient(c.Request.Context(), client); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update oidc client request"))
		return
	}

	if out, err = api.NewOIDCClient(client); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process update oidc client request"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) RevokeOIDCClient(c *gin.Context) {
	var (
		err error
		id  ulid.ULID
	)

	if id, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
		return
	}

	if err = s.store.RevokeOIDCClient(c.Request.Context(), id); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process revoke oidc client request"))
		return
	}

	c.JSON(http.StatusOK, api.Reply{Success: true})
}

func (s *Server) DeleteOIDCClient(c *gin.Context) {
	var (
		err error
		id  ulid.ULID
	)

	if id, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
		return
	}

	if err = s.store.DeleteOIDCClient(c.Request.Context(), id); err != nil {
		if errors.Is(err, errors.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("oidc client not found"))
			return
		}
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process delete oidc client request"))
		return
	}

	c.JSON(http.StatusOK, api.Reply{Success: true})
}
