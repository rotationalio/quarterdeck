package backend

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var oidcClients = tidal.New[*models.OIDCClient]("oidc_clients")

//===========================================================================
// Store Methods
//===========================================================================

func (s *Store) ListOIDCClients(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error) {
	return list(s, ctx, oidcClients, filter)
}

func (s *Store) CreateOIDCClient(ctx context.Context, client *models.OIDCClient) (*models.OIDCClient, error) {
	var created *models.OIDCClient
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateOIDCClient(client)
		return err
	})
	return created, err
}

func (s *Store) RetrieveOIDCClient(ctx context.Context, id ulid.ULID) (*models.OIDCClient, error) {
	var client *models.OIDCClient
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		client, err = t.RetrieveOIDCClient(id)
		return err
	})
	return client, err
}

func (s *Store) RetrieveOIDCClientByClientID(ctx context.Context, clientID string) (*models.OIDCClient, error) {
	var client *models.OIDCClient
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		client, err = t.RetrieveOIDCClientByClientID(clientID)
		return err
	})
	return client, err
}

func (s *Store) UpdateOIDCClient(ctx context.Context, client *models.OIDCClient) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateOIDCClient(client)
	})
}

func (s *Store) DeleteOIDCClient(ctx context.Context, id ulid.ULID) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.DeleteOIDCClient(id)
	})
}

//===========================================================================
// Tx Methods
//===========================================================================

// ListOIDCClients returns a cursor over OIDC clients matching filter. [tidal.Cursor.Close]
// rolls back the transaction; use [tidal.Cursor.CloseRows] to release the result set and
// continue using this transaction.
func (t *tx) ListOIDCClients(filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error) {
	return listInTx(t, oidcClients, filter)
}

func (t *tx) CreateOIDCClient(client *models.OIDCClient) (*models.OIDCClient, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if !client.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	if _, err := oidcClients.Create(t.tx, client); err != nil {
		return nil, tidalErr(err)
	}

	return t.retrieveOIDCClient(client.ID)
}

func (t *tx) RetrieveOIDCClient(id ulid.ULID) (*models.OIDCClient, error) {
	return t.retrieveOIDCClient(id)
}

func (t *tx) RetrieveOIDCClientByClientID(clientID string) (*models.OIDCClient, error) {
	found, err := retrieveBy(t, oidcClients, "client_id", clientID)
	if err != nil {
		return nil, err
	}
	return t.retrieveOIDCClient(found.ID)
}

func (t *tx) UpdateOIDCClient(client *models.OIDCClient) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return tidalErr(oidcClients.Update(t.tx, client))
}

func (t *tx) DeleteOIDCClient(id ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if id.IsZero() {
		return qerrors.ErrMissingID
	}
	result, err := oidcClients.Delete(t.tx, sql.Named("id", id))
	if err != nil {
		return tidalErr(err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return qerrors.ErrNotFound
	}
	return nil
}

//===========================================================================
// Helpers
//===========================================================================

func (t *tx) retrieveOIDCClient(id ulid.ULID) (*models.OIDCClient, error) {
	client, err := oidcClients.Retrieve(t.tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}
	return client, nil
}
