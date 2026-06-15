package mock_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v2/mock"
	"go.rtnl.ai/ulid"
)

//=============================================================================
// Tests
//=============================================================================

// TestScanner verifies the mock Scanner handles conversions, errors, and panics.
func TestScanner(t *testing.T) {
	var (
		tulid      = ulid.MakeSecure()
		tnow       = time.Now()
		tint       = 808
		tint64     = int64(16016)
		tfloat64   = 3.14159
		tstring    = "Mahalo"
		tbytes     = []byte("Makai")
		tbool      = true
		ttimestamp = "2025-01-01T12:34:56.123456-10:00"
	)

	t.Run("ScanTests", func(t *testing.T) {
		// Setup: one value per destination field; time.Duration is unscannable.
		data := []any{
			tulid.String(),
			tnow,
			tint64,
			tfloat64,
			tstring,
			tulid.String(),
			tnow,
			ttimestamp,
			tint,
			tint64,
			tfloat64,
			tstring,
			tbytes,
			tbool,
			tulid[:],
			tnow,
			tstring,
			tint,
			tint64,
			tfloat64,
			tbool,
			nil,
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)
		shouldNotScan := 1
		shouldScan := len(data) - shouldNotScan

		// Action: scan all fields.
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		// Assert: scan succeeds and counts match.
		require.NoError(t, err)
		mockScanner.AssertScanned(t, shouldScan)
		mockScanner.AssertNotScanned(t, shouldNotScan)
	})

	t.Run("SetError", func(t *testing.T) {
		// Setup: scanner configured to return an error.
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(errors.ErrInternal)

		// Action + assert: error propagates unchanged.
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		require.Error(t, err)
		require.Equal(t, errors.ErrInternal, err)
	})

	t.Run("Panics", func(t *testing.T) {
		// Setup: fewer data values than destination fields.
		data := []any{
			tulid.String(),
			tnow,
			tint64,
			tfloat64,
			tstring,
			tulid.String(),
			tnow,
			ttimestamp,
			tint,
			tint64,
			tfloat64,
			tstring,
			tbytes,
			tbool,
			tulid[:],
			tnow,
			tstring,
			tint,
			tint64,
			tfloat64,
			tbool,
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// Assert: missing destination panics.
		model := &MockTestModel{}
		require.Panics(t, func() { _ = model.Scan(mockScanner) })
	})
}

//=============================================================================
// Test Helpers
//=============================================================================

// rowScanner is the minimal interface models use when scanning from a row.
type rowScanner interface {
	Scan(dest ...any) error
}

// MockTestModel exercises every type the mock Scanner can convert.
type MockTestModel struct {
	TestNullULID     ulid.NullULID
	TestNullTime     sql.NullTime
	TestNullInt64    sql.NullInt64
	TestNullFloat64  sql.NullFloat64
	TestNullString   sql.NullString
	TestULID         ulid.ULID
	TestTime         time.Time
	TestStringToTime string
	TestInt          int
	TestInt64        int64
	TestFloat64      float64
	TestString       string
	TestBytes        []byte
	TestBool         bool
	TestPtrULID      *ulid.ULID
	TestPtrTime      *time.Time
	TestPtrString    *string
	TestPtrInt       *int
	TestPtrInt64     *int64
	TestPtrFloat64   *float64
	TestPtrBool      *bool
	TestNoScan       time.Duration
}

func (m *MockTestModel) Scan(scanner rowScanner) error {
	return scanner.Scan(
		&m.TestNullULID,
		&m.TestNullTime,
		&m.TestNullInt64,
		&m.TestNullFloat64,
		&m.TestNullString,
		&m.TestULID,
		&m.TestTime,
		&m.TestStringToTime,
		&m.TestInt,
		&m.TestInt64,
		&m.TestFloat64,
		&m.TestString,
		&m.TestBytes,
		&m.TestBool,
		&m.TestPtrULID,
		&m.TestPtrTime,
		&m.TestPtrString,
		&m.TestPtrInt,
		&m.TestPtrInt64,
		&m.TestPtrFloat64,
		&m.TestPtrBool,
		&m.TestNoScan,
	)
}
