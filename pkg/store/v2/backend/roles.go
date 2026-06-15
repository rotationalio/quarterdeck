package backend

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
)

var roles = tidal.New[*models.Role]("roles")
var rolePermissions = tidal.New[*models.RolePermission]("role_permissions")

const (
	rolePermissionsSQL      = `SELECT p.id, p.title, p.description, p.created, p.modified FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE rp.role_id = :role_id`
	deleteRolePermissionSQL = `DELETE FROM role_permissions WHERE role_id = :role_id AND permission_id = :permission_id`
)

//===========================================================================
// Store Methods
//===========================================================================

func (s *Store) ListRoles(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Role], error) {
	return list(s, ctx, roles, filter)
}

func (s *Store) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	var created *models.Role
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateRole(role)
		return err
	})
	return created, err
}

func (s *Store) RetrieveRole(ctx context.Context, id int64) (*models.Role, error) {
	var role *models.Role
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		role, err = t.RetrieveRole(id)
		return err
	})
	return role, err
}

func (s *Store) RetrieveRoleByTitle(ctx context.Context, title string) (*models.Role, error) {
	var role *models.Role
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		role, err = t.RetrieveRoleByTitle(title)
		return err
	})
	return role, err
}

func (s *Store) UpdateRole(ctx context.Context, role *models.Role) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateRole(role)
	})
}

func (s *Store) AddPermissionToRole(ctx context.Context, roleID int64, permissionID int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.AddPermissionToRole(roleID, permissionID)
	})
}

func (s *Store) AddPermissionToRoleByTitle(ctx context.Context, roleID int64, title string) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.AddPermissionToRoleByTitle(roleID, title)
	})
}

func (s *Store) RemovePermissionFromRole(ctx context.Context, roleID int64, permissionID int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.RemovePermissionFromRole(roleID, permissionID)
	})
}

func (s *Store) DeleteRole(ctx context.Context, id int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.DeleteRole(id)
	})
}

//===========================================================================
// Tx Methods
//===========================================================================

// ListRoles returns a cursor over roles matching filter. [tidal.Cursor.Close] rolls back
// the transaction; use [tidal.Cursor.CloseRows] to release the result set and continue
// using this transaction.
func (t *tx) ListRoles(filter tidal.ListFilter) (tidal.Cursor[*models.Role], error) {
	return listInTx(t, roles, filter)
}

func (t *tx) CreateRole(role *models.Role) (*models.Role, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if role.ID != 0 {
		return nil, qerrors.ErrNoIDOnCreate
	}

	result, err := roles.Create(t.tx, role)
	if err != nil {
		return nil, tidalErr(err)
	}

	if err = captureInsertID(t, result, func(id int64) { role.ID = id }); err != nil {
		return nil, err
	}

	for _, permission := range role.Permissions {
		permID := permission.ID
		if permID == 0 && permission.Title != "" {
			resolved, err := t.retrievePermissionByTitle(permission.Title)
			if err != nil {
				return nil, err
			}
			permID = resolved.ID
		}
		if err = t.addPermissionToRole(role.ID, permID); err != nil {
			return nil, err
		}
	}

	return t.retrieveRole(role.ID)
}

func (t *tx) RetrieveRole(id int64) (*models.Role, error) {
	return t.retrieveRole(id)
}

func (t *tx) RetrieveRoleByTitle(title string) (*models.Role, error) {
	return t.retrieveRoleByTitle(title)
}

func (t *tx) UpdateRole(role *models.Role) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return tidalErr(roles.Update(t.tx, role))
}

func (t *tx) AddPermissionToRole(roleID int64, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.addPermissionToRole(roleID, permissionID)
}

func (t *tx) AddPermissionToRoleByTitle(roleID int64, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	permission, err := t.retrievePermissionByTitle(title)
	if err != nil {
		return err
	}
	return t.addPermissionToRole(roleID, permission.ID)
}

func (t *tx) RemovePermissionFromRole(roleID int64, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	_, err := t.tx.Exec(
		deleteRolePermissionSQL,
		sql.Named("role_id", roleID),
		sql.Named("permission_id", permissionID),
	)
	return tidalErr(err)
}

func (t *tx) DeleteRole(id int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if id == 0 {
		return qerrors.ErrMissingID
	}
	result, err := roles.Delete(t.tx, sql.Named("id", id))
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

func (t *tx) retrieveRole(id int64) (*models.Role, error) {
	role, err := roles.Retrieve(t.tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}

	permissions, err := t.rolePermissions(id)
	if err != nil {
		return nil, err
	}
	role.Permissions = permissions
	return role, nil
}

func (t *tx) retrieveRoleByTitle(title string) (*models.Role, error) {
	role, err := retrieveBy(t, roles, "title", title)
	if err != nil {
		return nil, err
	}
	return t.retrieveRole(role.ID)
}

func (t *tx) resolveRoleID(role *models.Role) (int64, error) {
	if role.ID != 0 {
		return role.ID, nil
	}
	if role.Title == "" {
		return 0, qerrors.ErrMissingID
	}
	resolved, err := t.retrieveRoleByTitle(role.Title)
	if err != nil {
		return 0, err
	}
	return resolved.ID, nil
}

func (t *tx) rolePermissions(roleID int64) ([]models.Permission, error) {
	rows, err := t.tx.Query(rolePermissionsSQL, sql.Named("role_id", roleID))
	if err != nil {
		return nil, tidalErr(err)
	}
	defer rows.Close()

	permissions := make([]models.Permission, 0)
	for rows.Next() {
		permission := models.Permission{}
		if err = permission.Scan(tidal.Retrieve, rows); err != nil {
			return nil, tidalErr(err)
		}
		permissions = append(permissions, permission)
	}
	return permissions, tidalErr(rows.Err())
}

func (t *tx) addPermissionToRole(roleID int64, permissionID int64) error {
	junction := &models.RolePermission{RoleID: roleID, PermissionID: permissionID}
	_, err := rolePermissions.Create(t.tx, junction)
	return tidalErr(err)
}
