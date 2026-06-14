package txn

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

// Tx composes multiple store operations inside a single database transaction so that
// if all operations succeed the transaction can be committed; if any operation fails,
// the transaction can be rolled back. Tx mirrors [store.Store] methods but without
// requiring context (context is bound when the transaction is created). Use
// [StoreTx] to start and run transactions, which should be implemented by the
// [store.Store] implementation.
//
// NOTE: if a method is added to [store.Store] for transactional use, add it here too.
type Tx interface {
	Commit() error
	Rollback() error

	CreateUser(user *models.User) (*models.User, error)
	RetrieveUser(id ulid.ULID) (*models.User, error)
	RetrieveUserByEmail(email string) (*models.User, error)
	UpdateUser(user *models.User) error
	UpdatePassword(userID ulid.ULID, password string) error
	UpdateLastLogin(userID ulid.ULID, lastLogin time.Time) error
	VerifyEmail(userID ulid.ULID) error
	DeleteUser(userID ulid.ULID) error
	AddRoleToUser(userID ulid.ULID, roleID int64) error
	AddRoleToUserByTitle(userID ulid.ULID, title string) error
	RemoveRoleFromUser(userID ulid.ULID, roleID int64) error
	RemoveRoleFromUserByTitle(userID ulid.ULID, title string) error
	ReplaceUserRoles(userID ulid.ULID, roleIDs []int64) error
	// ListUsers returns a cursor over users matching filter. [tidal.Cursor.Close] rolls back
	// the transaction; use [tidal.Cursor.CloseRows] to release the result set and continue
	// using this transaction.
	ListUsers(filter tidal.ListFilter) (tidal.Cursor[*models.User], error)

	CreateRole(role *models.Role) (*models.Role, error)
	RetrieveRole(id int64) (*models.Role, error)
	RetrieveRoleByTitle(title string) (*models.Role, error)
	UpdateRole(role *models.Role) error
	AddPermissionToRole(roleID int64, permissionID int64) error
	AddPermissionToRoleByTitle(roleID int64, title string) error
	RemovePermissionFromRole(roleID int64, permissionID int64) error
	DeleteRole(id int64) error
	// ListRoles returns a cursor over roles matching filter. [tidal.Cursor.Close] rolls back
	// the transaction; use [tidal.Cursor.CloseRows] to release the result set and continue
	// using this transaction.
	ListRoles(filter tidal.ListFilter) (tidal.Cursor[*models.Role], error)

	CreatePermission(permission *models.Permission) (*models.Permission, error)
	RetrievePermission(id int64) (*models.Permission, error)
	RetrievePermissionByTitle(title string) (*models.Permission, error)
	UpdatePermission(permission *models.Permission) error
	DeletePermission(id int64) error
	// ListPermissions returns a cursor over permissions matching filter. [tidal.Cursor.Close]
	// rolls back the transaction; use [tidal.Cursor.CloseRows] to release the result set and
	// continue using this transaction.
	ListPermissions(filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error)

	CreateAPIKey(key *models.APIKey) (*models.APIKey, error)
	RetrieveAPIKey(id ulid.ULID) (*models.APIKey, error)
	RetrieveAPIKeyByClientID(clientID string) (*models.APIKey, error)
	UpdateAPIKey(key *models.APIKey) error
	UpdateLastSeen(keyID ulid.ULID, lastSeen time.Time) error
	AddPermissionToAPIKey(keyID ulid.ULID, permissionID int64) error
	AddPermissionToAPIKeyByTitle(keyID ulid.ULID, title string) error
	RemovePermissionFromAPIKey(keyID ulid.ULID, permissionID int64) error
	ReplaceAPIKeyPermissions(keyID ulid.ULID, permissionIDs []int64) error
	DeleteAPIKey(keyID ulid.ULID) error
	// RevokeAPIKey soft-revokes a key; DeleteAPIKey removes the row.
	RevokeAPIKey(keyID ulid.ULID) error
	// CreateAPIKeyFor creates a key on behalf of creator, who must be authorized to delegate.
	CreateAPIKeyFor(key *models.APIKey, creator ulid.ULID) (*models.APIKey, error)
	// ListAPIKeys returns a cursor over API keys matching filter. [tidal.Cursor.Close] rolls
	// back the transaction; use [tidal.Cursor.CloseRows] to release the result set and
	// continue using this transaction.
	ListAPIKeys(filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error)

	CreateOIDCClient(client *models.OIDCClient) (*models.OIDCClient, error)
	RetrieveOIDCClient(id ulid.ULID) (*models.OIDCClient, error)
	RetrieveOIDCClientByClientID(clientID string) (*models.OIDCClient, error)
	UpdateOIDCClient(client *models.OIDCClient) error
	DeleteOIDCClient(id ulid.ULID) error
	// ListOIDCClients returns a cursor over OIDC clients matching filter. [tidal.Cursor.Close]
	// rolls back the transaction; use [tidal.Cursor.CloseRows] to release the result set and
	// continue using this transaction.
	ListOIDCClients(filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error)

	CreateVeroToken(token *models.VeroToken) (*models.VeroToken, error)
	RetrieveVeroToken(id ulid.ULID) (*models.VeroToken, error)
	UpdateVeroToken(token *models.VeroToken) error
	DeleteVeroToken(id ulid.ULID) error
	// RetrieveVeroTokenByResource looks up a token by linked resource ID and type, not token ID.
	RetrieveVeroTokenByResource(resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error)
	// CreateResetPasswordVeroToken allows at most one unexpired reset-password token per resource.
	CreateResetPasswordVeroToken(token *models.VeroToken) (*models.VeroToken, error)
	// CreateTeamInviteVeroToken allows at most one unexpired team-invite token per resource.
	CreateTeamInviteVeroToken(token *models.VeroToken) (*models.VeroToken, error)
	// CompletePasswordReset validates the token, sets the password, and deletes the token.
	CompletePasswordReset(veroTokenID ulid.ULID, newPassword string) error
}

// StoreTx is the interface for Store transactional methods.
type StoreTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	BeginReadTx(ctx context.Context) (Tx, error)
	WithTx(ctx context.Context, opts *sql.TxOptions, fn func(Tx) error) error
	WithReadTx(ctx context.Context, fn func(Tx) error) error
}
