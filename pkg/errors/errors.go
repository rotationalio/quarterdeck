package errors

import (
	"errors"
	"fmt"
)

var (
	// Database DSN errors
	ErrDSNParse      = errors.New("could not parse dsn")
	ErrInvalidDSN    = errors.New("could not parse DSN, critical component missing")
	ErrUnknownScheme = errors.New("database scheme not handled by this package")
	ErrPathRequired  = errors.New("a path is required for this database scheme")

	// Database constraint errors
	ErrReadOnly           = errors.New("cannot perform operation in read-only mode")
	ErrMissingAssociation = errors.New("associated record(s) not cached on model")
	ErrMissingReference   = errors.New("missing id of foreign key reference")
	ErrNotFound           = errors.New("record not found")
	ErrAlreadyExists      = errors.New("record already exists in database")
	ErrTooSoon            = errors.New("a previous record has not expired yet")
	ErrNotImplemented     = errors.New("method not implemented")
	ErrNoIDOnCreate       = errors.New("cannot create a resource with an id")
	ErrMissingID          = errors.New("id required for this resource")
	ErrIDMismatch         = errors.New("resource id does not match target")
	ErrAmbiguous          = errors.New("ambiguous query: more than one result returned")

	// Server related errors
	ErrNotAccepted = errors.New("the accepted formats are not offered by the server")
	ErrNotAllowed  = errors.New("the requested action is not allowed")
	ErrInternal    = errors.New("something critical went wrong, please try again later")
)

// Reduce namespacing conflicts by adding error functions from the errors package.
var (
	New = errors.New
	Fmt = fmt.Errorf
	Is  = errors.Is
)
