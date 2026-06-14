package backend

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
)

var permissions = tidal.New[*models.Permission]("permissions")

//===========================================================================
// Store Methods
//===========================================================================

func (s *Store) ListPermissions(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error) {
	return list(s, ctx, permissions, filter)
}

func (s *Store) CreatePermission(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	var created *models.Permission
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreatePermission(permission)
		return err
	})
	return created, err
}

func (s *Store) RetrievePermission(ctx context.Context, id int64) (*models.Permission, error) {
	var permission *models.Permission
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		permission, err = t.RetrievePermission(id)
		return err
	})
	return permission, err
}

func (s *Store) RetrievePermissionByTitle(ctx context.Context, title string) (*models.Permission, error) {
	var permission *models.Permission
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		permission, err = t.RetrievePermissionByTitle(title)
		return err
	})
	return permission, err
}

func (s *Store) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdatePermission(permission)
	})
}

func (s *Store) DeletePermission(ctx context.Context, id int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.DeletePermission(id)
	})
}

//===========================================================================
// Tx Methods
//===========================================================================

// ListPermissions returns a cursor over permissions matching filter. [tidal.Cursor.Close]
// rolls back the transaction; use [tidal.Cursor.CloseRows] to release the result set and
// continue using this transaction.
func (t *tx) ListPermissions(filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error) {
	return listInTx(t, permissions, filter)
}

func (t *tx) CreatePermission(permission *models.Permission) (*models.Permission, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if permission.ID != 0 {
		return nil, qerrors.ErrNoIDOnCreate
	}

	result, err := permissions.Create(t.tx, permission)
	if err != nil {
		return nil, tidalErr(err)
	}

	if err = captureInsertID(t, result, func(id int64) { permission.ID = id }); err != nil {
		return nil, err
	}

	return permissions.Retrieve(t.tx, sql.Named("id", permission.ID))
}

func (t *tx) RetrievePermission(id int64) (*models.Permission, error) {
	permission, err := permissions.Retrieve(t.tx, sql.Named("id", id))
	return permission, tidalErr(err)
}

func (t *tx) RetrievePermissionByTitle(title string) (*models.Permission, error) {
	return t.retrievePermissionByTitle(title)
}

func (t *tx) UpdatePermission(permission *models.Permission) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return tidalErr(permissions.Update(t.tx, permission))
}

func (t *tx) DeletePermission(id int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if id == 0 {
		return qerrors.ErrMissingID
	}
	result, err := permissions.Delete(t.tx, sql.Named("id", id))
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

func (t *tx) retrievePermissionByTitle(title string) (*models.Permission, error) {
	permission, err := retrieveBy(t, permissions, "title", title)
	if err != nil {
		return nil, err
	}
	return permission, nil
}
