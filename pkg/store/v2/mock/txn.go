package mock

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

const (
	BeginTx     = "BeginTx"
	BeginReadTx = "BeginReadTx"
	WithTx      = "WithTx"
	WithReadTx  = "WithReadTx"
)

var (
	_ txn.Tx      = (*Txn)(nil)
	_ txn.StoreTx = (*Store)(nil)
)

func (s *Store) BeginTx(ctx context.Context, opts *sql.TxOptions) (txn.Tx, error) {
	s.calls[BeginTx]++
	if s.readOnly && (opts == nil || !opts.ReadOnly) {
		return nil, errors.ErrReadOnly
	}
	readOnly := opts != nil && opts.ReadOnly
	return &Txn{store: s, ctx: ctx, readOnly: readOnly}, nil
}

func (s *Store) BeginReadTx(ctx context.Context) (txn.Tx, error) {
	s.calls[BeginReadTx]++
	return s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
}

func (s *Store) WithTx(ctx context.Context, opts *sql.TxOptions, fn func(txn.Tx) error) error {
	s.calls[WithTx]++
	t, err := s.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	if err = fn(t); err != nil {
		_ = t.Rollback()
		return err
	}
	return t.Commit()
}

func (s *Store) WithReadTx(ctx context.Context, fn func(txn.Tx) error) error {
	s.calls[WithReadTx]++
	return s.WithTx(ctx, &sql.TxOptions{ReadOnly: true}, fn)
}

// Txn is a mock transaction that delegates to the parent [Store].
type Txn struct {
	store    *Store
	ctx      context.Context
	readOnly bool
}

func (t *Txn) Commit() error   { return nil }
func (t *Txn) Rollback() error { return nil }

func (t *Txn) requireWrite() error {
	if t.readOnly || t.store.readOnly {
		return errors.ErrReadOnly
	}
	return nil
}

//===========================================================================
// UserStore
//===========================================================================

func (t *Txn) CreateUser(user *models.User) (*models.User, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateUser(t.ctx, user)
}

func (t *Txn) ListUsers(filter tidal.ListFilter) (tidal.Cursor[*models.User], error) {
	return t.store.ListUsers(t.ctx, filter)
}

func (t *Txn) RetrieveUser(id ulid.ULID) (*models.User, error) {
	return t.store.RetrieveUser(t.ctx, id)
}

func (t *Txn) RetrieveUserByEmail(email string) (*models.User, error) {
	return t.store.RetrieveUserByEmail(t.ctx, email)
}

func (t *Txn) UpdateUser(user *models.User) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateUser(t.ctx, user)
}

func (t *Txn) UpdatePassword(userID ulid.ULID, password string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdatePassword(t.ctx, userID, password)
}

func (t *Txn) UpdateLastLogin(userID ulid.ULID, lastLogin time.Time) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateLastLogin(t.ctx, userID, lastLogin)
}

func (t *Txn) VerifyEmail(userID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.VerifyEmail(t.ctx, userID)
}

func (t *Txn) DeleteUser(userID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.DeleteUser(t.ctx, userID)
}

func (t *Txn) AddRoleToUser(userID ulid.ULID, roleID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.AddRoleToUser(t.ctx, userID, roleID)
}

func (t *Txn) AddRoleToUserByTitle(userID ulid.ULID, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.AddRoleToUserByTitle(t.ctx, userID, title)
}

func (t *Txn) RemoveRoleFromUser(userID ulid.ULID, roleID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.RemoveRoleFromUser(t.ctx, userID, roleID)
}

func (t *Txn) RemoveRoleFromUserByTitle(userID ulid.ULID, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.RemoveRoleFromUserByTitle(t.ctx, userID, title)
}

func (t *Txn) ReplaceUserRoles(userID ulid.ULID, roleIDs []int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.ReplaceUserRoles(t.ctx, userID, roleIDs)
}

//===========================================================================
// RoleStore
//===========================================================================

func (t *Txn) CreateRole(role *models.Role) (*models.Role, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateRole(t.ctx, role)
}

func (t *Txn) ListRoles(filter tidal.ListFilter) (tidal.Cursor[*models.Role], error) {
	return t.store.ListRoles(t.ctx, filter)
}

func (t *Txn) RetrieveRole(id int64) (*models.Role, error) {
	return t.store.RetrieveRole(t.ctx, id)
}

func (t *Txn) RetrieveRoleByTitle(title string) (*models.Role, error) {
	return t.store.RetrieveRoleByTitle(t.ctx, title)
}

func (t *Txn) UpdateRole(role *models.Role) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateRole(t.ctx, role)
}

func (t *Txn) AddPermissionToRole(roleID int64, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.AddPermissionToRole(t.ctx, roleID, permissionID)
}

func (t *Txn) AddPermissionToRoleByTitle(roleID int64, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.AddPermissionToRoleByTitle(t.ctx, roleID, title)
}

func (t *Txn) RemovePermissionFromRole(roleID int64, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.RemovePermissionFromRole(t.ctx, roleID, permissionID)
}

func (t *Txn) DeleteRole(id int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.DeleteRole(t.ctx, id)
}

//===========================================================================
// PermissionStore
//===========================================================================

func (t *Txn) CreatePermission(permission *models.Permission) (*models.Permission, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreatePermission(t.ctx, permission)
}

func (t *Txn) ListPermissions(filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error) {
	return t.store.ListPermissions(t.ctx, filter)
}

func (t *Txn) RetrievePermission(id int64) (*models.Permission, error) {
	return t.store.RetrievePermission(t.ctx, id)
}

func (t *Txn) RetrievePermissionByTitle(title string) (*models.Permission, error) {
	return t.store.RetrievePermissionByTitle(t.ctx, title)
}

func (t *Txn) UpdatePermission(permission *models.Permission) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdatePermission(t.ctx, permission)
}

func (t *Txn) DeletePermission(id int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.DeletePermission(t.ctx, id)
}

//===========================================================================
// APIKeyStore
//===========================================================================

func (t *Txn) CreateAPIKey(key *models.APIKey) (*models.APIKey, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateAPIKey(t.ctx, key)
}

func (t *Txn) ListAPIKeys(filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error) {
	return t.store.ListAPIKeys(t.ctx, filter)
}

func (t *Txn) CreateAPIKeyFor(key *models.APIKey, creator ulid.ULID) (*models.APIKey, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateAPIKeyFor(t.ctx, key, creator)
}

func (t *Txn) RetrieveAPIKey(id ulid.ULID) (*models.APIKey, error) {
	return t.store.RetrieveAPIKey(t.ctx, id)
}

func (t *Txn) RetrieveAPIKeyByClientID(clientID string) (*models.APIKey, error) {
	return t.store.RetrieveAPIKeyByClientID(t.ctx, clientID)
}

func (t *Txn) UpdateAPIKey(key *models.APIKey) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateAPIKey(t.ctx, key)
}

func (t *Txn) UpdateLastSeen(keyID ulid.ULID, lastSeen time.Time) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateLastSeen(t.ctx, keyID, lastSeen)
}

func (t *Txn) AddPermissionToAPIKey(keyID ulid.ULID, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.AddPermissionToAPIKey(t.ctx, keyID, permissionID)
}

func (t *Txn) AddPermissionToAPIKeyByTitle(keyID ulid.ULID, title string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.AddPermissionToAPIKeyByTitle(t.ctx, keyID, title)
}

func (t *Txn) RemovePermissionFromAPIKey(keyID ulid.ULID, permissionID int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.RemovePermissionFromAPIKey(t.ctx, keyID, permissionID)
}

func (t *Txn) ReplaceAPIKeyPermissions(keyID ulid.ULID, permissionIDs []int64) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.ReplaceAPIKeyPermissions(t.ctx, keyID, permissionIDs)
}

func (t *Txn) RevokeAPIKey(keyID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.RevokeAPIKey(t.ctx, keyID)
}

func (t *Txn) DeleteAPIKey(keyID ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.DeleteAPIKey(t.ctx, keyID)
}

//===========================================================================
// OIDCClientStore
//===========================================================================

func (t *Txn) CreateOIDCClient(client *models.OIDCClient) (*models.OIDCClient, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateOIDCClient(t.ctx, client)
}

func (t *Txn) ListOIDCClients(filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error) {
	return t.store.ListOIDCClients(t.ctx, filter)
}

func (t *Txn) RetrieveOIDCClient(id ulid.ULID) (*models.OIDCClient, error) {
	return t.store.RetrieveOIDCClient(t.ctx, id)
}

func (t *Txn) RetrieveOIDCClientByClientID(clientID string) (*models.OIDCClient, error) {
	return t.store.RetrieveOIDCClientByClientID(t.ctx, clientID)
}

func (t *Txn) UpdateOIDCClient(client *models.OIDCClient) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateOIDCClient(t.ctx, client)
}

func (t *Txn) DeleteOIDCClient(id ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.DeleteOIDCClient(t.ctx, id)
}

//===========================================================================
// VeroTokenStore
//===========================================================================

func (t *Txn) CreateVeroToken(token *models.VeroToken) (*models.VeroToken, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateVeroToken(t.ctx, token)
}

func (t *Txn) CreateResetPasswordVeroToken(token *models.VeroToken) (*models.VeroToken, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateResetPasswordVeroToken(t.ctx, token)
}

func (t *Txn) CreateTeamInviteVeroToken(token *models.VeroToken) (*models.VeroToken, error) {
	if err := t.requireWrite(); err != nil {
		return nil, err
	}
	return t.store.CreateTeamInviteVeroToken(t.ctx, token)
}

func (t *Txn) RetrieveVeroToken(id ulid.ULID) (*models.VeroToken, error) {
	return t.store.RetrieveVeroToken(t.ctx, id)
}

func (t *Txn) RetrieveVeroTokenByResource(resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	return t.store.RetrieveVeroTokenByResource(t.ctx, resourceID, tokenType)
}

func (t *Txn) UpdateVeroToken(token *models.VeroToken) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.UpdateVeroToken(t.ctx, token)
}

func (t *Txn) DeleteVeroToken(id ulid.ULID) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.DeleteVeroToken(t.ctx, id)
}

func (t *Txn) CompletePasswordReset(veroTokenID ulid.ULID, newPassword string) error {
	if err := t.requireWrite(); err != nil {
		return err
	}
	return t.store.CompletePasswordReset(t.ctx, veroTokenID, newPassword)
}
