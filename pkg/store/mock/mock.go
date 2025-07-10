package mock

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/dsn"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
	"go.rtnl.ai/ulid"
)

// Method names for the Store interface
const (
	Close = "Close"
	Begin = "Begin"
)

// Store implements the store.Store interface with callback functions that tests can
// specify to simulate a specific behavior. The Store is not thread-safe and one mock
// store should be used per test.
type Store struct {
	calls    map[string]int
	readonly bool

	// Store Callbacks
	OnClose func() error
	OnBegin func(context.Context, *sql.TxOptions) (txn.Txn, error)

	// UserStore Callbacks
	OnListUsers       func(context.Context, *models.UserPage) (*models.UserList, error)
	OnCreateUser      func(context.Context, *models.User) error
	OnRetrieveUser    func(context.Context, any) (*models.User, error)
	OnUpdateUser      func(context.Context, *models.User) error
	OnUpdatePassword  func(context.Context, ulid.ULID, string) error
	OnUpdateLastLogin func(context.Context, ulid.ULID, time.Time) error
	OnDeleteUser      func(context.Context, ulid.ULID) error

	// RoleStore Callbacks
	OnListRoles                func(context.Context, *models.Page) (*models.RoleList, error)
	OnCreateRole               func(context.Context, *models.Role) error
	OnRetrieveRole             func(context.Context, any) (*models.Role, error)
	OnUpdateRole               func(context.Context, *models.Role) error
	OnAddPermissionToRole      func(context.Context, int64, any) error
	OnRemovePermissionFromRole func(context.Context, int64, int64) error
	OnDeleteRole               func(context.Context, int64) error

	// PermissionStore Callbacks
	OnListPermissions    func(context.Context, *models.Page) (*models.PermissionList, error)
	OnCreatePermission   func(context.Context, *models.Permission) error
	OnRetrievePermission func(context.Context, any) (*models.Permission, error)
	OnUpdatePermission   func(context.Context, *models.Permission) error
	OnDeletePermission   func(context.Context, int64) error
}

func Open(uri *dsn.DSN) (*Store, error) {
	if uri != nil && uri.Scheme != dsn.Mock {
		return nil, errors.ErrUnknownScheme
	}

	if uri == nil {
		uri = &dsn.DSN{ReadOnly: false, Scheme: dsn.Mock}
	}

	return &Store{
		calls:    make(map[string]int),
		readonly: uri.ReadOnly,
	}, nil
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the store.
func (s *Store) Reset() {
	// reset the call counts
	s.calls = nil
	s.calls = make(map[string]int)

	// reset the callbacks using reflection
	v := reflect.ValueOf(s).Elem()
	t := v.Type()
	for _, f := range reflect.VisibleFields(t) {
		// only reset functions named `OnSomething`
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
// Store Interface Methods
//===========================================================================

func (s *Store) Close() error {
	s.calls[Close]++
	if s.OnClose != nil {
		return s.OnClose()
	}
	return nil
}

func (s *Store) Begin(ctx context.Context, opts *sql.TxOptions) (txn.Txn, error) {
	s.calls[Begin]++
	if s.OnBegin != nil {
		return s.OnBegin(ctx, opts)
	}

	if opts == nil {
		opts = &sql.TxOptions{ReadOnly: s.readonly}
	} else if s.readonly && !opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	return &Tx{
		opts:  opts,
		calls: make(map[string]int),
	}, nil
}

//===========================================================================
// UserStore
//===========================================================================

const (
	ListUsers       = "ListUsers"
	CreateUser      = "CreateUser"
	RetrieveUser    = "RetrieveUser"
	UpdateUser      = "UpdateUser"
	UpdatePassword  = "UpdatePassword"
	UpdateLastLogin = "UpdateLastLogin"
	DeleteUser      = "DeleteUser"
)

func (s *Store) ListUsers(ctx context.Context, page *models.UserPage) (*models.UserList, error) {
	s.calls[ListUsers]++
	if s.OnListUsers != nil {
		return s.OnListUsers(ctx, page)
	}
	panic(errors.Fmt("%s callback is not mocked", ListUsers))
}

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	s.calls[CreateUser]++
	if s.OnCreateUser != nil {
		return s.OnCreateUser(ctx, user)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateUser))
}

func (s *Store) RetrieveUser(ctx context.Context, id any) (*models.User, error) {
	s.calls[RetrieveUser]++
	if s.OnRetrieveUser != nil {
		return s.OnRetrieveUser(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveUser))
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

func (s *Store) DeleteUser(ctx context.Context, id ulid.ULID) error {
	s.calls[DeleteUser]++
	if s.OnDeleteUser != nil {
		return s.OnDeleteUser(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteUser))
}

//===========================================================================
// RoleStore
//===========================================================================

const (
	ListRoles                = "ListRoles"
	CreateRole               = "CreateRole"
	RetrieveRole             = "RetrieveRole"
	UpdateRole               = "UpdateRole"
	AddPermissionToRole      = "AddPermissionToRole"
	RemovePermissionFromRole = "RemovePermissionFromRole"
	DeleteRole               = "DeleteRole"
)

func (s *Store) ListRoles(ctx context.Context, page *models.Page) (*models.RoleList, error) {
	s.calls[ListRoles]++
	if s.OnListRoles != nil {
		return s.OnListRoles(ctx, page)
	}
	panic(errors.Fmt("%s callback is not mocked", ListRoles))
}

func (s *Store) CreateRole(ctx context.Context, role *models.Role) error {
	s.calls[CreateRole]++
	if s.OnCreateRole != nil {
		return s.OnCreateRole(ctx, role)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateRole))
}

func (s *Store) RetrieveRole(ctx context.Context, id any) (*models.Role, error) {
	s.calls[RetrieveRole]++
	if s.OnRetrieveRole != nil {
		return s.OnRetrieveRole(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveRole))
}

func (s *Store) UpdateRole(ctx context.Context, role *models.Role) error {
	s.calls[UpdateRole]++
	if s.OnUpdateRole != nil {
		return s.OnUpdateRole(ctx, role)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateRole))
}

func (s *Store) AddPermissionToRole(ctx context.Context, roleID int64, permission any) error {
	s.calls[AddPermissionToRole]++
	if s.OnAddPermissionToRole != nil {
		return s.OnAddPermissionToRole(ctx, roleID, permission)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToRole))
}

func (s *Store) RemovePermissionFromRole(ctx context.Context, roleID int64, permission int64) error {
	s.calls[RemovePermissionFromRole]++
	if s.OnRemovePermissionFromRole != nil {
		return s.OnRemovePermissionFromRole(ctx, roleID, permission)
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
	ListPermissions    = "ListPermissions"
	CreatePermission   = "CreatePermission"
	RetrievePermission = "RetrievePermission"
	UpdatePermission   = "UpdatePermission"
	DeletePermission   = "DeletePermission"
)

func (s *Store) ListPermissions(ctx context.Context, page *models.Page) (*models.PermissionList, error) {
	s.calls[ListPermissions]++
	if s.OnListPermissions != nil {
		return s.OnListPermissions(ctx, page)
	}
	panic(errors.Fmt("%s callback is not mocked", ListPermissions))
}

func (s *Store) CreatePermission(ctx context.Context, permission *models.Permission) error {
	s.calls[CreatePermission]++
	if s.OnCreatePermission != nil {
		return s.OnCreatePermission(ctx, permission)
	}
	panic(errors.Fmt("%s callback is not mocked", CreatePermission))
}

func (s *Store) RetrievePermission(ctx context.Context, id any) (*models.Permission, error) {
	s.calls[RetrievePermission]++
	if s.OnRetrievePermission != nil {
		return s.OnRetrievePermission(ctx, id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrievePermission))
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
