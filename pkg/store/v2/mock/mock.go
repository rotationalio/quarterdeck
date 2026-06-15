package mock

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/models"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/dsn"
)

// Method names for the Store interface.
const (
	Close                 = "Close"
	Stats                 = "Stats"
	CompletePasswordReset = "CompletePasswordReset"
	CreateAPIKeyFor       = "CreateAPIKeyFor"
)

// Store implements the store.Store interface with callback functions that tests can
// specify to simulate a specific behavior. The Store is not thread-safe and one mock
// store should be used per test.
type Store struct {
	calls    map[string]int
	readOnly bool

	// Store callbacks
	OnClose func() error
	OnStats func() sql.DBStats

	// Composite callbacks
	OnCompletePasswordReset  func(context.Context, ulid.ULID, string) error
	OnCreateAPIKeyForCreator func(context.Context, *models.APIKey, ulid.ULID) (*models.APIKey, error)

	// UserStore callbacks
	OnListUsers                 func(context.Context, tidal.ListFilter) (tidal.Cursor[*models.User], error)
	OnCreateUser                func(context.Context, *models.User) (*models.User, error)
	OnRetrieveUser              func(context.Context, ulid.ULID) (*models.User, error)
	OnRetrieveUserByEmail       func(context.Context, string) (*models.User, error)
	OnUpdateUser                func(context.Context, *models.User) error
	OnUpdatePassword            func(context.Context, ulid.ULID, string) error
	OnUpdateLastLogin           func(context.Context, ulid.ULID, time.Time) error
	OnVerifyEmail               func(context.Context, ulid.ULID) error
	OnDeleteUser                func(context.Context, ulid.ULID) error
	OnAddRoleToUser             func(context.Context, ulid.ULID, int64) error
	OnAddRoleToUserByTitle      func(context.Context, ulid.ULID, string) error
	OnRemoveRoleFromUser        func(context.Context, ulid.ULID, int64) error
	OnRemoveRoleFromUserByTitle func(context.Context, ulid.ULID, string) error
	OnReplaceUserRoles          func(context.Context, ulid.ULID, []int64) error

	// RoleStore callbacks
	OnListRoles                  func(context.Context, tidal.ListFilter) (tidal.Cursor[*models.Role], error)
	OnCreateRole                 func(context.Context, *models.Role) (*models.Role, error)
	OnRetrieveRole               func(context.Context, int64) (*models.Role, error)
	OnRetrieveRoleByTitle        func(context.Context, string) (*models.Role, error)
	OnUpdateRole                 func(context.Context, *models.Role) error
	OnAddPermissionToRole        func(context.Context, int64, int64) error
	OnAddPermissionToRoleByTitle func(context.Context, int64, string) error
	OnRemovePermissionFromRole   func(context.Context, int64, int64) error
	OnDeleteRole                 func(context.Context, int64) error

	// PermissionStore callbacks
	OnListPermissions           func(context.Context, tidal.ListFilter) (tidal.Cursor[*models.Permission], error)
	OnCreatePermission          func(context.Context, *models.Permission) (*models.Permission, error)
	OnRetrievePermission        func(context.Context, int64) (*models.Permission, error)
	OnRetrievePermissionByTitle func(context.Context, string) (*models.Permission, error)
	OnUpdatePermission          func(context.Context, *models.Permission) error
	OnDeletePermission          func(context.Context, int64) error

	// APIKeyStore callbacks
	OnListAPIKeys                  func(context.Context, tidal.ListFilter) (tidal.Cursor[*models.APIKey], error)
	OnCreateAPIKey                 func(context.Context, *models.APIKey) (*models.APIKey, error)
	OnRetrieveAPIKey               func(context.Context, ulid.ULID) (*models.APIKey, error)
	OnRetrieveAPIKeyByClientID     func(context.Context, string) (*models.APIKey, error)
	OnUpdateAPIKey                 func(context.Context, *models.APIKey) error
	OnUpdateLastSeen               func(context.Context, ulid.ULID, time.Time) error
	OnAddPermissionToAPIKey        func(context.Context, ulid.ULID, int64) error
	OnAddPermissionToAPIKeyByTitle func(context.Context, ulid.ULID, string) error
	OnRemovePermissionFromAPIKey   func(context.Context, ulid.ULID, int64) error
	OnReplaceAPIKeyPermissions     func(context.Context, ulid.ULID, []int64) error
	OnRevokeAPIKey                 func(context.Context, ulid.ULID) error
	OnDeleteAPIKey                 func(context.Context, ulid.ULID) error

	// OIDCClientStore callbacks
	OnListOIDCClients              func(context.Context, tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error)
	OnCreateOIDCClient             func(context.Context, *models.OIDCClient) (*models.OIDCClient, error)
	OnRetrieveOIDCClient           func(context.Context, ulid.ULID) (*models.OIDCClient, error)
	OnRetrieveOIDCClientByClientID func(context.Context, string) (*models.OIDCClient, error)
	OnUpdateOIDCClient             func(context.Context, *models.OIDCClient) error
	OnDeleteOIDCClient             func(context.Context, ulid.ULID) error

	// VeroTokenStore callbacks
	OnCreateVeroToken              func(context.Context, *models.VeroToken) (*models.VeroToken, error)
	OnRetrieveVeroToken            func(context.Context, ulid.ULID) (*models.VeroToken, error)
	OnRetrieveVeroTokenByResource  func(context.Context, ulid.ULID, enum.TokenType) (*models.VeroToken, error)
	OnUpdateVeroToken              func(context.Context, *models.VeroToken) error
	OnDeleteVeroToken              func(context.Context, ulid.ULID) error
	OnCreateResetPasswordVeroToken func(context.Context, *models.VeroToken) (*models.VeroToken, error)
	OnCreateTeamInviteVeroToken    func(context.Context, *models.VeroToken) (*models.VeroToken, error)
}

func Open(uri *dsn.DSN) (*Store, error) {
	if uri != nil && uri.Provider != dsn.Mock {
		return nil, errors.ErrUnknownScheme
	}

	if uri == nil {
		uri = &dsn.DSN{Provider: dsn.Mock}
	}

	return &Store{
		calls:    make(map[string]int),
		readOnly: uri != nil && uri.Options.ReadOnly(),
	}, nil
}

//===========================================================================
// Mock helper methods
//===========================================================================

// Reset all the calls and callbacks in the store.
func (s *Store) Reset() {
	s.calls = nil
	s.calls = make(map[string]int)

	v := reflect.ValueOf(s).Elem()
	t := v.Type()
	for _, f := range reflect.VisibleFields(t) {
		if strings.HasPrefix(f.Name, "On") && f.Type.Kind() == reflect.Func {
			fv := v.FieldByIndex(f.Index)
			fv.SetZero()
		}
	}
}

// Assert that the expected number of calls were made to the given method.
func (s *Store) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, s.calls[method], "expected %d calls to %s, got %d", expected, method, s.calls[method])
}

//===========================================================================
// Store interface methods
//===========================================================================

func (s *Store) Close() error {
	s.calls[Close]++
	if s.OnClose != nil {
		return s.OnClose()
	}
	return nil
}

func (s *Store) Stats() sql.DBStats {
	s.calls[Stats]++
	if s.OnStats != nil {
		return s.OnStats()
	}
	return sql.DBStats{}
}

func (s *Store) CompletePasswordReset(ctx context.Context, veroTokenID ulid.ULID, newPassword string) error {
	s.calls[CompletePasswordReset]++
	if s.OnCompletePasswordReset != nil {
		return s.OnCompletePasswordReset(ctx, veroTokenID, newPassword)
	}
	panic(errors.Fmt("%s callback is not mocked", CompletePasswordReset))
}

func (s *Store) CreateAPIKeyFor(ctx context.Context, key *models.APIKey, creatorAPIKeyID ulid.ULID) (*models.APIKey, error) {
	s.calls[CreateAPIKeyFor]++
	if s.OnCreateAPIKeyForCreator != nil {
		return s.OnCreateAPIKeyForCreator(ctx, key, creatorAPIKeyID)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateAPIKeyFor))
}

//===========================================================================
// UserStore
//===========================================================================

const (
	ListUsers                 = "ListUsers"
	CreateUser                = "CreateUser"
	RetrieveUser              = "RetrieveUser"
	RetrieveUserByEmail       = "RetrieveUserByEmail"
	UpdateUser                = "UpdateUser"
	UpdatePassword            = "UpdatePassword"
	UpdateLastLogin           = "UpdateLastLogin"
	VerifyEmail               = "VerifyEmail"
	DeleteUser                = "DeleteUser"
	AddRoleToUser             = "AddRoleToUser"
	AddRoleToUserByTitle      = "AddRoleToUserByTitle"
	RemoveRoleFromUser        = "RemoveRoleFromUser"
	RemoveRoleFromUserByTitle = "RemoveRoleFromUserByTitle"
	ReplaceUserRoles          = "ReplaceUserRoles"
)

func (s *Store) ListUsers(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.User], error) {
	s.calls[ListUsers]++
	if s.OnListUsers != nil {
		return s.OnListUsers(ctx, filter)
	}
	panic(errors.Fmt("%s callback is not mocked", ListUsers))
}

func (s *Store) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	s.calls[CreateUser]++
	if s.OnCreateUser != nil {
		return s.OnCreateUser(ctx, user)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateUser))
}

func (s *Store) RetrieveUser(ctx context.Context, id ulid.ULID) (*models.User, error) {
	s.calls[RetrieveUser]++
	if s.OnRetrieveUser != nil {
		return s.OnRetrieveUser(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveUser))
}

func (s *Store) RetrieveUserByEmail(ctx context.Context, email string) (*models.User, error) {
	s.calls[RetrieveUserByEmail]++
	if s.OnRetrieveUserByEmail != nil {
		return s.OnRetrieveUserByEmail(ctx, email)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveUserByEmail))
}

func (s *Store) UpdateUser(ctx context.Context, user *models.User) error {
	s.calls[UpdateUser]++
	if s.OnUpdateUser != nil {
		return s.OnUpdateUser(ctx, user)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateUser))
}

func (s *Store) UpdatePassword(ctx context.Context, id ulid.ULID, password string) error {
	s.calls[UpdatePassword]++
	if s.OnUpdatePassword != nil {
		return s.OnUpdatePassword(ctx, id, password)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdatePassword))
}

func (s *Store) UpdateLastLogin(ctx context.Context, id ulid.ULID, lastLogin time.Time) error {
	s.calls[UpdateLastLogin]++
	if s.OnUpdateLastLogin != nil {
		return s.OnUpdateLastLogin(ctx, id, lastLogin)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateLastLogin))
}

func (s *Store) VerifyEmail(ctx context.Context, id ulid.ULID) error {
	s.calls[VerifyEmail]++
	if s.OnVerifyEmail != nil {
		return s.OnVerifyEmail(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", VerifyEmail))
}

func (s *Store) DeleteUser(ctx context.Context, id ulid.ULID) error {
	s.calls[DeleteUser]++
	if s.OnDeleteUser != nil {
		return s.OnDeleteUser(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteUser))
}

func (s *Store) AddRoleToUser(ctx context.Context, userID ulid.ULID, roleID int64) error {
	s.calls[AddRoleToUser]++
	if s.OnAddRoleToUser != nil {
		return s.OnAddRoleToUser(ctx, userID, roleID)
	}
	panic(errors.Fmt("%s callback is not mocked", AddRoleToUser))
}

func (s *Store) AddRoleToUserByTitle(ctx context.Context, userID ulid.ULID, title string) error {
	s.calls[AddRoleToUserByTitle]++
	if s.OnAddRoleToUserByTitle != nil {
		return s.OnAddRoleToUserByTitle(ctx, userID, title)
	}
	panic(errors.Fmt("%s callback is not mocked", AddRoleToUserByTitle))
}

func (s *Store) RemoveRoleFromUser(ctx context.Context, userID ulid.ULID, roleID int64) error {
	s.calls[RemoveRoleFromUser]++
	if s.OnRemoveRoleFromUser != nil {
		return s.OnRemoveRoleFromUser(ctx, userID, roleID)
	}
	panic(errors.Fmt("%s callback is not mocked", RemoveRoleFromUser))
}

func (s *Store) RemoveRoleFromUserByTitle(ctx context.Context, userID ulid.ULID, title string) error {
	s.calls[RemoveRoleFromUserByTitle]++
	if s.OnRemoveRoleFromUserByTitle != nil {
		return s.OnRemoveRoleFromUserByTitle(ctx, userID, title)
	}
	panic(errors.Fmt("%s callback is not mocked", RemoveRoleFromUserByTitle))
}

func (s *Store) ReplaceUserRoles(ctx context.Context, userID ulid.ULID, roleIDs []int64) error {
	s.calls[ReplaceUserRoles]++
	if s.OnReplaceUserRoles != nil {
		return s.OnReplaceUserRoles(ctx, userID, roleIDs)
	}
	panic(errors.Fmt("%s callback is not mocked", ReplaceUserRoles))
}

//===========================================================================
// RoleStore
//===========================================================================

const (
	ListRoles                  = "ListRoles"
	CreateRole                 = "CreateRole"
	RetrieveRole               = "RetrieveRole"
	RetrieveRoleByTitle        = "RetrieveRoleByTitle"
	UpdateRole                 = "UpdateRole"
	AddPermissionToRole        = "AddPermissionToRole"
	AddPermissionToRoleByTitle = "AddPermissionToRoleByTitle"
	RemovePermissionFromRole   = "RemovePermissionFromRole"
	DeleteRole                 = "DeleteRole"
)

func (s *Store) ListRoles(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Role], error) {
	s.calls[ListRoles]++
	if s.OnListRoles != nil {
		return s.OnListRoles(ctx, filter)
	}
	panic(errors.Fmt("%s callback is not mocked", ListRoles))
}

func (s *Store) CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	s.calls[CreateRole]++
	if s.OnCreateRole != nil {
		return s.OnCreateRole(ctx, role)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateRole))
}

func (s *Store) RetrieveRole(ctx context.Context, id int64) (*models.Role, error) {
	s.calls[RetrieveRole]++
	if s.OnRetrieveRole != nil {
		return s.OnRetrieveRole(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveRole))
}

func (s *Store) RetrieveRoleByTitle(ctx context.Context, title string) (*models.Role, error) {
	s.calls[RetrieveRoleByTitle]++
	if s.OnRetrieveRoleByTitle != nil {
		return s.OnRetrieveRoleByTitle(ctx, title)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveRoleByTitle))
}

func (s *Store) UpdateRole(ctx context.Context, role *models.Role) error {
	s.calls[UpdateRole]++
	if s.OnUpdateRole != nil {
		return s.OnUpdateRole(ctx, role)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateRole))
}

func (s *Store) AddPermissionToRole(ctx context.Context, roleID int64, permissionID int64) error {
	s.calls[AddPermissionToRole]++
	if s.OnAddPermissionToRole != nil {
		return s.OnAddPermissionToRole(ctx, roleID, permissionID)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToRole))
}

func (s *Store) AddPermissionToRoleByTitle(ctx context.Context, roleID int64, title string) error {
	s.calls[AddPermissionToRoleByTitle]++
	if s.OnAddPermissionToRoleByTitle != nil {
		return s.OnAddPermissionToRoleByTitle(ctx, roleID, title)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToRoleByTitle))
}

func (s *Store) RemovePermissionFromRole(ctx context.Context, roleID int64, permissionID int64) error {
	s.calls[RemovePermissionFromRole]++
	if s.OnRemovePermissionFromRole != nil {
		return s.OnRemovePermissionFromRole(ctx, roleID, permissionID)
	}
	panic(errors.Fmt("%s callback is not mocked", RemovePermissionFromRole))
}

func (s *Store) DeleteRole(ctx context.Context, id int64) error {
	s.calls[DeleteRole]++
	if s.OnDeleteRole != nil {
		return s.OnDeleteRole(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteRole))
}

//===========================================================================
// PermissionStore
//===========================================================================

const (
	ListPermissions           = "ListPermissions"
	CreatePermission          = "CreatePermission"
	RetrievePermission        = "RetrievePermission"
	RetrievePermissionByTitle = "RetrievePermissionByTitle"
	UpdatePermission          = "UpdatePermission"
	DeletePermission          = "DeletePermission"
)

func (s *Store) ListPermissions(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.Permission], error) {
	s.calls[ListPermissions]++
	if s.OnListPermissions != nil {
		return s.OnListPermissions(ctx, filter)
	}
	panic(errors.Fmt("%s callback is not mocked", ListPermissions))
}

func (s *Store) CreatePermission(ctx context.Context, permission *models.Permission) (*models.Permission, error) {
	s.calls[CreatePermission]++
	if s.OnCreatePermission != nil {
		return s.OnCreatePermission(ctx, permission)
	}
	panic(errors.Fmt("%s callback is not mocked", CreatePermission))
}

func (s *Store) RetrievePermission(ctx context.Context, id int64) (*models.Permission, error) {
	s.calls[RetrievePermission]++
	if s.OnRetrievePermission != nil {
		return s.OnRetrievePermission(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrievePermission))
}

func (s *Store) RetrievePermissionByTitle(ctx context.Context, title string) (*models.Permission, error) {
	s.calls[RetrievePermissionByTitle]++
	if s.OnRetrievePermissionByTitle != nil {
		return s.OnRetrievePermissionByTitle(ctx, title)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrievePermissionByTitle))
}

func (s *Store) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	s.calls[UpdatePermission]++
	if s.OnUpdatePermission != nil {
		return s.OnUpdatePermission(ctx, permission)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdatePermission))
}

func (s *Store) DeletePermission(ctx context.Context, id int64) error {
	s.calls[DeletePermission]++
	if s.OnDeletePermission != nil {
		return s.OnDeletePermission(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeletePermission))
}

//===========================================================================
// APIKeyStore
//===========================================================================

const (
	ListAPIKeys                  = "ListAPIKeys"
	CreateAPIKey                 = "CreateAPIKey"
	RetrieveAPIKey               = "RetrieveAPIKey"
	RetrieveAPIKeyByClientID     = "RetrieveAPIKeyByClientID"
	UpdateAPIKey                 = "UpdateAPIKey"
	UpdateLastSeen               = "UpdateLastSeen"
	AddPermissionToAPIKey        = "AddPermissionToAPIKey"
	AddPermissionToAPIKeyByTitle = "AddPermissionToAPIKeyByTitle"
	RemovePermissionFromAPIKey   = "RemovePermissionFromAPIKey"
	ReplaceAPIKeyPermissions     = "ReplaceAPIKeyPermissions"
	RevokeAPIKey                 = "RevokeAPIKey"
	DeleteAPIKey                 = "DeleteAPIKey"
)

func (s *Store) ListAPIKeys(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.APIKey], error) {
	s.calls[ListAPIKeys]++
	if s.OnListAPIKeys != nil {
		return s.OnListAPIKeys(ctx, filter)
	}
	panic(errors.Fmt("%s callback is not mocked", ListAPIKeys))
}

func (s *Store) CreateAPIKey(ctx context.Context, key *models.APIKey) (*models.APIKey, error) {
	s.calls[CreateAPIKey]++
	if s.OnCreateAPIKey != nil {
		return s.OnCreateAPIKey(ctx, key)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateAPIKey))
}

func (s *Store) RetrieveAPIKey(ctx context.Context, id ulid.ULID) (*models.APIKey, error) {
	s.calls[RetrieveAPIKey]++
	if s.OnRetrieveAPIKey != nil {
		return s.OnRetrieveAPIKey(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveAPIKey))
}

func (s *Store) RetrieveAPIKeyByClientID(ctx context.Context, clientID string) (*models.APIKey, error) {
	s.calls[RetrieveAPIKeyByClientID]++
	if s.OnRetrieveAPIKeyByClientID != nil {
		return s.OnRetrieveAPIKeyByClientID(ctx, clientID)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveAPIKeyByClientID))
}

func (s *Store) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	s.calls[UpdateAPIKey]++
	if s.OnUpdateAPIKey != nil {
		return s.OnUpdateAPIKey(ctx, key)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateAPIKey))
}

func (s *Store) UpdateLastSeen(ctx context.Context, id ulid.ULID, lastSeen time.Time) error {
	s.calls[UpdateLastSeen]++
	if s.OnUpdateLastSeen != nil {
		return s.OnUpdateLastSeen(ctx, id, lastSeen)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateLastSeen))
}

func (s *Store) AddPermissionToAPIKey(ctx context.Context, id ulid.ULID, permissionID int64) error {
	s.calls[AddPermissionToAPIKey]++
	if s.OnAddPermissionToAPIKey != nil {
		return s.OnAddPermissionToAPIKey(ctx, id, permissionID)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToAPIKey))
}

func (s *Store) AddPermissionToAPIKeyByTitle(ctx context.Context, id ulid.ULID, title string) error {
	s.calls[AddPermissionToAPIKeyByTitle]++
	if s.OnAddPermissionToAPIKeyByTitle != nil {
		return s.OnAddPermissionToAPIKeyByTitle(ctx, id, title)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToAPIKeyByTitle))
}

func (s *Store) RemovePermissionFromAPIKey(ctx context.Context, id ulid.ULID, permissionID int64) error {
	s.calls[RemovePermissionFromAPIKey]++
	if s.OnRemovePermissionFromAPIKey != nil {
		return s.OnRemovePermissionFromAPIKey(ctx, id, permissionID)
	}
	panic(errors.Fmt("%s callback is not mocked", RemovePermissionFromAPIKey))
}

func (s *Store) ReplaceAPIKeyPermissions(ctx context.Context, id ulid.ULID, permissionIDs []int64) error {
	s.calls[ReplaceAPIKeyPermissions]++
	if s.OnReplaceAPIKeyPermissions != nil {
		return s.OnReplaceAPIKeyPermissions(ctx, id, permissionIDs)
	}
	panic(errors.Fmt("%s callback is not mocked", ReplaceAPIKeyPermissions))
}

func (s *Store) RevokeAPIKey(ctx context.Context, id ulid.ULID) error {
	s.calls[RevokeAPIKey]++
	if s.OnRevokeAPIKey != nil {
		return s.OnRevokeAPIKey(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RevokeAPIKey))
}

func (s *Store) DeleteAPIKey(ctx context.Context, id ulid.ULID) error {
	s.calls[DeleteAPIKey]++
	if s.OnDeleteAPIKey != nil {
		return s.OnDeleteAPIKey(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteAPIKey))
}

//===========================================================================
// OIDCClientStore
//===========================================================================

const (
	ListOIDCClients              = "ListOIDCClients"
	CreateOIDCClient             = "CreateOIDCClient"
	RetrieveOIDCClient           = "RetrieveOIDCClient"
	RetrieveOIDCClientByClientID = "RetrieveOIDCClientByClientID"
	UpdateOIDCClient             = "UpdateOIDCClient"
	DeleteOIDCClient             = "DeleteOIDCClient"
)

func (s *Store) ListOIDCClients(ctx context.Context, filter tidal.ListFilter) (tidal.Cursor[*models.OIDCClient], error) {
	s.calls[ListOIDCClients]++
	if s.OnListOIDCClients != nil {
		return s.OnListOIDCClients(ctx, filter)
	}
	panic(errors.Fmt("%s callback is not mocked", ListOIDCClients))
}

func (s *Store) CreateOIDCClient(ctx context.Context, client *models.OIDCClient) (*models.OIDCClient, error) {
	s.calls[CreateOIDCClient]++
	if s.OnCreateOIDCClient != nil {
		return s.OnCreateOIDCClient(ctx, client)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateOIDCClient))
}

func (s *Store) RetrieveOIDCClient(ctx context.Context, id ulid.ULID) (*models.OIDCClient, error) {
	s.calls[RetrieveOIDCClient]++
	if s.OnRetrieveOIDCClient != nil {
		return s.OnRetrieveOIDCClient(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveOIDCClient))
}

func (s *Store) RetrieveOIDCClientByClientID(ctx context.Context, clientID string) (*models.OIDCClient, error) {
	s.calls[RetrieveOIDCClientByClientID]++
	if s.OnRetrieveOIDCClientByClientID != nil {
		return s.OnRetrieveOIDCClientByClientID(ctx, clientID)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveOIDCClientByClientID))
}

func (s *Store) UpdateOIDCClient(ctx context.Context, client *models.OIDCClient) error {
	s.calls[UpdateOIDCClient]++
	if s.OnUpdateOIDCClient != nil {
		return s.OnUpdateOIDCClient(ctx, client)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateOIDCClient))
}

func (s *Store) DeleteOIDCClient(ctx context.Context, id ulid.ULID) error {
	s.calls[DeleteOIDCClient]++
	if s.OnDeleteOIDCClient != nil {
		return s.OnDeleteOIDCClient(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteOIDCClient))
}

//===========================================================================
// VeroTokenStore
//===========================================================================

const (
	CreateVeroToken              = "CreateVeroToken"
	RetrieveVeroToken            = "RetrieveVeroToken"
	RetrieveVeroTokenByResource  = "RetrieveVeroTokenByResource"
	UpdateVeroToken              = "UpdateVeroToken"
	DeleteVeroToken              = "DeleteVeroToken"
	CreateResetPasswordVeroToken = "CreateResetPasswordVeroToken"
	CreateTeamInviteVeroToken    = "CreateTeamInviteVeroToken"
)

func (s *Store) CreateVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	s.calls[CreateVeroToken]++
	if s.OnCreateVeroToken != nil {
		return s.OnCreateVeroToken(ctx, token)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateVeroToken))
}

func (s *Store) RetrieveVeroToken(ctx context.Context, id ulid.ULID) (*models.VeroToken, error) {
	s.calls[RetrieveVeroToken]++
	if s.OnRetrieveVeroToken != nil {
		return s.OnRetrieveVeroToken(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveVeroToken))
}

func (s *Store) RetrieveVeroTokenByResource(ctx context.Context, resourceID ulid.ULID, tokenType enum.TokenType) (*models.VeroToken, error) {
	s.calls[RetrieveVeroTokenByResource]++
	if s.OnRetrieveVeroTokenByResource != nil {
		return s.OnRetrieveVeroTokenByResource(ctx, resourceID, tokenType)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveVeroTokenByResource))
}

func (s *Store) UpdateVeroToken(ctx context.Context, token *models.VeroToken) error {
	s.calls[UpdateVeroToken]++
	if s.OnUpdateVeroToken != nil {
		return s.OnUpdateVeroToken(ctx, token)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateVeroToken))
}

func (s *Store) DeleteVeroToken(ctx context.Context, id ulid.ULID) error {
	s.calls[DeleteVeroToken]++
	if s.OnDeleteVeroToken != nil {
		return s.OnDeleteVeroToken(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteVeroToken))
}

func (s *Store) CreateResetPasswordVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	s.calls[CreateResetPasswordVeroToken]++
	if s.OnCreateResetPasswordVeroToken != nil {
		return s.OnCreateResetPasswordVeroToken(ctx, token)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateResetPasswordVeroToken))
}

func (s *Store) CreateTeamInviteVeroToken(ctx context.Context, token *models.VeroToken) (*models.VeroToken, error) {
	s.calls[CreateTeamInviteVeroToken]++
	if s.OnCreateTeamInviteVeroToken != nil {
		return s.OnCreateTeamInviteVeroToken(ctx, token)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateTeamInviteVeroToken))
}
