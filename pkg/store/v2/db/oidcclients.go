package db

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var oidcClients = tidal.New[*models.OIDCClient]("oidc_clients")

func (d *DB) ListOIDCClients(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error) {
	return list(d, ctx, oidcClients, filter)
}

func (d *DB) CreateOIDCClient(ctx context.Context, client *models.OIDCClient) (*models.OIDCClient, error) {
	if !client.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	var created *models.OIDCClient
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		if _, err := oidcClients.Create(tx, client); err != nil {
			return tidalErr(err)
		}
		var err error
		created, err = d.retrieveOIDCClientTx(tx, client.ID)
		return err
	})
	return created, err
}

func (d *DB) RetrieveOIDCClient(ctx context.Context, id ulid.ULID) (*models.OIDCClient, error) {
	var client *models.OIDCClient
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		client, err = d.retrieveOIDCClientTx(tx, id)
		return err
	})
	return client, err
}

func (d *DB) RetrieveOIDCClientByClientID(ctx context.Context, clientID string) (*models.OIDCClient, error) {
	var client *models.OIDCClient
	err := d.withReadTx(ctx, func(tx tidal.Tx) error {
		found, err := retrieveBy(tx, oidcClients, "client_id", clientID)
		if err != nil {
			return err
		}
		client, err = d.retrieveOIDCClientTx(tx, found.ID)
		return err
	})
	return client, err
}

func (d *DB) UpdateOIDCClient(ctx context.Context, client *models.OIDCClient) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return tidalErr(oidcClients.Update(tx, client))
	})
}

func (d *DB) DeleteOIDCClient(ctx context.Context, id ulid.ULID) error {
	if id.IsZero() {
		return qerrors.ErrMissingID
	}
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := oidcClients.Delete(tx, sql.Named("id", id))
		if err != nil {
			return tidalErr(err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return qerrors.ErrNotFound
		}
		return nil
	})
}

func (d *DB) retrieveOIDCClientTx(tx tidal.Tx, id ulid.ULID) (*models.OIDCClient, error) {
	client, err := oidcClients.Retrieve(tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}
	return client, nil
}
