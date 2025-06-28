package models_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var ErrModelScan = errors.New("test scan error")

func CheckParams(t testing.TB, actual []any, expectedNames []string, expectedValues []any) {
	t.Helper()
	for i, param := range actual {
		arg, ok := param.(sql.NamedArg)
		require.True(t, ok, "param %d is not a named arg", i)
		require.Equal(t, expectedNames[i], arg.Name, "param %d name mismatch", i)
		require.Equal(t, expectedValues[i], arg.Value, "param %d value mismatch", i)
	}
}
