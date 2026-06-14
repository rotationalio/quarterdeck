package db

import (
	"context"
	"database/sql"
	"time"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var users = tidal.New[*models.User]("users")
var userRoles = tidal.New[*models.UserRole]("user_roles")

const (
	userRolesSQL       = `SELECT r.id, r.title, r.description, r.is_default, r.created, r.modified FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = :user_id`
	userPermissionsSQL = `SELECT DISTINCT p.id, p.title, p.description, p.created, p.modified
	FROM user_permissions up JOIN permissions p ON p.title = up.permission WHERE up.user_id = :user_id ORDER BY p.title`
	defaultRolesSQL            = `SELECT id FROM roles WHERE is_default = 't' OR is_default = true`
	updateUserPasswordSQL      = `UPDATE users SET password = :password, modified = :modified WHERE id = :id`
	updateUserLastLoginSQL     = `UPDATE users SET last_login = :last_login, modified = :modified WHERE id = :id`
	updateUserEmailVerifiedSQL = `UPDATE users SET email_verified = :email_verified, modified = :modified WHERE id = :id`
	deleteUserRoleSQL          = `DELETE FROM user_roles WHERE user_id = :user_id AND role_id = :role_id`
	deleteUserRolesByUserSQL   = `DELETE FROM user_roles WHERE user_id = :user_id`
)

func (d *DB) ListUsers(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.User], error) {
	return list(d, ctx, users, filter)
}

func (d *DB) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	if !user.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	var created *models.User
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		if _, err := users.Create(tx, user); err != nil {
			return tidalErr(err)
		}

		rolesToAssign := user.Roles
		if len(rolesToAssign) == 0 {
			var err error
			rolesToAssign, err = d.defaultRolesTx(tx)
			if err != nil {
				return err
			}
		}

		for i := range rolesToAssign {
			roleID, err := d.resolveRoleIDTx(tx, &rolesToAssign[i])
			if err != nil {
				return err
			}
			if err := d.addRoleToUserTx(tx, user.ID, roleID); err != nil {
				return err
			}
		}

		var err error
		created, err = d.retrieveUserTx(tx, user.ID)
		return err
	})
	return created, err
}

func (d *DB) RetrieveUser(ctx context.Context, id ulid.ULID) (*models.User, error) {
	var user *models.User
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		user, err = d.retrieveUserTx(tx, id)
		return err
	})
	return user, err
}

func (d *DB) RetrieveUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user *models.User
	err := d.withReadTx(ctx, func(tx tidal.Tx) error {
		found, err := retrieveBy(tx, users, "email", email)
		if err != nil {
			return err
		}
		user, err = d.retrieveUserTx(tx, found.ID)
		return err
	})
	return user, err
}

func (d *DB) UpdateUser(ctx context.Context, user *models.User) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return tidalErr(users.Update(tx, user))
	})
}

func (d *DB) UpdatePassword(ctx context.Context, userID ulid.ULID, password string) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := tx.Exec(
			updateUserPasswordSQL,
			sql.Named("id", userID),
			sql.Named("password", password),
			sql.Named("modified", time.Now().UTC()),
		)
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

func (d *DB) UpdateLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := tx.Exec(
			updateUserLastLoginSQL,
			sql.Named("id", userID),
			sql.Named("last_login", sql.NullTime{Time: lastLogin, Valid: !lastLogin.IsZero()}),
			sql.Named("modified", time.Now().UTC()),
		)
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

func (d *DB) VerifyEmail(ctx context.Context, userID ulid.ULID) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := tx.Exec(
			updateUserEmailVerifiedSQL,
			sql.Named("id", userID),
			sql.Named("email_verified", true),
			sql.Named("modified", time.Now().UTC()),
		)
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

func (d *DB) DeleteUser(ctx context.Context, userID ulid.ULID) error {
	if userID.IsZero() {
		return qerrors.ErrMissingID
	}
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := users.Delete(tx, sql.Named("id", userID))
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

func (d *DB) AddRoleToUser(ctx context.Context, userID ulid.ULID, roleID int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return d.addRoleToUserTx(tx, userID, roleID)
	})
}

func (d *DB) AddRoleToUserByTitle(ctx context.Context, userID ulid.ULID, title string) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		role, err := d.retrieveRoleByTitleTx(tx, title)
		if err != nil {
			return err
		}
		return d.addRoleToUserTx(tx, userID, role.ID)
	})
}

func (d *DB) RemoveRoleFromUser(ctx context.Context, userID ulid.ULID, roleID int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		_, err := tx.Exec(
			deleteUserRoleSQL,
			sql.Named("user_id", userID),
			sql.Named("role_id", roleID),
		)
		return tidalErr(err)
	})
}

func (d *DB) RemoveRoleFromUserByTitle(ctx context.Context, userID ulid.ULID, title string) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		role, err := d.retrieveRoleByTitleTx(tx, title)
		if err != nil {
			return err
		}
		_, err = tx.Exec(
			deleteUserRoleSQL,
			sql.Named("user_id", userID),
			sql.Named("role_id", role.ID),
		)
		return tidalErr(err)
	})
}

func (d *DB) ReplaceUserRoles(ctx context.Context, userID ulid.ULID, roleIDs []int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		if _, err := tx.Exec(deleteUserRolesByUserSQL, sql.Named("user_id", userID)); err != nil {
			return tidalErr(err)
		}
		for _, roleID := range roleIDs {
			if err := d.addRoleToUserTx(tx, userID, roleID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) retrieveUserTx(tx tidal.Tx, id ulid.ULID) (*models.User, error) {
	user, err := users.Retrieve(tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}

	roles, err := d.userRolesTx(tx, id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	perms, err := d.userPermissionsTx(tx, id)
	if err != nil {
		return nil, err
	}
	user.Permissions = perms
	return user, nil
}

func (d *DB) userRolesTx(tx tidal.Tx, userID ulid.ULID) ([]models.Role, error) {
	rows, err := tx.Query(userRolesSQL, sql.Named("user_id", userID))
	if err != nil {
		return nil, tidalErr(err)
	}
	defer rows.Close()

	roles := make([]models.Role, 0)
	for rows.Next() {
		role := models.Role{}
		if err = role.Scan(tidal.Retrieve, rows); err != nil {
			return nil, tidalErr(err)
		}
		roles = append(roles, role)
	}
	return roles, tidalErr(rows.Err())
}

func (d *DB) userPermissionsTx(tx tidal.Tx, userID ulid.ULID) ([]models.Permission, error) {
	rows, err := tx.Query(userPermissionsSQL, sql.Named("user_id", userID))
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

func (d *DB) defaultRolesTx(tx tidal.Tx) ([]models.Role, error) {
	rows, err := tx.Query(defaultRolesSQL)
	if err != nil {
		return nil, tidalErr(err)
	}
	defer rows.Close()

	roles := make([]models.Role, 0)
	for rows.Next() {
		role := models.Role{}
		if err = rows.Scan(&role.ID); err != nil {
			return nil, tidalErr(err)
		}
		roles = append(roles, role)
	}
	return roles, tidalErr(rows.Err())
}

func (d *DB) addRoleToUserTx(tx tidal.Tx, userID ulid.ULID, roleID int64) error {
	junction := &models.UserRole{UserID: userID, RoleID: roleID}
	_, err := userRoles.Create(tx, junction)
	return tidalErr(err)
}
