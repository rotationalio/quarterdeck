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
// Users Store
//===========================================================================

const (
	listUsersSQL   = "SELECT id, name, email, last_login, created, modified FROM users ORDER BY created DESC"
	filterUsersSQL = "SELECT u.id, u.name, u.email, u.last_login, u.created, u.modified FROM users u JOIN user_roles ur ON u.id=ur.user_id JOIN roles r ON ur.role_id=r.id WHERE r.title=:role COLLATE NOCASE ORDER BY u.created DESC"
)

func (s *Store) ListUsers(ctx context.Context, page *models.UserPage) (out *models.UserList, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListUsers(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (tx *Tx) ListUsers(page *models.UserPage) (out *models.UserList, err error) {
	// TODO: handle pagination
	out = &models.UserList{
		Users: make([]*models.User, 0),
		Page:  models.UserPageFrom(page),
	}

	var rows *sql.Rows
	if page != nil && page.Role != "" {
		if rows, err = tx.Query(filterUsersSQL, sql.Named("role", page.Role)); err != nil {
			return nil, dbe(err)
		}
	} else {
		if rows, err = tx.Query(listUsersSQL); err != nil {
			return nil, dbe(err)
		}
	}
	defer rows.Close()

	for rows.Next() {
		// Scan user summary into a new User struct.
		user := &models.User{}
		if err = user.ScanSummary(rows); err != nil {
			return nil, err
		}
		out.Users = append(out.Users, user)
	}

	return out, nil
}

const (
	defaultRolesSQL   = "SELECT id FROM roles WHERE is_default='t'"
	createUserSQL     = "INSERT INTO users (id, name, email, password, last_login, created, modified) VALUES (:id, :name, :email, :password, :lastLogin, :created, :modified)"
	createUserRoleSQL = "INSERT INTO user_roles (user_id, role_id, created, modified) VALUES (:userID, :roleID, :created, :modified)"
)

func (s *Store) CreateUser(ctx context.Context, user *models.User) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateUser(user); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) CreateUser(user *models.User) (err error) {
	if !user.ID.IsZero() {
		return errors.ErrNoIDOnCreate
	}

	user.ID = ulid.MakeSecure()
	user.Created = time.Now()
	user.Modified = user.Created

	if _, err = tx.Exec(createUserSQL, user.Params()...); err != nil {
		return dbe(err)
	}

	// Add the roles to the user; or if no roles are set, assign the default role(s) if any.
	var roles []*models.Role
	if roles, err = user.Roles(); err != nil {
		if !errors.Is(err, errors.ErrMissingAssociation) {
			return err
		}

		// If no roles are set, assign the default role(s).
		var rows *sql.Rows
		if rows, err = tx.Query(defaultRolesSQL); err != nil {
			return errors.Fmt("could not query default roles: %w", dbe(err))
		}
		defer rows.Close()

		for rows.Next() {
			role := &models.Role{}
			if err = rows.Scan(&role.ID); err != nil {
				return dbe(err)
			}
			roles = append(roles, role)
		}
	}

	for _, role := range roles {
		params := []any{
			sql.Named("userID", user.ID),
			sql.Named("roleID", role.ID),
			sql.Named("created", user.Created),
			sql.Named("modified", user.Modified),
		}
		if _, err = tx.Exec(createUserRoleSQL, params...); err != nil {
			return dbe(err)
		}
	}

	return nil
}

const (
	retrieveUserByIDSQL    = "SELECT * FROM users WHERE id=:id"
	retrieveUserByEmailSQL = "SELECT * FROM users WHERE email=:email"
	userRolesSQL           = "SELECT r.id, r.title, r.description, r.is_default, r.created, r.modified FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id=:userID"
	userPermissionsSQL     = "SELECT permission FROM user_permissions WHERE user_id=:userID"
)

func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (out *models.User, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.RetrieveUser(emailOrUserID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (tx *Tx) RetrieveUser(emailOrUserID any) (out *models.User, err error) {
	var (
		query string
		param sql.NamedArg
	)

	switch t := emailOrUserID.(type) {
	case ulid.ULID:
		query = retrieveUserByIDSQL
		param = sql.Named("id", t)
	case string:
		query = retrieveUserByEmailSQL
		param = sql.Named("email", t)
	default:
		return nil, errors.Fmt("invalid type %T for emailOrUserID", t)
	}

	out = &models.User{}
	if err = out.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	// Fetch user role information.
	var roles []*models.Role
	if roles, err = tx.userRoles(out.ID); err != nil {
		return nil, err
	}
	out.SetRoles(roles)

	// Fetch user permissions.
	var permissions []string
	if permissions, err = tx.userPermissions(out.ID); err != nil {
		return nil, err
	}
	out.SetPermissions(permissions)

	return out, nil
}

func (tx *Tx) userRoles(userID ulid.ULID) (roles []*models.Role, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(userRolesSQL, sql.Named("userID", userID)); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	roles = make([]*models.Role, 0)
	for rows.Next() {
		role := &models.Role{}
		if err = role.Scan(rows); err != nil {
			return nil, dbe(err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func (tx *Tx) userPermissions(userID ulid.ULID) (permissions []string, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(userPermissionsSQL, sql.Named("userID", userID)); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	permissions = make([]string, 0)
	for rows.Next() {
		var permission string
		if err = rows.Scan(&permission); err != nil {
			return nil, dbe(err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

const (
	updateUserSQL = "UPDATE users SET name=:name, email=:email, modified=:modified WHERE id=:id"
)

func (s *Store) UpdateUser(ctx context.Context, user *models.User) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateUser(user); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdateUser(user *models.User) (err error) {
	if user.ID.IsZero() {
		return errors.ErrMissingID
	}

	user.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateUserSQL, user.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (s *Store) UpdatePassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdatePassword(userID, password); err != nil {
		return err
	}

	return tx.Commit()
}

const (
	updatePasswordSQL = "UPDATE users SET password=:password, modified=:modified WHERE id=:id"
)

func (tx *Tx) UpdatePassword(userID ulid.ULID, password string) (err error) {
	if userID.IsZero() {
		return errors.ErrMissingID
	}

	params := []any{
		sql.Named("id", userID),
		sql.Named("password", password),
		sql.Named("modified", time.Now()),
	}

	var result sql.Result
	if result, err = tx.Exec(updatePasswordSQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const (
	updateLastLoginSQL = "UPDATE users SET last_login=:lastLogin, modified=:modified WHERE id=:id"
)

func (s *Store) UpdateLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateLastLogin(userID, lastLogin); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdateLastLogin(userID ulid.ULID, lastLogin time.Time) (err error) {
	if userID.IsZero() {
		return errors.ErrMissingID
	}

	params := []any{
		sql.Named("id", userID),
		sql.Named("lastLogin", sql.NullTime{Time: lastLogin, Valid: !lastLogin.IsZero()}),
		sql.Named("modified", time.Now()),
	}

	var result sql.Result
	if result, err = tx.Exec(updateLastLoginSQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const (
	deleteUserSQL = "DELETE FROM users WHERE id=:id"
)

func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteUser(userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) DeleteUser(userID ulid.ULID) (err error) {
	if userID.IsZero() {
		return errors.ErrMissingID
	}

	var result sql.Result
	if result, err = tx.Exec(deleteUserSQL, sql.Named("id", userID)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}
