package backend

import (
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal"

	// cSpell:ignore pgconn pgx jackc
	"github.com/jackc/pgx/v5/pgconn"
)

// domainErrors are store-level errors that must not be wrapped as ErrDatabase.
var domainErrors = []error{
	errors.ErrZeroValuedNotNull,
	errors.ErrNoIDOnCreate,
	errors.ErrMissingID,
	errors.ErrMissingReference,
	errors.ErrTypeMismatch,
	errors.ErrTooSoon,
	errors.ErrNotAuthorized,
	errors.ErrNotFound,
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
		return errors.ErrNotFound
	}
	if errors.Is(err, tidal.ErrMissingID) {
		return errors.ErrMissingID
	}
	if errors.Is(err, tidal.ErrReadOnly) {
		return errors.ErrReadOnly
	}
	if errors.Is(err, tidal.ErrAlreadyExists) {
		return errors.ErrAlreadyExists
	}

	// postgres specific errors that we need to break down
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "25006": // read_only_sql_transaction
			return errors.ErrReadOnly
		case "23505": // unique_violation
			return errors.ErrAlreadyExists
		}
	}

	// default to wrapping (via Join) the error with ErrDatabase
	return errors.Join(errors.ErrDatabase, err)
}
