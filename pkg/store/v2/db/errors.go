package db

import (
	"database/sql"
	"errors"

	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/mattn/go-sqlite3"
)

// domainErrors are store-level errors that must not be wrapped as ErrDatabase.
var domainErrors = []error{
	qerrors.ErrZeroValuedNotNull,
	qerrors.ErrNoIDOnCreate,
	qerrors.ErrMissingID,
	qerrors.ErrMissingReference,
	qerrors.ErrTypeMismatch,
	qerrors.ErrTooSoon,
	qerrors.ErrNotAuthorized,
	qerrors.ErrNotFound,
}

func isDomainErr(err error) bool {
	for _, domain := range domainErrors {
		if errors.Is(err, domain) {
			return true
		}
	}
	return false
}

// Returns a Quarterdeck error for a Tidal error.
func tidalErr(err error) error {
	if err == nil {
		return nil
	}

	// already a quarterdeck domain error
	if isDomainErr(err) {
		return err
	}

	// sql/tidal errors that we need to break down
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, tidal.ErrNotFound) {
		return qerrors.ErrNotFound
	}
	if errors.Is(err, tidal.ErrMissingID) {
		return qerrors.ErrMissingID
	}

	// sqlite specific errors that we need to break down
	var sqliteErr sqlite3.Error
	if errors.As(err, &sqliteErr) {
		if errors.Is(sqliteErr.Code, sqlite3.ErrReadonly) {
			return qerrors.ErrReadOnly
		}

		if errors.Is(sqliteErr.Code, sqlite3.ErrConstraint) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return qerrors.ErrAlreadyExists
		}
	}

	// postgres specific errors that we need to break down
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pqerror.ReadOnlySQLTransaction:
			return qerrors.ErrReadOnly
		case pqerror.UniqueViolation:
			return qerrors.ErrAlreadyExists
		}
	}

	// default to wrapping (via Join) the error with ErrDatabase
	return qerrors.Join(qerrors.ErrDatabase, err)
}
