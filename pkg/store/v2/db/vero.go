package db

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

// CRUD operations for VeroTokens.
var veroTokens = tidal.New[*models.VeroToken]("vero_tokens")

// SQL query to retrieve a VeroToken by :resource_id and :token_type.
const retrieveVeroByResourceSQL = `SELECT id, token_type, resource_id, email, expiration, signature, sent_on, created, modified FROM vero_tokens WHERE resource_id = :resource_id AND token_type = :token_type`

func (d *DB) CreateVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	if !token.ID.IsZero() {
		return nil, errors.ErrNoIDOnCreate
	}

	var created *models.VeroToken
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		if _, err := veroTokens.Create(tx, token); err != nil {
			return tidalErr(err)
		}
		var err error
		created, err = d.retrieveVeroTokenTx(tx, token.ID)
		return err
	})
	return created, err
}

func (d *DB) RetrieveVeroToken(ctx context.Context, id ulid.ULID) (*models.VeroToken, error) {
	var token *models.VeroToken
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		token, err = d.retrieveVeroTokenTx(tx, id)
		return err
	})
	return token, err
}

func (d *DB) RetrieveVeroTokenByResource(ctx context.Context, resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	var token *models.VeroToken
	err := d.withReadTx(ctx, func(tx tidal.Tx) (err error) {
		token, err = d.findVeroTokenTx(tx, resourceID, tokenType)
		return err
	})
	return token, err
}

func (d *DB) UpdateVeroToken(ctx context.Context, token *models.VeroToken) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		return tidalErr(veroTokens.Update(tx, token))
	})
}

func (d *DB) DeleteVeroToken(ctx context.Context, id ulid.ULID) error {
	if id.IsZero() {
		return errors.ErrMissingID
	}
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		result, err := veroTokens.Delete(tx, sql.Named("id", id))
		if err != nil {
			return tidalErr(err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return errors.ErrNotFound
		}
		return nil
	})
}

func (d *DB) CreateResetPasswordVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	return d.createResourceVeroToken(ctx, token, enum.TokenTypeResetPassword)
}

func (d *DB) CreateTeamInviteVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	return d.createResourceVeroToken(ctx, token, enum.TokenTypeTeamInvite)
}

// Creates a VeroToken for a resource with the given token type, ensuring that
// there is at most one unexpired token for the resource.
func (d *DB) createResourceVeroToken(ctx context.Context, token *models.VeroToken, tokenType enum.TokenType) (*models.VeroToken, error) {
	if !token.ID.IsZero() {
		return nil, errors.ErrNoIDOnCreate
	}
	if token.TokenType != tokenType {
		return nil, errors.ErrTypeMismatch
	}
	if !token.ResourceID.Valid || token.ResourceID.ULID.IsZero() {
		return nil, errors.ErrMissingReference
	}

	var created *models.VeroToken
	err := d.withTx(ctx, nil, func(tx tidal.Tx) error {
		existing, err := d.findVeroTokenTx(tx, token.ResourceID.ULID, tokenType)
		if err != nil && !errors.Is(err, errors.ErrNotFound) {
			return err
		}
		if existing != nil {
			if !existing.IsExpired() {
				return errors.ErrTooSoon
			}
			if err = d.deleteVeroTokenTx(tx, existing.ID); err != nil {
				return err
			}
		}

		if _, err = veroTokens.Create(tx, token); err != nil {
			return tidalErr(err)
		}
		created, err = d.retrieveVeroTokenTx(tx, token.ID)
		return err
	})
	return created, err
}

func (d *DB) retrieveVeroTokenTx(tx tidal.Tx, id ulid.ULID) (*models.VeroToken, error) {
	token, err := veroTokens.Retrieve(tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}
	return token, nil
}

func (d *DB) findVeroTokenTx(tx tidal.Tx, resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	token := &models.VeroToken{}
	err := token.Scan(tidal.Retrieve, tx.QueryRow(
		retrieveVeroByResourceSQL,
		sql.Named("resource_id", ulid.NullULID{Valid: true, ULID: resourceID}),
		sql.Named("token_type", tokenType),
	))
	if err != nil {
		return nil, tidalErr(err)
	}
	return token, nil
}

func (d *DB) deleteVeroTokenTx(tx tidal.Tx, id ulid.ULID) error {
	_, err := veroTokens.Delete(tx, sql.Named("id", id))
	return tidalErr(err)
}

func (d *DB) CompletePasswordReset(ctx context.Context, veroTokenID ulid.ULID, newPassword string) error {
	return d.withTx(ctx, nil, func(tx tidal.Tx) error {
		token, err := d.retrieveVeroTokenTx(tx, veroTokenID)
		if err != nil {
			return err
		}

		if token.TokenType != enum.TokenTypeResetPassword {
			return errors.ErrTypeMismatch
		}
		if token.IsExpired() {
			return errors.ErrExpiredToken
		}
		if !token.ResourceID.Valid || token.ResourceID.ULID.IsZero() {
			return errors.ErrMissingReference
		}

		result, err := tx.Exec(
			updateUserPasswordSQL,
			sql.Named("id", token.ResourceID.ULID),
			sql.Named("password", newPassword),
			sql.Named("modified", time.Now().UTC()),
		)
		if err != nil {
			return tidalErr(err)
		}
		n, _ := result.RowsAffected()
		if n == 0 {
			return errors.ErrNotFound
		}

		return d.deleteVeroTokenTx(tx, token.ID)
	})
}
