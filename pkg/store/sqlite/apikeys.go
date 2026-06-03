package sqlite

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/cursor"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// APIKey Store
//===========================================================================

var apikeyCRUD = MakeCRUD[*models.APIKey]("api_keys")

func (s *Store) ListAPIKeys(ctx context.Context, filter cursor.Filter) (out cursor.Cursor[*models.APIKey], err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}

	if out, err = tx.ListAPIKeys(filter); err != nil {
		return nil, err
	}
	return out, nil
}

func (tx *Tx) ListAPIKeys(filter cursor.Filter) (out cursor.Cursor[*models.APIKey], err error) {
	return apikeyCRUD.List(tx, filter)
}

func (s *Store) CreateAPIKey(ctx context.Context, key *models.APIKey) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateAPIKey(key); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) CreateAPIKey(key *models.APIKey) (err error) {
	if _, err = apikeyCRUD.Create(tx, key); err != nil {
		return err
	}

	for _, permission := range key.Permissions {
		if err = tx.AddPermissionToAPIKey(key.ID, permission.Title); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) RetrieveAPIKey(ctx context.Context, id any) (key *models.APIKey, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if key, err = tx.RetrieveAPIKey(id); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return key, nil
}

func (tx *Tx) RetrieveAPIKey(id any) (key *models.APIKey, err error) {

	var param sql.NamedArg
	switch t := id.(type) {
	case string:
		if t == "" {
			return nil, errors.ErrMissingID
		}
		param = sql.Named("clientID", t)
	case ulid.ULID:
		if t.IsZero() {
			return nil, errors.ErrMissingID
		}
		param = sql.Named("id", t)
	default:
		return nil, errors.Fmt("invalid type %T for API key ID", id)
	}

	if key, err = apikeyCRUD.Retrieve(tx, param); err != nil {
		return nil, err
	}

	// Fetch api key permissions
	var permissions []string
	if permissions, err = tx.apikeyPermissions(key.ID); err != nil {
		return nil, err
	}
	key.Permissions.Load(permissions)

	return key, nil
}

const (
	apikeyPermissionsSQL = "SELECT p.title FROM api_key_permissions akp JOIN permissions p ON akp.permission_id = p.id WHERE akp.api_key_id=:keyID ORDER BY p.title"
)

func (tx *Tx) apikeyPermissions(keyID ulid.ULID) (permissions []string, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(apikeyPermissionsSQL, sql.Named("keyID", keyID)); err != nil {
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

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return permissions, nil
}

func (s *Store) UpdateAPIKey(ctx context.Context, key *models.APIKey) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateAPIKey(key); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdateAPIKey(key *models.APIKey) (err error) {
	return apikeyCRUD.Update(tx, key)
}

const (
	updateLastSeenSQL = "UPDATE api_keys SET last_seen=:lastSeen, modified=:modified WHERE id=:id"
)

func (s *Store) UpdateLastSeen(ctx context.Context, keyID ulid.ULID, lastSeen time.Time) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateLastSeen(keyID, lastSeen); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) UpdateLastSeen(keyID ulid.ULID, lastSeen time.Time) (err error) {
	if keyID.IsZero() {
		return errors.ErrMissingID
	}

	params := []sql.NamedArg{
		sql.Named("id", keyID),
		sql.Named("lastSeen", sql.NullTime{Time: lastSeen, Valid: !lastSeen.IsZero()}),
		sql.Named("modified", time.Now()),
	}

	var result sql.Result
	if result, err = tx.Exec(updateLastSeenSQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

const (
	addPermissionToKeySQL = "INSERT INTO api_key_permissions (api_key_id, permission_id, created) VALUES (:keyID, :permissionID, :created)"
)

func (s *Store) AddPermissionToAPIKey(ctx context.Context, keyID ulid.ULID, permission any) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.AddPermissionToAPIKey(keyID, permission); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) AddPermissionToAPIKey(keyID ulid.ULID, permission any) (err error) {
	if keyID.IsZero() {
		return errors.ErrMissingID
	}

	// Retrieve the permission to ensure we have a valid ID.
	var resolvedPermission *models.Permission
	if resolvedPermission, err = tx.RetrievePermission(permission); err != nil {
		return err
	}

	params := []sql.NamedArg{
		sql.Named("keyID", keyID),
		sql.Named("permissionID", resolvedPermission.ID),
		sql.Named("created", time.Now()),
	}

	if _, err = tx.Exec(addPermissionToKeySQL, params...); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	removePermissionFromKeySQL = "DELETE FROM api_key_permissions WHERE api_key_id=:keyID AND permission_id=:permissionID"
)

func (s *Store) RemovePermissionFromAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.RemovePermissionFromAPIKey(keyID, permissionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) RemovePermissionFromAPIKey(keyID ulid.ULID, permissionID int64) (err error) {
	if keyID.IsZero() || permissionID == 0 {
		return errors.ErrMissingID
	}

	params := []sql.NamedArg{
		sql.Named("keyID", keyID),
		sql.Named("permissionID", permissionID),
	}

	if _, err = tx.Exec(removePermissionFromKeySQL, params...); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	revokeAPIKeySQL = "UPDATE api_keys SET revoked=:revoked, modified=:modified WHERE id=:id"
)

func (s *Store) RevokeAPIKey(ctx context.Context, keyID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.RevokeAPIKey(keyID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) RevokeAPIKey(keyID ulid.ULID) (err error) {
	if keyID.IsZero() {
		return errors.ErrMissingID
	}

	now := time.Now()
	params := []sql.NamedArg{
		sql.Named("id", keyID),
		sql.Named("revoked", sql.NullTime{Time: now, Valid: true}),
		sql.Named("modified", now),
	}

	var result sql.Result
	if result, err = tx.Exec(revokeAPIKeySQL, params...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteAPIKey(keyID); err != nil {
		return err
	}

	return tx.Commit()
}

func (tx *Tx) DeleteAPIKey(keyID ulid.ULID) (err error) {
	_, err = apikeyCRUD.Delete(tx, sql.Named("id", keyID))
	return err
}
