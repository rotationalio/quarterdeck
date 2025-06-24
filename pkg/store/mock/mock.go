package mock

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/dsn"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
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
