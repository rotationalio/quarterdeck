package mock

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

type Model struct {
	OnScan     func(operation models.Operation, scanner models.Scanner) error
	OnFields   func(operation models.Operation) []string
	OnParams   func(operation models.Operation) []sql.NamedArg
	OnPrepare  func(operation models.Operation)
	OnValidate func(operation models.Operation) error

	calls map[string]int
}

var (
	_ models.Model     = (*Model)(nil)
	_ models.Preparer  = (*Model)(nil)
	_ models.Validator = (*Model)(nil)
)

func (m *Model) Scan(operation models.Operation, scanner models.Scanner) error {
	m.call("Scan", operation)
	return m.OnScan(operation, scanner)
}

func (m *Model) Fields(operation models.Operation) []string {
	m.call("Fields", operation)
	return m.OnFields(operation)
}

func (m *Model) Params(operation models.Operation) []sql.NamedArg {
	m.call("Params", operation)
	return m.OnParams(operation)
}

func (m *Model) Prepare(operation models.Operation) {
	m.call("Prepare", operation)
	m.OnPrepare(operation)
}

func (m *Model) Validate(operation models.Operation) error {
	m.call("Validate", operation)
	return m.OnValidate(operation)
}

// Assert that the expected number of calls were made to the given method.
func (m *Model) AssertCalls(t testing.TB, method string, operation models.Operation, expected int) {
	method += "(" + operation.String() + ")"
	require.Equal(t, expected, m.calls[method], "expected %d calls to %s, got %d", expected, method, m.calls[method])
}

func (m *Model) call(name string, op models.Operation) {
	if m.calls == nil {
		m.calls = make(map[string]int)
	}
	m.calls[name+"("+op.String()+")"]++
}
