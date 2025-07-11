package sqlite

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

const (
	createVeroSQL = "INSERT INTO vero_tokens (id, token_type, resource_id, email, expiration, signature, sent_on, created, modified) VALUES (:id, :tokenType, :resourceID, :email, :expiration, :signature, :sentOn, :created, :modified)"
)

func (s *Store) CreateVeroToken(ctx context.Context, token *models.VeroToken) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateVeroToken(token); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) CreateVeroToken(token *models.VeroToken) (err error) {
	if !token.ID.IsZero() {
		return errors.ErrNoIDOnCreate
	}

	token.ID = ulid.MakeSecure()
	token.Created = time.Now()
	token.Modified = token.Created

	if _, err = tx.Exec(createVeroSQL, token.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	retrieveVeroTokenSQL = "SELECT * from vero_tokens WHERE id=:id"
)

func (s *Store) RetrieveVeroToken(ctx context.Context, id ulid.ULID) (token *models.VeroToken, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if token, err = tx.RetrieveVeroToken(id); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return token, nil
}

func (tx *Tx) RetrieveVeroToken(id ulid.ULID) (token *models.VeroToken, err error) {
	if id.IsZero() {
		return nil, errors.ErrMissingID
	}

	token = &models.VeroToken{}
	if err = token.Scan(tx.QueryRow(retrieveVeroTokenSQL, sql.Named("id", id))); err != nil {
		return nil, dbe(err)
	}

	return token, nil
}

const (
	updateVeroSQL = "UPDATE vero_tokens SET token_type=:tokenType, resource_id=:resourceID, email=:email, expiration=:expiration, signature=:signature, sent_on=:sentOn, modified=:modified WHERE id=:id"
)

func (s *Store) UpdateVeroToken(ctx context.Context, token *models.VeroToken) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateVeroToken(token); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdateVeroToken(token *models.VeroToken) (err error) {
	if token.ID.IsZero() {
		return errors.ErrMissingID
	}

	token.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateVeroSQL, token.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const (
	deleteVeroSQL = "DELETE FROM vero_tokens WHERE id=:id"
)

func (s *Store) DeleteVeroToken(ctx context.Context, id ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteVeroToken(id); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) DeleteVeroToken(id ulid.ULID) (err error) {
	if id.IsZero() {
		return errors.ErrMissingID
	}

	var result sql.Result
	if result, err = tx.Exec(deleteVeroSQL, sql.Named("id", id)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}
