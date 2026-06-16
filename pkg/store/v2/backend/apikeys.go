package backend

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var apiKeys = tidal.New[*models.APIKey]("api_keys")
var apiKeyPermissions = tidal.New[*models.APIKeyPermission]("api_key_permissions")

const (
	apiKeyPermissionsSQL            = `SELECT p.id, p.title, p.description, p.created, p.modified FROM api_key_permissions akp JOIN permissions p ON p.id = akp.permission_id WHERE akp.api_key_id = :api_key_id ORDER BY p.title`
	updateAPIKeyLastSeenSQL         = `UPDATE api_keys SET last_seen = :last_seen, modified = :modified WHERE id = :id`
	deleteAPIKeyPermissionSQL       = `DELETE FROM api_key_permissions WHERE api_key_id = :api_key_id AND permission_id = :permission_id`
	deleteAPIKeyPermissionsByKeySQL = `DELETE FROM api_key_permissions WHERE api_key_id = :api_key_id`
	revokeAPIKeySQL                 = `UPDATE api_keys SET revoked = :revoked, modified = :modified WHERE id = :id`
)

//===========================================================================
// Store Methods
//===========================================================================

// NOTE: when using [tidal.Clause] as filter, the user is responsible for
// filtering out revoked keys, because this function does not modify a
// [tidal.Clause] to do so. If a [tidal.Filter] is used, then this function will
// automatically filter out revoked keys.
func (s *Store) ListAPIKeys(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error) {
	return list(s, ctx, apiKeys, apiKeyListFilter(filter))
}

func (s *Store) CreateAPIKey(ctx context.Context, key *models.APIKey) (*models.APIKey, error) {
	var created *models.APIKey
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateAPIKey(key)
		return err
	})
	return created, err
}

func (s *Store) CreateAPIKeyFor(ctx context.Context, key *models.APIKey, creator ulid.ULID) (*models.APIKey, error) {
	var created *models.APIKey
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateAPIKeyFor(key, creator)
		return err
	})
	return created, err
}

func (s *Store) RetrieveAPIKey(ctx context.Context, id ulid.ULID) (*models.APIKey, error) {
	var key *models.APIKey
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		key, err = t.RetrieveAPIKey(id)
		return err
	})
	return key, err
}

func (s *Store) RetrieveAPIKeyByClientID(ctx context.Context, clientID string) (*models.APIKey, error) {
	var key *models.APIKey
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		key, err = t.RetrieveAPIKeyByClientID(clientID)
		return err
	})
	return key, err
}

func (s *Store) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateAPIKey(key)
	})
}

func (s *Store) UpdateLastSeen(ctx context.Context, keyID ulid.ULID, lastSeen time.Time) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateLastSeen(keyID, lastSeen)
	})
}

func (s *Store) AddPermissionToAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.AddPermissionToAPIKey(keyID, permissionID)
	})
}

func (s *Store) AddPermissionToAPIKeyByTitle(ctx context.Context, keyID ulid.ULID, title string) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.AddPermissionToAPIKeyByTitle(keyID, title)
	})
}

func (s *Store) RemovePermissionFromAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.RemovePermissionFromAPIKey(keyID, permissionID)
	})
}

func (s *Store) ReplaceAPIKeyPermissions(ctx context.Context, keyID ulid.ULID, permissionIDs []int64) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.ReplaceAPIKeyPermissions(keyID, permissionIDs)
	})
}

func (s *Store) RevokeAPIKey(ctx context.Context, keyID ulid.ULID) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.RevokeAPIKey(keyID)
	})
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.DeleteAPIKey(keyID)
	})
}

//===========================================================================
// Tx Methods
//===========================================================================

// ListAPIKeys returns a cursor over API keys matching filter. [tidal.Cursor.Close] rolls
// back the transaction; use [tidal.Cursor.CloseRows] to release the result set and
// continue using this transaction.
func (t *tx) ListAPIKeys(filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error) {
	return listInTx(t, apiKeys, apiKeyListFilter(filter))
}

func (t *tx) CreateAPIKey(key *models.APIKey) (*models.APIKey, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if !key.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}
	return t.createAPIKey(key)
}

func (t *tx) CreateAPIKeyFor(key *models.APIKey, creator ulid.ULID) (*models.APIKey, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if !key.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	parent, err := t.retrieveAPIKey(creator)
	if err != nil {
		return nil, err
	}
	if parent.Status() == enum.APIKeyStatusRevoked {
		return nil, qerrors.ErrNotAuthorized
	}

	if key.CreatedBy.IsZero() {
		key.CreatedBy = parent.CreatedBy
	}

	return t.createAPIKey(key)
}

func (t *tx) RetrieveAPIKey(id ulid.ULID) (*models.APIKey, error) {
	return t.retrieveAPIKey(id)
}

func (t *tx) RetrieveAPIKeyByClientID(clientID string) (*models.APIKey, error) {
	found, err := retrieveBy(t, apiKeys, "client_id", clientID)
	if err != nil {
		return nil, err
	}
	return t.retrieveAPIKey(found.ID)
}

func (t *tx) UpdateAPIKey(key *models.APIKey) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return tidalErr(apiKeys.Update(t.tx, key))
}

func (t *tx) UpdateLastSeen(keyID ulid.ULID, lastSeen time.Time) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	result, err := t.tx.Exec(
		updateAPIKeyLastSeenSQL,
		sql.Named("id", keyID),
		sql.Named("last_seen", sql.NullTime{Time: lastSeen, Valid: !lastSeen.IsZero()}),
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

func (t *tx) AddPermissionToAPIKey(keyID ulid.ULID, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.addPermissionToAPIKey(keyID, permissionID)
}

func (t *tx) AddPermissionToAPIKeyByTitle(keyID ulid.ULID, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.addPermissionToAPIKeyByTitle(keyID, title)
}

func (t *tx) RemovePermissionFromAPIKey(keyID ulid.ULID, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	_, err := t.tx.Exec(
		deleteAPIKeyPermissionSQL,
		sql.Named("api_key_id", keyID),
		sql.Named("permission_id", permissionID),
	)
	return tidalErr(err)
}

func (t *tx) ReplaceAPIKeyPermissions(keyID ulid.ULID, permissionIDs []int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if _, err := t.tx.Exec(deleteAPIKeyPermissionsByKeySQL, sql.Named("api_key_id", keyID)); err != nil {
		return tidalErr(err)
	}
	for _, permissionID := range permissionIDs {
		if err := t.addPermissionToAPIKey(keyID, permissionID); err != nil {
			return err
		}
	}
	return nil
}

func (t *tx) RevokeAPIKey(keyID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := t.tx.Exec(
		revokeAPIKeySQL,
		sql.Named("id", keyID),
		sql.Named("revoked", sql.NullTime{Time: now, Valid: true}),
		sql.Named("modified", now),
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

func (t *tx) DeleteAPIKey(keyID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if keyID.IsZero() {
		return qerrors.ErrMissingID
	}
	result, err := apiKeys.Delete(t.tx, sql.Named("id", keyID))
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

func (t *tx) createAPIKey(key *models.APIKey) (*models.APIKey, error) {
	if _, err := apiKeys.Create(t.tx, key); err != nil {
		return nil, tidalErr(err)
	}

	if err := t.addPermissionsToAPIKey(key.ID, key.Permissions); err != nil {
		return nil, err
	}

	return t.retrieveAPIKey(key.ID)
}

func (t *tx) retrieveAPIKey(id ulid.ULID) (*models.APIKey, error) {
	key, err := apiKeys.Retrieve(t.tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}

	perms, err := t.apiKeyPermissions(id)
	if err != nil {
		return nil, err
	}
	key.Permissions = perms
	return key, nil
}

func (t *tx) apiKeyPermissions(keyID ulid.ULID) ([]models.Permission, error) {
	rows, err := t.tx.Query(apiKeyPermissionsSQL, sql.Named("api_key_id", keyID))
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

func (t *tx) addPermissionsToAPIKey(keyID ulid.ULID, permissions []models.Permission) error {
	for _, permission := range permissions {
		permID := permission.ID
		if permID == 0 && permission.Title != "" {
			resolved, err := t.retrievePermissionByTitle(permission.Title)
			if err != nil {
				return err
			}
			permID = resolved.ID
		}
		if permID == 0 {
			continue
		}
		if err := t.addPermissionToAPIKey(keyID, permID); err != nil {
			return err
		}
	}
	return nil
}

func (t *tx) addPermissionToAPIKey(keyID ulid.ULID, permissionID int64) error {
	junction := &models.APIKeyPermission{APIKeyID: keyID, PermissionID: permissionID}
	_, err := apiKeyPermissions.Create(t.tx, junction)
	return tidalErr(err)
}

func (t *tx) addPermissionToAPIKeyByTitle(keyID ulid.ULID, title string) error {
	permission, err := t.retrievePermissionByTitle(title)
	if err != nil {
		return err
	}
	return t.addPermissionToAPIKey(keyID, permission.ID)
}

func apiKeyListFilter(filter tidal.ListFilter) tidal.ListFilter {
	switch f := filter.(type) {
	case *tidal.Clause:
		// We don't want to mess up the user's SQL, so we'll just pass it
		// through; they may get revoked keys back in this case, so it's up
		// to them to handle that.
		return filter
	case *tidal.Filter:
		return f.And("revoked", tidal.IsNull, nil)
	default:
		return (&tidal.Filter{}).Where("revoked", tidal.IsNull, nil)
	}
}
