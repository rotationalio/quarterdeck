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
// APIKey Store
//===========================================================================

const (
	listAPIKeysSQL = "SELECT id, description, client_id, created_by, last_seen, revoked, created, modified FROM api_keys WHERE revoked IS NULL ORDER BY created DESC"
)

func (s *Store) ListAPIKeys(ctx context.Context, page *models.Page) (out *models.APIKeyList, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListAPIKeys(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (tx *Tx) ListAPIKeys(page *models.Page) (out *models.APIKeyList, err error) {
	// TODO: handle pagination
	out = &models.APIKeyList{
		APIKeys: make([]*models.APIKey, 0),
		Page:    models.PageFrom(page),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listAPIKeysSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		key := &models.APIKey{}
		if err = key.ScanSummary(rows); err != nil {
			return nil, err
		}
		out.APIKeys = append(out.APIKeys, key)
	}

	if err = rows.Err(); err != nil {
		return nil, dbe(err)
	}

	return out, nil
}

const (
	createAPIKeySQL = "INSERT INTO api_keys (id, description, client_id, secret, created_by, last_seen, revoked, created, modified) VALUES (:id, :description, :clientID, :secret, :createdBy, :lastSeen, :revoked, :created, :modified)"
)

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
	if !key.ID.IsZero() {
		return errors.ErrNoIDOnCreate
	}

	if key.ClientID == "" || key.Secret == "" || key.CreatedBy.IsZero() {
		return errors.ErrZeroValuedNotNull
	}

	key.ID = ulid.MakeSecure()
	key.Created = time.Now()
	key.Modified = key.Created

	if _, err = tx.Exec(createAPIKeySQL, key.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	retrieveAPIKeyByClientIDSQL = "SELECT * FROM api_keys WHERE client_id=:clientID"
	retrieveAPIKeyByIDSQL       = "SELECT * FROM api_keys WHERE id=:id"
)

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
	var (
		query string
		param sql.NamedArg
	)

	switch t := id.(type) {
	case string:
		if t == "" {
			return nil, errors.ErrMissingID
		}

		query = retrieveAPIKeyByClientIDSQL
		param = sql.Named("clientID", t)
	case ulid.ULID:
		if t.IsZero() {
			return nil, errors.ErrMissingID
		}

		query = retrieveAPIKeyByIDSQL
		param = sql.Named("id", t)
	}

	key = &models.APIKey{}
	if err = key.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	return key, nil
}

const (
	updateAPIKeySQL = "UPDATE api_keys SET description=:description, modified=:modified WHERE id=:id"
)

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
	if key.ID.IsZero() {
		return errors.ErrMissingID
	}

	key.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateAPIKeySQL, key.Params()...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
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

	params := []any{
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

	params := []any{
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

	params := []any{
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
	params := []any{
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

const (
	deleteAPIKeySQL = "DELETE FROM api_keys WHERE id=:id"
)

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
	if keyID.IsZero() {
		return errors.ErrMissingID
	}

	var result sql.Result
	if result, err = tx.Exec(deleteAPIKeySQL, sql.Named("id", keyID)); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}
