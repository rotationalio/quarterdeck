package sqlite

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// OIDCClient Tx
//===========================================================================

const (
	listOIDCClientsSQL = "SELECT id, client_name, client_uri, logo_uri, policy_uri, tos_uri, redirect_uris, contacts, client_id, created_by, revoked, created, modified FROM oidc_clients WHERE revoked IS NULL ORDER BY created DESC"
)

func (tx *Tx) ListOIDCClients(page *models.Page) (out *models.OIDCClientList, err error) {
	out = &models.OIDCClientList{
		OIDCClients: make([]*models.OIDCClient, 0),
		Page:        models.PageFrom(page),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listOIDCClientsSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		client := &models.OIDCClient{}
		if err = client.ScanSummary(rows); err != nil {
			return nil, err
		}
		out.OIDCClients = append(out.OIDCClients, client)
	}

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return out, nil
}

const (
	createOIDCClientSQL = "INSERT INTO oidc_clients (id, client_name, client_uri, logo_uri, policy_uri, tos_uri, redirect_uris, contacts, client_id, secret, created_by, revoked, created, modified) VALUES (:id, :clientName, :clientURI, :logoURI, :policyURI, :tosURI, :redirectURIs, :contacts, :clientID, :secret, :createdBy, :revoked, :created, :modified)"
)

func (tx *Tx) CreateOIDCClient(client *models.OIDCClient) (err error) {
	if !client.ID.IsZero() {
		return errors.ErrNoIDOnCreate
	}

	if err = client.Validate(); err != nil {
		return err
	}

	client.ID = ulid.MakeSecure()
	client.Created = time.Now()
	client.Modified = client.Created

	if _, err = tx.Exec(createOIDCClientSQL, client.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	retrieveOIDCClientByClientIDSQL = "SELECT id, client_name, client_uri, logo_uri, policy_uri, tos_uri, redirect_uris, contacts, client_id, secret, created_by, revoked, created, modified FROM oidc_clients WHERE client_id=:clientID"
	retrieveOIDCClientByIDSQL       = "SELECT id, client_name, client_uri, logo_uri, policy_uri, tos_uri, redirect_uris, contacts, client_id, secret, created_by, revoked, created, modified FROM oidc_clients WHERE id=:id"
)

func (tx *Tx) RetrieveOIDCClient(id any) (client *models.OIDCClient, err error) {
	var (
		query string
		param sql.NamedArg
	)

	switch t := id.(type) {
	case string:
		if t == "" {
			return nil, errors.ErrMissingID
		}

		query = retrieveOIDCClientByClientIDSQL
		param = sql.Named("clientID", t)
	case ulid.ULID:
		if t.IsZero() {
			return nil, errors.ErrMissingID
		}

		query = retrieveOIDCClientByIDSQL
		param = sql.Named("id", t)
	default:
		return nil, errors.Fmt("invalid type %T for OIDC client ID", id)
	}

	client = &models.OIDCClient{}
	if err = client.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	return client, nil
}

const (
	updateOIDCClientSQL = "UPDATE oidc_clients SET client_name=:clientName, client_uri=:clientURI, logo_uri=:logoURI, policy_uri=:policyURI, tos_uri=:tosURI, redirect_uris=:redirectURIs, contacts=:contacts, modified=:modified WHERE id=:id"
)

func (tx *Tx) UpdateOIDCClient(client *models.OIDCClient) (err error) {
	if client.ID.IsZero() {
		return errors.ErrMissingID
	}

	if err = client.Validate(); err != nil {
		return err
	}

	client.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateOIDCClientSQL, client.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const (
	revokeOIDCClientSQL = "UPDATE oidc_clients SET revoked=:revoked, modified=:modified WHERE id=:id"
)

func (tx *Tx) RevokeOIDCClient(id ulid.ULID) (err error) {
	if id.IsZero() {
		return errors.ErrMissingID
	}

	now := time.Now()
	params := []any{
		sql.Named("id", id),
		sql.Named("revoked", sql.NullTime{Time: now, Valid: true}),
		sql.Named("modified", now),
	}

	var result sql.Result
	if result, err = tx.Exec(revokeOIDCClientSQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const (
	deleteOIDCClientSQL = "DELETE FROM oidc_clients WHERE id=:id"
)

func (tx *Tx) DeleteOIDCClient(id ulid.ULID) (err error) {
	if id.IsZero() {
		return errors.ErrMissingID
	}

	var result sql.Result
	if result, err = tx.Exec(deleteOIDCClientSQL, sql.Named("id", id)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

//===========================================================================
// OIDCClient Store
//===========================================================================

func (s *Store) ListOIDCClients(ctx context.Context, page *models.Page) (out *models.OIDCClientList, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListOIDCClients(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) CreateOIDCClient(ctx context.Context, client *models.OIDCClient) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateOIDCClient(client); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) RetrieveOIDCClient(ctx context.Context, id any) (client *models.OIDCClient, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if client, err = tx.RetrieveOIDCClient(id); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return client, nil
}

func (s *Store) UpdateOIDCClient(ctx context.Context, client *models.OIDCClient) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateOIDCClient(client); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) RevokeOIDCClient(ctx context.Context, id ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.RevokeOIDCClient(id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) DeleteOIDCClient(ctx context.Context, id ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteOIDCClient(id); err != nil {
		return err
	}

	return tx.Commit()
}
