package backend

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

var veroTokens = tidal.New[*models.VeroToken]("vero_tokens")

const retrieveVeroByResourceSQL = `SELECT id, token_type, resource_id, email, expiration, signature, sent_on, created, modified FROM vero_tokens WHERE resource_id = :resource_id AND token_type = :token_type`

//===========================================================================
// Store Methods
//===========================================================================

func (s *Store) CreateVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	var created *models.VeroToken
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateVeroToken(token)
		return err
	})
	return created, err
}

func (s *Store) CreateResetPasswordVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	var created *models.VeroToken
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateResetPasswordVeroToken(token)
		return err
	})
	return created, err
}

func (s *Store) CreateTeamInviteVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	var created *models.VeroToken
	err := s.WithTx(ctx, nil, func(t txn.Tx) (err error) {
		created, err = t.CreateTeamInviteVeroToken(token)
		return err
	})
	return created, err
}

func (s *Store) RetrieveVeroToken(ctx context.Context, id ulid.ULID) (*models.VeroToken, error) {
	var token *models.VeroToken
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		token, err = t.RetrieveVeroToken(id)
		return err
	})
	return token, err
}

func (s *Store) RetrieveVeroTokenByResource(ctx context.Context, resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	var token *models.VeroToken
	err := s.WithReadTx(ctx, func(t txn.Tx) (err error) {
		token, err = t.RetrieveVeroTokenByResource(resourceID, tokenType)
		return err
	})
	return token, err
}

func (s *Store) UpdateVeroToken(ctx context.Context, token *models.VeroToken) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.UpdateVeroToken(token)
	})
}

func (s *Store) DeleteVeroToken(ctx context.Context, id ulid.ULID) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.DeleteVeroToken(id)
	})
}

func (s *Store) CompletePasswordReset(ctx context.Context, veroTokenID ulid.ULID, newPassword string) error {
	return s.WithTx(ctx, nil, func(t txn.Tx) error {
		return t.CompletePasswordReset(veroTokenID, newPassword)
	})
}

//===========================================================================
// Tx Methods
//===========================================================================

func (t *tx) CreateVeroToken(token *models.VeroToken) (*models.VeroToken, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if !token.ID.IsZero() {
		return nil, errors.ErrNoIDOnCreate
	}

	if _, err := veroTokens.Create(t.tx, token); err != nil {
		return nil, tidalErr(err)
	}

	return t.retrieveVeroToken(token.ID)
}

func (t *tx) CreateResetPasswordVeroToken(token *models.VeroToken) (*models.VeroToken, error) {
	return t.createResourceVeroToken(token, enum.TokenTypeResetPassword)
}

func (t *tx) CreateTeamInviteVeroToken(token *models.VeroToken) (*models.VeroToken, error) {
	return t.createResourceVeroToken(token, enum.TokenTypeTeamInvite)
}

func (t *tx) RetrieveVeroToken(id ulid.ULID) (*models.VeroToken, error) {
	return t.retrieveVeroToken(id)
}

func (t *tx) RetrieveVeroTokenByResource(resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	return t.findVeroToken(resourceID, tokenType)
}

func (t *tx) UpdateVeroToken(token *models.VeroToken) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return tidalErr(veroTokens.Update(t.tx, token))
}

func (t *tx) DeleteVeroToken(id ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	if id.IsZero() {
		return errors.ErrMissingID
	}
	result, err := veroTokens.Delete(t.tx, sql.Named("id", id))
	if err != nil {
		return tidalErr(err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (t *tx) CompletePasswordReset(veroTokenID ulid.ULID, newPassword string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}

	token, err := t.retrieveVeroToken(veroTokenID)
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

	result, err := t.tx.Exec(
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

	return t.deleteVeroToken(token.ID)
}

//===========================================================================
// Helpers
//===========================================================================

func (t *tx) createResourceVeroToken(token *models.VeroToken, tokenType enum.TokenType) (*models.VeroToken, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	if !token.ID.IsZero() {
		return nil, errors.ErrNoIDOnCreate
	}
	if token.TokenType != tokenType {
		return nil, errors.ErrTypeMismatch
	}
	if !token.ResourceID.Valid || token.ResourceID.ULID.IsZero() {
		return nil, errors.ErrMissingReference
	}

	existing, err := t.findVeroToken(token.ResourceID.ULID, tokenType)
	if err != nil && !errors.Is(err, errors.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		if !existing.IsExpired() {
			return nil, errors.ErrTooSoon
		}
		if err = t.deleteVeroToken(existing.ID); err != nil {
			return nil, err
		}
	}

	if _, err = veroTokens.Create(t.tx, token); err != nil {
		return nil, tidalErr(err)
	}

	return t.retrieveVeroToken(token.ID)
}

func (t *tx) retrieveVeroToken(id ulid.ULID) (*models.VeroToken, error) {
	token, err := veroTokens.Retrieve(t.tx, sql.Named("id", id))
	if err != nil {
		return nil, tidalErr(err)
	}
	return token, nil
}

func (t *tx) findVeroToken(resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	token := &models.VeroToken{}
	err := token.Scan(tidal.Retrieve, t.tx.QueryRow(
		retrieveVeroByResourceSQL,
		sql.Named("resource_id", ulid.NullULID{Valid: true, ULID: resourceID}),
		sql.Named("token_type", tokenType),
	))
	if err != nil {
		return nil, tidalErr(err)
	}
	return token, nil
}

func (t *tx) deleteVeroToken(id ulid.ULID) error {
	_, err := veroTokens.Delete(t.tx, sql.Named("id", id))
	return tidalErr(err)
}
