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

	OnClose func() error
	OnBegin func(context.Context, *sql.TxOptions) (txn.Txn, error)

	OnListUsers       func(context.Context, *models.UserPage) (*models.UserList, error)
	OnCreateUser      func(context.Context, *models.User) error
	OnRetrieveUser    func(context.Context, any) (*models.User, error)
	OnUpdateUser      func(context.Context, *models.User) error
	OnUpdatePassword  func(context.Context, ulid.ULID, string) error
	OnUpdateLastLogin func(context.Context, ulid.ULID, time.Time) error
	OnDeleteUser      func(context.Context, ulid.ULID) error
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
