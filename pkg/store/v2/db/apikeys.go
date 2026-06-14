package db

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var apiKeys = tidal.New[*models.APIKey]("api_keys")
var apiKeyPermissions = tidal.New[*models.APIKeyPermission]("api_key_permissions")

const (
	apiKeyPermissionsSQL = `SELECT p.id, p.title, p.description, p.created, p.modified
	FROM api_key_permissions akp JOIN permissions p ON p.id = akp.permission_id WHERE akp.api_key_id = :api_key_id ORDER BY p.title`
	updateAPIKeyLastSeenSQL         = `UPDATE api_keys SET last_seen = :last_seen, modified = :modified WHERE id = :id`
	deleteAPIKeyPermissionSQL       = `DELETE FROM api_key_permissions WHERE api_key_id = :api_key_id AND permission_id = :permission_id`
	deleteAPIKeyPermissionsByKeySQL = `DELETE FROM api_key_permissions WHERE api_key_id = :api_key_id`
	revokeAPIKeySQL                 = `UPDATE api_keys SET revoked = :revoked, modified = :modified WHERE id = :id`
)

// NOTE: when using [tidal.Clause] as filter, the user is responsible for
// filtering out revoked keys, because this function does attempt to modify a
// [tidal.Clause] to do so. If a [tidal.Filter] is used, then this function will
// automatically filter out revoked keys.
func (d *DB) ListAPIKeys(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error) {
	switch filter.(type) {
	case *tidal.Clause:
		// We don't want to mess up the user's SQL, so we'll just pass it
		// through; they may get revoked keys back in this case, so it's up
		// to them to handle that.
	case *tidal.Filter:
		// TODO: currently, [tidal.Filter] does not support WHERE clauses,
		// so we have to add it manually via a [tidal.Clause] that we build
		// knowing that the user's [tidal.Filter] only has the ordering,
		// limit, and offset filters. In the future when [tidal.Filter]
		// supports WHERE clauses, we need to add a where clause  to that
		// filter here rather than manually templating a [tidal.Clause].
		filter = &tidal.Clause{SQL: "WHERE revoked IS NULL " + filter.Clause()}
	default:
		filter = &tidal.Clause{SQL: "WHERE revoked IS NULL"}
	}

	return list(d, ctx, apiKeys, filter)
}

func (d *DB) CreateAPIKey(ctx context.Context, key *models.APIKey) (*models.APIKey, error) {
	if !key.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	var created *models.APIKey
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		if _, err := apiKeys.Create(tx, key); err != nil {
			return tidalErr(err)
		}

		if err := d.addPermissionsToAPIKeyTx(tx, key.ID, key.Permissions); err != nil {
			return err
		}

		var err error
		created, err = d.retrieveAPIKeyTx(tx, key.ID)
		return err
	})
	return created, err
}

func (d *DB) RetrieveAPIKey(ctx context.Context, id ulid.ULID) (*models.APIKey, error) {
	var key *models.APIKey
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		key, err = d.retrieveAPIKeyTx(tx, id)
		return err
	})
	return key, err
}

func (d *DB) RetrieveAPIKeyByClientID(ctx context.Context, clientID string) (*models.APIKey, error) {
	var key *models.APIKey
	err := d.withReadTx(ctx, func(tx tidal.Tx) error {
		found, err := retrieveBy(tx, apiKeys, "client_id", clientID)
		if err != nil {
			return err
		}
		key, err = d.retrieveAPIKeyTx(tx, found.ID)
		return err
	})
	return key, err
}

func (d *DB) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return tidalErr(apiKeys.Update(tx, key))
	})
}

func (d *DB) UpdateLastSeen(ctx context.Context, keyID ulid.ULID, lastSeen time.Time) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := tx.Exec(
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
	})
}

func (d *DB) AddPermissionToAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return d.addPermissionToAPIKeyTx(tx, keyID, permissionID)
	})
}

func (d *DB) AddPermissionToAPIKeyByTitle(ctx context.Context, keyID ulid.ULID, title string) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return d.addPermissionToAPIKeyByTitleTx(tx, keyID, title)
	})
}

func (d *DB) RemovePermissionFromAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		_, err := tx.Exec(
			deleteAPIKeyPermissionSQL,
			sql.Named("api_key_id", keyID),
			sql.Named("permission_id", permissionID),
		)
		return tidalErr(err)
	})
}

func (d *DB) ReplaceAPIKeyPermissions(ctx context.Context, keyID ulid.ULID, permissionIDs []int64) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		if _, err := tx.Exec(deleteAPIKeyPermissionsByKeySQL, sql.Named("api_key_id", keyID)); err != nil {
			return tidalErr(err)
		}
		for _, permissionID := range permissionIDs {
			if err := d.addPermissionToAPIKeyTx(tx, keyID, permissionID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *DB) RevokeAPIKey(ctx context.Context, keyID ulid.ULID) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		now := time.Now().UTC()
		result, err := tx.Exec(
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
	})
}

func (d *DB) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error {
	if keyID.IsZero() {
		return qerrors.ErrMissingID
	}
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := apiKeys.Delete(tx, sql.Named("id", keyID))
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

func (d *DB) retrieveAPIKeyTx(tx tidal.Tx, id ulid.ULID) (*models.APIKey, error) {
	key, err := apiKeys.Retrieve(tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}

	perms, err := d.apiKeyPermissionsTx(tx, id)
	if err != nil {
		return nil, err
	}
	key.Permissions = perms
	return key, nil
}

func (d *DB) apiKeyPermissionsTx(tx tidal.Tx, keyID ulid.ULID) ([]models.Permission, error) {
	rows, err := tx.Query(apiKeyPermissionsSQL, sql.Named("api_key_id", keyID))
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

func (d *DB) addPermissionsToAPIKeyTx(tx tidal.Tx, keyID ulid.ULID, permissions []models.Permission) error {
	for _, permission := range permissions {
		permID := permission.ID
		if permID == 0 && permission.Title != "" {
			resolved, err := d.retrievePermissionByTitleTx(tx, permission.Title)
			if err != nil {
				return err
			}
			permID = resolved.ID
		}
		if permID == 0 {
			continue
		}
		if err := d.addPermissionToAPIKeyTx(tx, keyID, permID); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) addPermissionToAPIKeyTx(tx tidal.Tx, keyID ulid.ULID, permissionID int64) error {
	junction := &models.APIKeyPermission{APIKeyID: keyID, PermissionID: permissionID}
	_, err := apiKeyPermissions.Create(tx, junction)
	return tidalErr(err)
}

func (d *DB) addPermissionToAPIKeyByTitleTx(tx tidal.Tx, keyID ulid.ULID, title string) error {
	permission, err := d.retrievePermissionByTitleTx(tx, title)
	if err != nil {
		return err
	}
	return d.addPermissionToAPIKeyTx(tx, keyID, permission.ID)
}

func (d *DB) CreateAPIKeyFor(ctx context.Context, key *models.APIKey, creator ulid.ULID) (*models.APIKey, error) {
	if !key.ID.IsZero() {
		return nil, qerrors.ErrNoIDOnCreate
	}

	var created *models.APIKey
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		creator, err := d.retrieveAPIKeyTx(tx, creator)
		if err != nil {
			return err
		}
		if creator.Status() == enum.APIKeyStatusRevoked {
			return qerrors.ErrNotAuthorized
		}

		if key.CreatedBy.IsZero() {
			key.CreatedBy = creator.CreatedBy
		}

		created, err = d.createAPIKeyTx(tx, key)
		return err
	})
	return created, err
}

func (d *DB) createAPIKeyTx(tx tidal.Tx, key *models.APIKey) (*models.APIKey, error) {
	if _, err := apiKeys.Create(tx, key); err != nil {
		return nil, tidalErr(err)
	}

	if err := d.addPermissionsToAPIKeyTx(tx, key.ID, key.Permissions); err != nil {
		return nil, err
	}

	return d.retrieveAPIKeyTx(tx, key.ID)
}
