package store

import (
	"context"
	"database/sql"
	"io"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/backend"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/txn"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

// Store is a generic storage interface allowing multiple storage backends to be
// used based on the preference of the user.
//
// NOTE: to prevent import cycles, transactional methods live in [txn.Tx]. If a
// method is added here for transactional use, add it to [txn.Tx] as well.
type Store interface {
	io.Closer

	Stats() sql.DBStats

	txn.StoreTx

	UserStore
	RoleStore
	PermissionStore
	APIKeyStore
	OIDCClientStore
	VeroTokenStore
}

// Check that [backend.Store] implements [Store].
var _ Store = (*backend.Store)(nil)

type UserStore interface {
	ListUsers(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.User], error)
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	RetrieveUser(ctx context.Context, id ulid.ULID) (*models.User, error)
	RetrieveUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	UpdatePassword(ctx context.Context, userID ulid.ULID, password string) error
	UpdateLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) error
	VerifyEmail(ctx context.Context, userID ulid.ULID) error
	DeleteUser(ctx context.Context, userID ulid.ULID) error
	AddRoleToUser(ctx context.Context, userID ulid.ULID, roleID int64) error
	AddRoleToUserByTitle(ctx context.Context, userID ulid.ULID, title string) error
	RemoveRoleFromUser(ctx context.Context, userID ulid.ULID, roleID int64) error
	RemoveRoleFromUserByTitle(ctx context.Context, userID ulid.ULID, title string) error
	ReplaceUserRoles(ctx context.Context, userID ulid.ULID, roleIDs []int64) error
}

type RoleStore interface {
	ListRoles(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Role], error)
	CreateRole(ctx context.Context, role *models.Role) (*models.Role, error)
	RetrieveRole(ctx context.Context, id int64) (*models.Role, error)
	RetrieveRoleByTitle(ctx context.Context, title string) (*models.Role, error)
	UpdateRole(ctx context.Context, role *models.Role) error
	AddPermissionToRole(ctx context.Context, roleID int64, permissionID int64) error
	AddPermissionToRoleByTitle(ctx context.Context, roleID int64, title string) error
	RemovePermissionFromRole(ctx context.Context, roleID int64, permissionID int64) error
	DeleteRole(ctx context.Context, id int64) error
}

type PermissionStore interface {
	ListPermissions(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error)
	CreatePermission(ctx context.Context, permission *models.Permission) (*models.Permission, error)
	RetrievePermission(ctx context.Context, id int64) (*models.Permission, error)
	RetrievePermissionByTitle(ctx context.Context, title string) (*models.Permission, error)
	UpdatePermission(ctx context.Context, permission *models.Permission) error
	DeletePermission(ctx context.Context, id int64) error
}

type APIKeyStore interface {
	ListAPIKeys(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error)
	CreateAPIKey(ctx context.Context, key *models.APIKey) (*models.APIKey, error)
	RetrieveAPIKey(ctx context.Context, id ulid.ULID) (*models.APIKey, error)
	RetrieveAPIKeyByClientID(ctx context.Context, clientID string) (*models.APIKey, error)
	UpdateAPIKey(ctx context.Context, key *models.APIKey) error
	UpdateLastSeen(ctx context.Context, keyID ulid.ULID, lastSeen time.Time) error
	AddPermissionToAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) error
	AddPermissionToAPIKeyByTitle(ctx context.Context, keyID ulid.ULID, title string) error
	RemovePermissionFromAPIKey(ctx context.Context, keyID ulid.ULID, permissionID int64) error
	ReplaceAPIKeyPermissions(ctx context.Context, keyID ulid.ULID, permissionIDs []int64) error
	DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error
	// RevokeAPIKey soft-revokes a key; DeleteAPIKey removes the row.
	RevokeAPIKey(ctx context.Context, keyID ulid.ULID) error
	// CreateAPIKeyFor creates a key on behalf of creator, who must be authorized to delegate.
	CreateAPIKeyFor(ctx context.Context, key *models.APIKey, creator ulid.ULID) (*models.APIKey, error)
}

type OIDCClientStore interface {
	ListOIDCClients(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error)
	CreateOIDCClient(ctx context.Context, client *models.OIDCClient) (*models.OIDCClient, error)
	RetrieveOIDCClient(ctx context.Context, id ulid.ULID) (*models.OIDCClient, error)
	RetrieveOIDCClientByClientID(ctx context.Context, clientID string) (*models.OIDCClient, error)
	UpdateOIDCClient(ctx context.Context, client *models.OIDCClient) error
	DeleteOIDCClient(ctx context.Context, id ulid.ULID) error
}

type VeroTokenStore interface {
	CreateVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error)
	RetrieveVeroToken(ctx context.Context, id ulid.ULID) (*models.VeroToken, error)
	UpdateVeroToken(ctx context.Context, token *models.VeroToken) error
	DeleteVeroToken(ctx context.Context, id ulid.ULID) error
	// RetrieveVeroTokenByResource looks up a token by linked resource ID and type, not token ID.
	RetrieveVeroTokenByResource(ctx context.Context, resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error)
	// CreateResetPasswordVeroToken allows at most one unexpired reset-password token per resource.
	CreateResetPasswordVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error)
	// CreateTeamInviteVeroToken allows at most one unexpired team-invite token per resource.
	CreateTeamInviteVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error)
	// CompletePasswordReset validates the token, sets the password, and deletes the token.
	CompletePasswordReset(ctx context.Context, veroTokenID ulid.ULID, newPassword string) error
}
