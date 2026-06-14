package db

import (
	"context"
	"database/sql"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
)

var roles = tidal.New[*models.Role]("roles")
var rolePermissions = tidal.New[*models.RolePermission]("role_permissions")

const (
	rolePermissionsSQL = `SELECT p.id, p.title, p.description, p.created, p.modified
	FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE rp.role_id = :role_id`
	deleteRolePermissionSQL = `DELETE FROM role_permissions WHERE role_id = :role_id AND permission_id = :permission_id`
)

func (d *DB) ListRoles(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Role], error) {
	return list(d, ctx, roles, filter)
}

func (d *DB) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	if role.ID != 0 {
		return nil, qerrors.ErrNoIDOnCreate
	}

	var created *models.Role
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := roles.Create(tx, role)
		if err != nil {
			return tidalErr(err)
		}

		if err = captureInsertID(tx, d, result, func(id int64) { role.ID = id }); err != nil {
			return err
		}

		for _, permission := range role.Permissions {
			permID := permission.ID
			if permID == 0 && permission.Title != "" {
				resolved, err := d.retrievePermissionByTitleTx(tx, permission.Title)
				if err != nil {
					return err
				}
				permID = resolved.ID
			}
			if err = d.addPermissionToRoleTx(tx, role.ID, permID); err != nil {
				return err
			}
		}

		created, err = d.retrieveRoleTx(tx, role.ID)
		return err
	})
	return created, err
}

func (d *DB) RetrieveRole(ctx context.Context, id int64) (*models.Role, error) {
	var role *models.Role
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		role, err = d.retrieveRoleTx(tx, id)
		return err
	})
	return role, err
}

func (d *DB) RetrieveRoleByTitle(ctx context.Context, title string) (*models.Role, error) {
	var role *models.Role
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		role, err = d.retrieveRoleByTitleTx(tx, title)
		return err
	})
	return role, err
}

func (d *DB) UpdateRole(ctx context.Context, role *models.Role) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return tidalErr(roles.Update(tx, role))
	})
}

func (d *DB) AddPermissionToRole(ctx context.Context, roleID int64, permissionID int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return d.addPermissionToRoleTx(tx, roleID, permissionID)
	})
}

func (d *DB) AddPermissionToRoleByTitle(ctx context.Context, roleID int64, title string) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		permission, err := d.retrievePermissionByTitleTx(tx, title)
		if err != nil {
			return err
		}
		return d.addPermissionToRoleTx(tx, roleID, permission.ID)
	})
}

func (d *DB) RemovePermissionFromRole(ctx context.Context, roleID int64, permissionID int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		_, err := tx.Exec(
			deleteRolePermissionSQL,
			sql.Named("role_id", roleID),
			sql.Named("permission_id", permissionID),
		)
		return tidalErr(err)
	})
}

func (d *DB) DeleteRole(ctx context.Context, id int64) error {
	if id == 0 {
		return qerrors.ErrMissingID
	}
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := roles.Delete(tx, sql.Named("id", id))
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

func (d *DB) retrieveRoleTx(tx tidal.Tx, id int64) (*models.Role, error) {
	role, err := roles.Retrieve(tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}

	permissions, err := d.rolePermissionsTx(tx, id)
	if err != nil {
		return nil, err
	}
	role.Permissions = permissions
	return role, nil
}

func (d *DB) retrieveRoleByTitleTx(tx tidal.Tx, title string) (*models.Role, error) {
	role, err := retrieveBy(tx, roles, "title", title)
	if err != nil {
		return nil, err
	}
	return d.retrieveRoleTx(tx, role.ID)
}

func (d *DB) resolveRoleIDTx(tx tidal.Tx, role *models.Role) (int64, error) {
	if role.ID != 0 {
		return role.ID, nil
	}
	if role.Title == "" {
		return 0, qerrors.ErrMissingID
	}
	resolved, err := d.retrieveRoleByTitleTx(tx, role.Title)
	if err != nil {
		return 0, err
	}
	return resolved.ID, nil
}

func (d *DB) rolePermissionsTx(tx tidal.Tx, roleID int64) ([]models.Permission, error) {
	rows, err := tx.Query(rolePermissionsSQL, sql.Named("role_id", roleID))
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

func (d *DB) addPermissionToRoleTx(tx tidal.Tx, roleID int64, permissionID int64) error {
	junction := &models.RolePermission{RoleID: roleID, PermissionID: permissionID}
	_, err := rolePermissions.Create(tx, junction)
	return tidalErr(err)
}
