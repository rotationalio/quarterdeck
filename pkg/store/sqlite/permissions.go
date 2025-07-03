package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

//===========================================================================
// Roles Store
//===========================================================================

const listRolesSQL = `SELECT * FROM roles ORDER BY created DESC`

func (s *Store) ListRoles(ctx context.Context, page *models.Page) (out *models.RoleList, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListRoles(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (tx *Tx) ListRoles(page *models.Page) (out *models.RoleList, err error) {
	// TODO: handle pagination
	out = &models.RoleList{
		Page:  models.PageFrom(page),
		Roles: make([]*models.Role, 0),
	}

	rows, err := tx.Query(listRolesSQL)
	if err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		role := &models.Role{}
		if err = role.Scan(rows); err != nil {
			return nil, err
		}
		out.Roles = append(out.Roles, role)
	}

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return out, nil
}

const createRoleSQL = `INSERT INTO roles (title, description, is_default, created, modified) VALUES (:title, :description, :isDefault, :created, :modified)`

func (s *Store) CreateRole(ctx context.Context, role *models.Role) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateRole(role); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) CreateRole(role *models.Role) (err error) {
	if role.ID != 0 {
		return errors.ErrNoIDOnCreate
	}

	role.Created = time.Now()
	role.Modified = role.Created

	var result sql.Result
	if result, err = tx.Exec(createRoleSQL, role.Params()...); err != nil {
		return dbe(err)
	}

	if role.ID, err = result.LastInsertId(); err != nil {
		return dbe(err)
	}

	// Assign permissions to the role if any are associated.
	var permissions []*models.Permission
	if permissions, err = role.Permissions(); err != nil {
		if !errors.Is(err, errors.ErrMissingAssociation) {
			return err
		}
	}

	for _, permission := range permissions {
		if err = tx.AddPermissionToRole(role.ID, permission); err != nil {
			return fmt.Errorf("invalid permission %q (ID: %d): role not created: %w", permission.Title, permission.ID, err)
		}
	}

	return nil
}

const (
	retrieveRoleByIDSQL    = `SELECT * FROM roles WHERE id=:id`
	retrieveRoleByTitleSQL = `SELECT * FROM roles WHERE title=:title`
)

func (s *Store) RetrieveRole(ctx context.Context, titleOrName any) (out *models.Role, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.RetrieveRole(titleOrName); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (tx *Tx) RetrieveRole(titleOrName any) (out *models.Role, err error) {
	var (
		query string
		param sql.NamedArg
	)

	switch t := titleOrName.(type) {
	case int64:
		query = retrieveRoleByIDSQL
		param = sql.Named("id", t)
	case string:
		query = retrieveRoleByTitleSQL
		param = sql.Named("title", t)
	default:
		return nil, errors.Fmt("invalid type %T for titleOrName", t)
	}

	out = &models.Role{}
	if err = out.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	// Retrieve associated permissions for the role.
	var permissions []*models.Permission
	if permissions, err = tx.rolePermissions(out.ID); err != nil {
		return nil, err
	}
	out.SetPermissions(permissions)

	return out, nil
}

const rolePermissionsSQL = `SELECT p.* FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE rp.role_id=:id`

func (tx *Tx) rolePermissions(roleID int64) (permissions []*models.Permission, err error) {
	if roleID == 0 {
		return nil, errors.ErrMissingID
	}

	rows, err := tx.Query(rolePermissionsSQL, sql.Named("id", roleID))
	if err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		permission := &models.Permission{}
		if err = permission.Scan(rows); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return permissions, nil
}

const updateRoleSQL = `UPDATE roles SET title=:title, description=:description, is_default=:isDefault, modified=:modified WHERE id=:id`

func (s *Store) UpdateRole(ctx context.Context, role *models.Role) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateRole(role); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdateRole(role *models.Role) (err error) {
	if role.ID == 0 {
		return errors.ErrMissingID
	}

	role.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateRoleSQL, role.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const addPermissionToRoleSQL = `INSERT INTO role_permissions (role_id, permission_id, created, modified) VALUES (:roleID, :permissionID, :created, :modified)`

func (s *Store) AddPermissionToRole(ctx context.Context, roleID int64, permission any) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.AddPermissionToRole(roleID, permission); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) AddPermissionToRole(roleID int64, permission any) (err error) {
	// Retrieve the permission to ensure we have a valid ID.
	var resolvedPermission *models.Permission
	if resolvedPermission, err = tx.RetrievePermission(permission); err != nil {
		return err
	}

	created := time.Now()
	params := []any{
		sql.Named("roleID", roleID),
		sql.Named("permissionID", resolvedPermission.ID),
		sql.Named("created", created),
		sql.Named("modified", created),
	}

	if _, err = tx.Exec(addPermissionToRoleSQL, params...); err != nil {
		return dbe(err)
	}

	return nil
}

const removePermissionFromRoleSQL = `DELETE FROM role_permissions WHERE role_id=:roleID AND permission_id=:permissionID`

func (s *Store) RemovePermissionFromRole(ctx context.Context, roleID int64, permissionID int64) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.RemovePermissionFromRole(roleID, permissionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) RemovePermissionFromRole(roleID int64, permissionID int64) (err error) {
	if roleID == 0 || permissionID == 0 {
		return errors.ErrMissingID
	}

	params := []any{
		sql.Named("roleID", roleID),
		sql.Named("permissionID", permissionID),
	}

	if _, err = tx.Exec(removePermissionFromRoleSQL, params...); err != nil {
		return dbe(err)
	}

	return nil
}

const deleteRoleSQL = `DELETE FROM roles WHERE id=:id`

func (s *Store) DeleteRole(ctx context.Context, roleID int64) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteRole(roleID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) DeleteRole(roleID int64) (err error) {
	if roleID == 0 {
		return errors.ErrMissingID
	}

	var result sql.Result
	if result, err = tx.Exec(deleteRoleSQL, sql.Named("id", roleID)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

//===========================================================================
// Permissions Store
//===========================================================================

const listPermissionsSQL = `SELECT * FROM permissions ORDER BY title ASC`

func (s *Store) ListPermissions(ctx context.Context, page *models.Page) (out *models.PermissionList, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListPermissions(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (tx *Tx) ListPermissions(page *models.Page) (out *models.PermissionList, err error) {
	// TODO: handle pagination
	out = &models.PermissionList{
		Page:        models.PageFrom(page),
		Permissions: make([]*models.Permission, 0),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listPermissionsSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		// Scan permission summary into a new Permission struct.
		permission := &models.Permission{}
		if err = permission.Scan(rows); err != nil {
			return nil, err
		}
		out.Permissions = append(out.Permissions, permission)
	}

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return out, nil
}

const createPermissionSQL = `INSERT INTO permissions (title, description, created, modified) VALUES (:title, :description, :created, :modified)`

func (s *Store) CreatePermission(ctx context.Context, permission *models.Permission) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreatePermission(permission); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) CreatePermission(permission *models.Permission) (err error) {
	if permission.ID != 0 {
		return errors.ErrNoIDOnCreate
	}

	if permission.Title == "" {
		return errors.ErrZeroValuedNotNull
	}

	permission.Created = time.Now()
	permission.Modified = permission.Created

	var result sql.Result
	if result, err = tx.Exec(createPermissionSQL, permission.Params()...); err != nil {
		return dbe(err)
	}

	if permission.ID, err = result.LastInsertId(); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	retrievePermissionByIDSQL    = "SELECT * FROM permissions WHERE id=:id"
	retrievePermissionByTitleSQL = "SELECT * FROM permissions WHERE title=:title"
)

func (s *Store) RetrievePermission(ctx context.Context, titleOrID any) (permission *models.Permission, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if permission, err = tx.RetrievePermission(titleOrID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return permission, nil
}

func (tx *Tx) RetrievePermission(titleOrID any) (permission *models.Permission, err error) {
	var (
		query string
		param sql.NamedArg
	)

	switch t := titleOrID.(type) {
	case int:
		if t == 0 {
			return nil, errors.ErrMissingID
		}

		query = retrievePermissionByIDSQL
		param = sql.Named("id", int64(t)) // Convert int to int64 for SQL query
	case int64:
		if t == 0 {
			return nil, errors.ErrMissingID
		}

		query = retrievePermissionByIDSQL
		param = sql.Named("id", t)
	case string:
		if t == "" {
			return nil, errors.ErrMissingID
		}

		query = retrievePermissionByTitleSQL
		param = sql.Named("title", t)
	case *models.Permission:
		if t.ID == 0 && t.Title == "" {
			return nil, errors.ErrMissingID
		}

		if t.ID != 0 {
			query = retrievePermissionByIDSQL
			param = sql.Named("id", t.ID)
		} else {
			query = retrievePermissionByTitleSQL
			param = sql.Named("title", t.Title)
		}
	default:
		return nil, errors.Fmt("invalid type %T for titleOrID", t)
	}

	permission = &models.Permission{}
	if err = permission.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	return permission, nil
}

const updatePermissionSQL = `UPDATE permissions SET title=:title, description=:description, modified=:modified WHERE id=:id`

func (s *Store) UpdatePermission(ctx context.Context, permission *models.Permission) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdatePermission(permission); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdatePermission(permission *models.Permission) (err error) {
	if permission.ID == 0 {
		return errors.ErrMissingID
	}

	permission.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updatePermissionSQL, permission.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}
	return nil
}

const deletePermissionSQL = `DELETE FROM permissions WHERE id=:id`

func (s *Store) DeletePermission(ctx context.Context, permissionID int64) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeletePermission(permissionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) DeletePermission(permissionID int64) (err error) {
	if permissionID == 0 {
		return errors.ErrMissingID
	}

	var result sql.Result
	if result, err = tx.Exec(deletePermissionSQL, sql.Named("id", permissionID)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}
	return nil
}
