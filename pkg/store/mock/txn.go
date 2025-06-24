package mock

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// Transaction method names
const (
	Commit   = "Commit"
	Rollback = "Rollback"
)

type Tx struct {
	opts     *sql.TxOptions
	calls    map[string]int
	commit   bool
	rollback bool

	OnCommit   func() error
	OnRollback func() error
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the store.
func (tx *Tx) Reset() {
	// reset the call counts
	tx.calls = nil
	tx.calls = make(map[string]int)

	// reset the callbacks using reflection
	v := reflect.ValueOf(tx).Elem()
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
func (tx *Tx) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, tx.calls[method], "expected %d calls to %s, got %d", expected, method, tx.calls[method])
}

// Assert that Commit has been called on the transaction without rollback.
func (tx *Tx) AssertCommit(t testing.TB) {
	require.True(t, tx.commit && !tx.rollback, "expected Commit to be called but not Rollback")
}

// Assert that Rollback has been called on the transaction without commit.
func (tx *Tx) AssertRollback(t testing.TB) {
	require.True(t, tx.rollback && !tx.commit, "expected Rollback to be called but not Commit")
}

// Assert that Commit has not been called on the transaction.
func (tx *Tx) AssertNoCommit(t testing.TB) {
	require.False(t, tx.commit, "did not expect Commit to be called")
}

// Assert that Rollback has not been called on the transaction.
func (tx *Tx) AssertNoRollback(t testing.TB) {
	require.False(t, tx.rollback, "did not expect Rollback to be called")
}

// Check is a helper method that determines if the transaction is committed or rolled
// back. If so it returns ErrTxDone no matter the callback. Additionally, if the method
// is writeable and the transaction is read-only, it returns an error. This method also
// increments the call count for the method.
func (tx *Tx) check(method string, writeable bool) error {
	tx.calls[method]++

	if tx.commit || tx.rollback {
		return sql.ErrTxDone
	}

	if tx.opts != nil && tx.opts.ReadOnly && writeable {
		return errors.ErrReadOnly
	}

	return nil
}

//===========================================================================
// Transaction Base Methods
//===========================================================================

func (tx *Tx) Commit() (err error) {
	if err = tx.check(Commit, false); err != nil {
		return err
	}

	if tx.OnCommit != nil {
		err = tx.OnCommit()
	}

	tx.commit = true
	return err
}

func (tx *Tx) Rollback() (err error) {
	if err := tx.check(Rollback, false); err != nil {
		return err
	}

	if tx.OnRollback != nil {
		err = tx.OnRollback()
	}

	tx.rollback = true
	return err
}
