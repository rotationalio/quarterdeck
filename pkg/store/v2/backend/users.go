package backend

import (
	"context"
	"database/sql"
	"time"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var users = tidal.New[*models.User]("users")
var userRoles = tidal.New[*models.UserRole]("user_roles")

const (
	userRolesSQL               = `SELECT r.id, r.title, r.description, r.is_default, r.created, r.modified FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = :user_id`
	userPermissionsSQL         = `SELECT DISTINCT p.id, p.title, p.description, p.created, p.modified FROM user_permissions up JOIN permissions p ON p.title = up.permission WHERE up.user_id = :user_id ORDER BY p.title`
	defaultRolesSQL            = `SELECT id FROM roles WHERE is_default = 't' OR is_default = true`
	updateUserPasswordSQL      = `UPDATE users SET password = :password, modified = :modified WHERE id = :id`
	updateUserLastLoginSQL     = `UPDATE users SET last_login = :last_login, modified = :modified WHERE id = :id`
	updateUserEmailVerifiedSQL = `UPDATE users SET email_verified = :email_verified, modified = :modified WHERE id = :id`
	deleteUserRoleSQL          = `DELETE FROM user_roles WHERE user_id = :user_id AND role_id = :role_id`
	deleteUserRolesByUserSQL   = `DELETE FROM user_roles WHERE user_id = :user_id`
)

//===========================================================================
// Store Methods
//===========================================================================

func (s *Store) ListUsers(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.User], error) {
	return list(s, ctx, users, filter)
}

func (s *Store) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	var created *models.User
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateUser(user)
		return err
	})
	return created, err
}

func (s *Store) RetrieveUser(ctx context.Context, id ulid.ULID) (*models.User, error) {
	var user *models.User
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		user, err = t.RetrieveUser(id)
		return err
	})
	return user, err
}

func (s *Store) RetrieveUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user *models.User
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		user, err = t.RetrieveUserByEmail(email)
		return err
	})
	return user, err
}

func (s *Store) UpdateUser(ctx context.Context, user *models.User) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateUser(user)
	})
}

func (s *Store) UpdatePassword(ctx context.Context, userID ulid.ULID, password string) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdatePassword(userID, password)
	})
}

func (s *Store) UpdateLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateLastLogin(userID, lastLogin)
	})
}

func (s *Store) VerifyEmail(ctx context.Context, userID ulid.ULID) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.VerifyEmail(userID)
	})
}

func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.DeleteUser(userID)
	})
}

func (s *Store) AddRoleToUser(ctx context.Context, userID ulid.ULID, roleID int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.AddRoleToUser(userID, roleID)
	})
}

func (s *Store) AddRoleToUserByTitle(ctx context.Context, userID ulid.ULID, title string) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.AddRoleToUserByTitle(userID, title)
	})
}

func (s *Store) RemoveRoleFromUser(ctx context.Context, userID ulid.ULID, roleID int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.RemoveRoleFromUser(userID, roleID)
	})
}

func (s *Store) RemoveRoleFromUserByTitle(ctx context.Context, userID ulid.ULID, title string) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.RemoveRoleFromUserByTitle(userID, title)
	})
}

func (s *Store) ReplaceUserRoles(ctx context.Context, userID ulid.ULID, roleIDs []int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.ReplaceUserRoles(userID, roleIDs)
	})
}

//===========================================================================
// Tx Methods
//===========================================================================

// ListUsers returns a cursor over users matching filter. [tidal.Cursor.Close] rolls back
// the transaction; use [tidal.Cursor.CloseRows] to release the result set and continue
// using this transaction.
func (t *tx) ListUsers(filter tidal.ListFilter) (tidal.Cursor[*models.User], error) {
	return listInTx(t, users, filter)
}

func (t *tx) CreateUser(user *models.User) (*models.User, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if !user.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	if _, err := users.Create(t.tx, user); err != nil {
		return nil, tidalErr(err)
	}

	rolesToAssign := user.Roles
	if len(rolesToAssign) == 0 {
		var err error
		rolesToAssign, err = t.defaultRoles()
		if err != nil {
			return nil, err
		}
	}

	for i := range rolesToAssign {
		roleID, err := t.resolveRoleID(&rolesToAssign[i])
		if err != nil {
			return nil, err
		}
		if err := t.addRoleToUser(user.ID, roleID); err != nil {
			return nil, err
		}
	}

	return t.retrieveUser(user.ID)
}

func (t *tx) RetrieveUser(id ulid.ULID) (*models.User, error) {
	return t.retrieveUser(id)
}

func (t *tx) RetrieveUserByEmail(email string) (*models.User, error) {
	found, err := retrieveBy(t, users, "email", email)
	if err != nil {
		return nil, err
	}
	return t.retrieveUser(found.ID)
}

func (t *tx) UpdateUser(user *models.User) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return tidalErr(users.Update(t.tx, user))
}

func (t *tx) UpdatePassword(userID ulid.ULID, password string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	result, err := t.tx.Exec(
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
}

func (t *tx) UpdateLastLogin(userID ulid.ULID, lastLogin time.Time) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	result, err := t.tx.Exec(
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
}

func (t *tx) VerifyEmail(userID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	result, err := t.tx.Exec(
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
}

func (t *tx) DeleteUser(userID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if userID.IsZero() {
		return qerrors.ErrMissingID
	}
	result, err := users.Delete(t.tx, sql.Named("id", userID))
	if err != nil {
		return tidalErr(err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return qerrors.ErrNotFound
	}
	return nil
}

func (t *tx) AddRoleToUser(userID ulid.ULID, roleID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.addRoleToUser(userID, roleID)
}

func (t *tx) AddRoleToUserByTitle(userID ulid.ULID, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	role, err := t.retrieveRoleByTitle(title)
	if err != nil {
		return err
	}
	return t.addRoleToUser(userID, role.ID)
}

func (t *tx) RemoveRoleFromUser(userID ulid.ULID, roleID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	_, err := t.tx.Exec(
		deleteUserRoleSQL,
		sql.Named("user_id", userID),
		sql.Named("role_id", roleID),
	)
	return tidalErr(err)
}

func (t *tx) RemoveRoleFromUserByTitle(userID ulid.ULID, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	role, err := t.retrieveRoleByTitle(title)
	if err != nil {
		return err
	}
	_, err = t.tx.Exec(
		deleteUserRoleSQL,
		sql.Named("user_id", userID),
		sql.Named("role_id", role.ID),
	)
	return tidalErr(err)
}

func (t *tx) ReplaceUserRoles(userID ulid.ULID, roleIDs []int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if _, err := t.tx.Exec(deleteUserRolesByUserSQL, sql.Named("user_id", userID)); err != nil {
		return tidalErr(err)
	}
	for _, roleID := range roleIDs {
		if err := t.addRoleToUser(userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

//===========================================================================
// Helpers
//===========================================================================

func (t *tx) retrieveUser(id ulid.ULID) (*models.User, error) {
	user, err := users.Retrieve(t.tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}

	roles, err := t.userRoles(id)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	perms, err := t.userPermissions(id)
	if err != nil {
		return nil, err
	}
	user.Permissions = perms
	return user, nil
}

func (t *tx) userRoles(userID ulid.ULID) ([]models.Role, error) {
	rows, err := t.tx.Query(userRolesSQL, sql.Named("user_id", userID))
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

func (t *tx) userPermissions(userID ulid.ULID) ([]models.Permission, error) {
	rows, err := t.tx.Query(userPermissionsSQL, sql.Named("user_id", userID))
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

func (t *tx) defaultRoles() ([]models.Role, error) {
	rows, err := t.tx.Query(defaultRolesSQL)
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

func (t *tx) addRoleToUser(userID ulid.ULID, roleID int64) error {
	junction := &models.UserRole{UserID: userID, RoleID: roleID}
	_, err := userRoles.Create(t.tx, junction)
	return tidalErr(err)
}
