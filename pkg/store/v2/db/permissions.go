package db

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
)

var permissions = tidal.New[*models.Permission]("permissions")

func (d *DB) ListPermissions(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error) {
	return list(d, ctx, permissions, filter)
}

func (d *DB) CreatePermission(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	if permission.ID != 0 {
		return nil, qerrors.ErrNoIDOnCreate
	}

	var created *models.Permission
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := permissions.Create(tx, permission)
		if err != nil {
			return tidalErr(err)
		}

		if err = captureInsertID(tx, d, result, func(id int64) { permission.ID = id }); err != nil {
			return err
		}

		created, err = permissions.Retrieve(tx, sql.Named("id", permission.ID))
		return tidalErr(err)
	})
	return created, err
}

func (d *DB) RetrievePermission(ctx context.Context, id int64) (*models.Permission, error) {
	var permission *models.Permission
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		permission, err = permissions.Retrieve(tx, sql.Named("id", id))
		return tidalErr(err)
	})
	return permission, err
}

func (d *DB) RetrievePermissionByTitle(ctx context.Context, title string) (*models.Permission, error) {
	var permission *models.Permission
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		permission, err = d.retrievePermissionByTitleTx(tx, title)
		return err
	})
	return permission, err
}

func (d *DB) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return tidalErr(permissions.Update(tx, permission))
	})
}

func (d *DB) DeletePermission(ctx context.Context, id int64) error {
	if id == 0 {
		return qerrors.ErrMissingID
	}
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := permissions.Delete(tx, sql.Named("id", id))
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

func (d *DB) retrievePermissionByTitleTx(tx tidal.Tx, title string) (*models.Permission, error) {
	permission, err := retrieveBy(tx, permissions, "title", title)
	if err != nil {
		return nil, err
	}
	return permission, nil
}
