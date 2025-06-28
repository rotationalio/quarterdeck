package mock_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// A model to test MockScanner with.
type MockTestModel struct {

	// null types with Scanner
	TestNullULID    ulid.NullULID
	TestNullTime    sql.NullTime
	TestNullInt64   sql.NullInt64
	TestNullFloat64 sql.NullFloat64
	TestNullString  sql.NullString

	// base types without Scanner covered in `convertAssign`
	TestULID         ulid.ULID
	TestTime         time.Time
	TestStringToTime string
	TestInt          int
	TestInt64        int64
	TestFloat64      float64
	TestString       string
	TestBytes        []byte
	TestBool         bool

	// pointers to base types
	TestPtrULID    *ulid.ULID
	TestPtrTime    *time.Time
	TestPtrString  *string
	TestPtrInt     *int
	TestPtrInt64   *int64
	TestPtrFloat64 *float64
	TestPtrBool    *bool

	// "no scan" test (`convertAssign()` should fail for `time.Duration`)
	TestNoScan time.Duration
}

// Scans a MockTestModel.
func (m *MockTestModel) Scan(scanner models.Scanner) error {
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
		// setup
		data := []any{
			tulid.String(), // i = 0  (TestNullULID)
			tnow,           // i = 1  (TestNullTime)
			tint64,         // i = 2  (TestNullInt64)
			tfloat64,       // i = 3  (TestNullFloat64)
			tstring,        // i = 4  (TestNullString)
			tulid.String(), // i = 5  (TestULID)
			tnow,           // i = 6  (TestTime)
			ttimestamp,     // i = 7  (TestStringToTime)
			tint,           // i = 8  (TestInt)
			tint64,         // i = 9  (TestInt64)
			tfloat64,       // i = 10 (TestFloat64)
			tstring,        // i = 11 (TestString)
			tbytes,         // i = 12 (TestBytes)
			tbool,          // i = 13 (TestBool)
			tulid[:],       // i = 14 (TestPtrULID)
			tnow,           // i = 15 (TestPtrTime)
			tstring,        // i = 16 (TestPtrString)
			tint,           // i = 17 (TestPtrInt)
			tint64,         // i = 18 (TestPtrInt64)
			tfloat64,       // i = 19 (TestPtrFloat64)
			tbool,          // i = 20 (TestPtrBool)
			nil,            // i = 21 (TestNoScan)
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)
		shouldNotScan := 1 // "TestNoScan" type shouldn't scan properly in `convertAssign`
		shouldScan := len(data) - shouldNotScan

		// test
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, shouldScan)
		mockScanner.AssertNotScanned(t, shouldNotScan)
	})

	t.Run("SetError", func(t *testing.T) {
		//setup
		mockScanner := &mock.Scanner{}
		mockScanner.SetError(errors.ErrInternal)

		// test
		model := &MockTestModel{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.Equal(t, errors.ErrInternal, err, "expected errors.ErrInternal from the scanner")

	})

	t.Run("Panics", func(t *testing.T) {
		// setup
		data := []any{
			tulid.String(), // i = 0  (TestNullULID)
			tnow,           // i = 1  (TestNullTime)
			tint64,         // i = 2  (TestNullInt64)
			tfloat64,       // i = 3  (TestNullFloat64)
			tstring,        // i = 4  (TestNullString)
			tulid.String(), // i = 5  (TestULID)
			tnow,           // i = 6  (TestTime)
			ttimestamp,     // i = 7  (TestStringToTime)
			tint,           // i = 8  (TestInt)
			tint64,         // i = 9  (TestInt64)
			tfloat64,       // i = 10 (TestFloat64)
			tstring,        // i = 11 (TestString)
			tbytes,         // i = 12 (TestBytes)
			tbool,          // i = 13 (TestBool)
			tulid[:],       // i = 14 (TestPtrULID)
			tnow,           // i = 15 (TestPtrTime)
			tstring,        // i = 16 (TestPtrString)
			tint,           // i = 17 (TestPtrInt)
			tint64,         // i = 18 (TestPtrInt64)
			tfloat64,       // i = 19 (TestPtrFloat64)
			tbool,          // i = 20 (TestPtrBool)

			// CAUSES A PANIC BY COMMENTING OUT LAST ITEM
			// nil,                                // i = 21 (TestNoScan)
		}
		mockScanner := &mock.Scanner{}
		mockScanner.SetData(data)

		// test
		model := &MockTestModel{}
		require.Panics(t, func() { _ = model.Scan(mockScanner) }, "should panic, not enough data items")
	})
}
