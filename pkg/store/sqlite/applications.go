package sqlite

import (
	"context"
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

// ############################################################################
// Application Transaction
// ############################################################################

func (tx *Tx) ListApplications(*models.Page) (*models.ApplicationList, error) {
	//FIXME: TODO
	return nil, nil
}

func (tx *Tx) CreateApplication(*models.Application) error {
	//FIXME: TODO
	return nil
}

func (tx *Tx) RetrieveApplication(ulidOrClientID any) (*models.Application, error) {
	//FIXME: TODO
	return nil, nil
}

func (tx *Tx) UpdateApplication(*models.Application) error {
	//FIXME: TODO
	return nil
}

func (tx *Tx) DeleteApplication(ulidOrClientID any) error {
	//FIXME: TODO
	return nil
}

// ############################################################################
// Application Store
// ############################################################################

func (s *Store) ListApplications(ctx context.Context, page *models.Page) (out *models.ApplicationList, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListApplications(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) CreateApplication(ctx context.Context, application *models.Application) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateApplication(application); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) RetrieveApplication(ctx context.Context, ulidOrClientID any) (out *models.Application, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.RetrieveApplication(ulidOrClientID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) UpdateApplication(ctx context.Context, application *models.Application) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateApplication(application); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) DeleteApplication(ctx context.Context, ulidOrClientID any) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteApplication(ulidOrClientID); err != nil {
		return err
	}

	return tx.Commit()
}
