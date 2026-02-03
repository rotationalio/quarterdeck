package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func TestListOIDCClients(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// set mock callback
		clientID := ulid.MakeSecure()
		mockStore.OnListOIDCClients = func(ctx context.Context, page *models.Page) (*models.OIDCClientList, error) {
			return &models.OIDCClientList{
				OIDCClients: []*models.OIDCClient{
					{
						Model:        models.Model{ID: clientID, Created: time.Now(), Modified: time.Now()},
						ClientName:   "Test",
						RedirectURIs: []string{"https://example.com/cb"},
						ClientID:     "cid",
						CreatedBy:    ulid.MakeSecure(),
					},
				},
			}, nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients", nil, nil)

		// execute handler
		srv.ListOIDCClients(c)

		// parse response
		var out api.OIDCClientList
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))

		// assert response
		require.Equal(t, http.StatusOK, w.Code)
		require.Len(t, out.OIDCClients, 1)
		require.Equal(t, clientID, out.OIDCClients[0].ID)
		mockStore.AssertCalls(t, mock.ListOIDCClients, 1)
	})

	t.Run("BadRequest", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// build request (invalid query)
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients?page_size=notanint", nil, nil)

		// execute handler
		srv.ListOIDCClients(c)

		// assert response
		require.Equal(t, http.StatusBadRequest, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "invalid query parameters", reply.Error)
	})

	t.Run("StoreError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnListOIDCClients = func(ctx context.Context, page *models.Page) (*models.OIDCClientList, error) {
			return nil, errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients", nil, nil)

		// execute handler
		srv.ListOIDCClients(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process oidc clients list request", reply.Error)
	})
}

func TestCreateOIDCClient(t *testing.T) {
	t.Run("SuccessUser", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		var created *models.OIDCClient
		srv := newTestServer(mockStore)

		userID := ulid.MakeSecure()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectUser, userID)

		// set mock callback
		mockStore.OnCreateOIDCClient = func(ctx context.Context, in *models.OIDCClient) error {
			created = in
			created.ID = ulid.MakeSecure()
			created.Created = time.Now()
			created.Modified = created.Created
			created.CreatedBy = userID
			return nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", validCreateBody(), nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusCreated, w.Code)
		out := parseOIDCClient(t, w)
		require.Equal(t, "Test Client", out.ClientName)
		require.NotEmpty(t, out.Secret)
		require.NotEmpty(t, out.ClientID)
		mockStore.AssertCalls(t, mock.CreateOIDCClient, 1)
		require.Equal(t, userID, created.CreatedBy)
	})

	t.Run("SuccessAPIKey", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		apiKeyID := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		userID := ulid.MakeSecure()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectAPIKey, apiKeyID)

		// set mock callbacks
		var created *models.OIDCClient
		mockStore.OnRetrieveAPIKey = func(ctx context.Context, id any) (*models.APIKey, error) {
			return &models.APIKey{Model: models.Model{ID: apiKeyID}, CreatedBy: userID}, nil
		}
		mockStore.OnCreateOIDCClient = func(ctx context.Context, in *models.OIDCClient) error {
			created = in
			return nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", validCreateBody(), nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusCreated, w.Code)
		mockStore.AssertCalls(t, mock.RetrieveAPIKey, 1)
		mockStore.AssertCalls(t, mock.CreateOIDCClient, 1)
		require.Equal(t, userID, created.CreatedBy)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// build request (no claims)
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", validCreateBody(), nil)
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusUnauthorized, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not get user claims", reply.Error)
	})

	t.Run("BadRequest", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectUser, ulid.MakeSecure())
		srv := newTestServer(mockStore)

		// build request (invalid JSON)
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", []byte("not json"), nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusBadRequest, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not parse oidc client data", reply.Error)
	})

	t.Run("ValidationError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectUser, ulid.MakeSecure())
		srv := newTestServer(mockStore)

		body, _ := json.Marshal(&api.OIDCClient{ClientName: "Test", RedirectURIs: []string{}})

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", body, nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		reply := parseReply(t, w)
		require.Contains(t, reply.Error, "redirect_uris")
	})

	t.Run("APIKeyParentNotFound", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectAPIKey, ulid.MakeSecure())
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRetrieveAPIKey = func(ctx context.Context, id any) (*models.APIKey, error) {
			return nil, errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", validCreateBody(), nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process create oidc client request", reply.Error)
	})

	t.Run("ForbiddenSubjectType", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectVero, ulid.MakeSecure())
		srv := newTestServer(mockStore)

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", validCreateBody(), nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusForbidden, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "only users and api keys can create oidc clients", reply.Error)
	})

	t.Run("StoreCreateError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		claims := &auth.Claims{}
		claims.SetSubjectID(auth.SubjectUser, ulid.MakeSecure())
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnCreateOIDCClient = func(ctx context.Context, in *models.OIDCClient) error {
			return errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients", validCreateBody(), nil)
		c.Request.Header.Set("Content-Type", "application/json")
		gimlet.Set(c, gimlet.KeyUserClaims, claims)

		// execute handler
		srv.CreateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process create oidc client request", reply.Error)
	})
}

func TestOIDCClientDetail(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		clientID := ulid.MakeSecure()
		client := &models.OIDCClient{
			Model:        models.Model{ID: clientID, Created: time.Now(), Modified: time.Now()},
			ClientName:   "Detail Client",
			RedirectURIs: []string{"https://example.com/cb"},
			ClientID:     "cid",
			CreatedBy:    ulid.MakeSecure(),
		}
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRetrieveOIDCClient = func(ctx context.Context, id any) (*models.OIDCClient, error) {
			require.Equal(t, clientID, id)
			return client, nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients/"+clientID.String(), nil, gin.Params{{Key: "id", Value: clientID.String()}})

		// execute handler
		srv.OIDCClientDetail(c)

		// assert response
		require.Equal(t, http.StatusOK, w.Code)
		out := parseOIDCClient(t, w)
		require.Equal(t, "Detail Client", out.ClientName)
	})

	t.Run("NotFoundBadID", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// build request (invalid ID)
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients/invalid", nil, gin.Params{{Key: "id", Value: "invalid"}})

		// execute handler
		srv.OIDCClientDetail(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("NotFoundStore", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRetrieveOIDCClient = func(ctx context.Context, id any) (*models.OIDCClient, error) {
			return nil, errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients/"+id.String(), nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.OIDCClientDetail(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("StoreError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRetrieveOIDCClient = func(ctx context.Context, id any) (*models.OIDCClient, error) {
			return nil, errors.Fmt("db error")
		}

		// build request and context
		w, c := requestContext(t, http.MethodGet, "/v1/oidc/oidcclients/"+id.String(), nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.OIDCClientDetail(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process oidc client detail request", reply.Error)
	})
}

func TestUpdateOIDCClient(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		clientID := ulid.MakeSecure()
		var updated *models.OIDCClient
		srv := newTestServer(mockStore)

		mockStore.OnUpdateOIDCClient = func(ctx context.Context, in *models.OIDCClient) error {
			updated = in
			return nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/"+clientID.String(), validUpdateBody(clientID, "Updated"), gin.Params{{Key: "id", Value: clientID.String()}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusOK, w.Code)
		out := parseOIDCClient(t, w)
		require.Equal(t, "Updated", out.ClientName)
		require.Equal(t, clientID, updated.ID)
	})

	t.Run("NotFoundBadID", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// build request (invalid ID in URL)
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/badid", validUpdateBody(ulid.MakeSecure(), "Updated"), gin.Params{{Key: "id", Value: "badid"}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("BadRequestIDMismatch", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		paramID := ulid.MakeSecure()
		bodyID := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// build request (body id differs from URL param)
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/"+paramID.String(), validUpdateBody(bodyID, "Updated"), gin.Params{{Key: "id", Value: paramID.String()}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusBadRequest, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "client ulid must match the id parameter", reply.Error)
	})

	t.Run("BadRequest", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// build request (invalid JSON)
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/"+id.String(), []byte("not json"), gin.Params{{Key: "id", Value: id.String()}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusBadRequest, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not parse oidc client data", reply.Error)
	})

	t.Run("ValidationError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		body, _ := json.Marshal(&api.OIDCClient{ID: id, ClientName: "x", RedirectURIs: []string{}})

		// build request and context
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/"+id.String(), body, gin.Params{{Key: "id", Value: id.String()}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusUnprocessableEntity, w.Code)
		reply := parseReply(t, w)
		require.Contains(t, reply.Error, "redirect_uris")
	})

	t.Run("NotFoundUpdate", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		mockStore.OnUpdateOIDCClient = func(ctx context.Context, in *models.OIDCClient) error {
			return errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/"+id.String(), validUpdateBody(id, "Updated"), gin.Params{{Key: "id", Value: id.String()}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("StoreUpdateError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		mockStore.OnUpdateOIDCClient = func(ctx context.Context, in *models.OIDCClient) error {
			return errors.Fmt("db error")
		}

		// build request and context
		w, c := requestContext(t, http.MethodPut, "/v1/oidc/oidcclients/"+id.String(), validUpdateBody(id, "Updated"), gin.Params{{Key: "id", Value: id.String()}})
		c.Request.Header.Set("Content-Type", "application/json")

		// execute handler
		srv.UpdateOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process update oidc client request", reply.Error)
	})
}

func TestRevokeOIDCClient(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRevokeOIDCClient = func(ctx context.Context, id ulid.ULID) error {
			return nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients/"+id.String()+"/revoke", nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.RevokeOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusOK, w.Code)
		reply := parseReply(t, w)
		require.True(t, reply.Success)
	})

	t.Run("NotFoundBadID", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// build request (invalid ID)
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients/invalid/revoke", nil, gin.Params{{Key: "id", Value: "invalid"}})

		// execute handler
		srv.RevokeOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("NotFoundStore", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRevokeOIDCClient = func(ctx context.Context, id ulid.ULID) error {
			return errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients/"+id.String()+"/revoke", nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.RevokeOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("StoreError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnRevokeOIDCClient = func(ctx context.Context, id ulid.ULID) error {
			return errors.Fmt("db error")
		}

		// build request and context
		w, c := requestContext(t, http.MethodPost, "/v1/oidc/oidcclients/"+id.String()+"/revoke", nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.RevokeOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process revoke oidc client request", reply.Error)
	})
}

func TestDeleteOIDCClient(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnDeleteOIDCClient = func(ctx context.Context, id ulid.ULID) error {
			return nil
		}

		// build request and context
		w, c := requestContext(t, http.MethodDelete, "/v1/oidc/oidcclients/"+id.String(), nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.DeleteOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusOK, w.Code)
		reply := parseReply(t, w)
		require.True(t, reply.Success)
	})

	t.Run("NotFoundBadID", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		srv := newTestServer(mockStore)

		// build request (invalid ID)
		w, c := requestContext(t, http.MethodDelete, "/v1/oidc/oidcclients/invalid", nil, gin.Params{{Key: "id", Value: "invalid"}})

		// execute handler
		srv.DeleteOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("NotFoundStore", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnDeleteOIDCClient = func(ctx context.Context, id ulid.ULID) error {
			return errors.ErrNotFound
		}

		// build request and context
		w, c := requestContext(t, http.MethodDelete, "/v1/oidc/oidcclients/"+id.String(), nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.DeleteOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusNotFound, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "oidc client not found", reply.Error)
	})

	t.Run("StoreError", func(t *testing.T) {
		// prepare mocks
		mockStore := openMockStore(t)
		defer mockStore.Close()
		id := ulid.MakeSecure()
		srv := newTestServer(mockStore)

		// set mock callback
		mockStore.OnDeleteOIDCClient = func(ctx context.Context, id ulid.ULID) error {
			return errors.Fmt("db error")
		}

		// build request and context
		w, c := requestContext(t, http.MethodDelete, "/v1/oidc/oidcclients/"+id.String(), nil, gin.Params{{Key: "id", Value: id.String()}})

		// execute handler
		srv.DeleteOIDCClient(c)

		// assert response
		require.Equal(t, http.StatusInternalServerError, w.Code)
		reply := parseReply(t, w)
		require.Equal(t, "could not process delete oidc client request", reply.Error)
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestServer creates a Server with the given store and only the OIDC client routes
// registered, for use in handler tests.
func newTestServer(store store.Store) *Server {
	s := &Server{store: store}
	s.router = gin.New()
	v1 := s.router.Group("/v1")
	oidc := v1.Group("oidc")
	oidcclients := oidc.Group("oidcclients")
	oidcclients.GET("", s.ListOIDCClients)
	oidcclients.POST("", s.CreateOIDCClient)
	oidcclients.GET("/:id", s.OIDCClientDetail)
	oidcclients.PUT("/:id", s.UpdateOIDCClient)
	oidcclients.POST("/:id/revoke", s.RevokeOIDCClient)
	oidcclients.DELETE("/:id", s.DeleteOIDCClient)
	return s
}

// openMockStore opens a mock store; caller must defer store.Close().
func openMockStore(t *testing.T) *mock.Store {
	t.Helper()
	mockStore, err := mock.Open(nil)
	require.NoError(t, err)
	return mockStore
}

// requestContext builds an HTTP request and gin context for handler tests.
// body may be nil for GET/DELETE; params may be nil if no path params.
func requestContext(t *testing.T, method, path string, body []byte, params gin.Params) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	w := httptest.NewRecorder()

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	if params != nil {
		c.Params = params
	}

	return w, c
}

// validCreateBody returns JSON for a minimal valid OIDC client create request.
func validCreateBody() []byte {
	b, _ := json.Marshal(&api.OIDCClient{
		ClientName:   "Test Client",
		RedirectURIs: []string{"https://example.com/cb"},
	})
	return b
}

// validUpdateBody returns JSON for a minimal valid OIDC client update request.
// The id must match the URL param so the handler accepts the body.
func validUpdateBody(id ulid.ULID, clientName string) []byte {
	b, _ := json.Marshal(&api.OIDCClient{
		ID:           id,
		ClientName:   clientName,
		RedirectURIs: []string{"https://example.com/cb"},
	})
	return b
}

// parseReply decodes the response body as api.Reply.
func parseReply(t *testing.T, w *httptest.ResponseRecorder) api.Reply {
	t.Helper()
	var reply api.Reply
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reply))
	return reply
}

// parseOIDCClient decodes the response body as api.OIDCClient.
func parseOIDCClient(t *testing.T, w *httptest.ResponseRecorder) api.OIDCClient {
	t.Helper()
	var out api.OIDCClient
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	return out
}
