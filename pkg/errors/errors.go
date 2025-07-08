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
	ErrZeroValuedNotNull  = errors.New("query contains a not null field with a zero valued parameter")

	// Server related errors
	ErrNotAccepted = errors.New("the accepted formats are not offered by the server")
	ErrNotAllowed  = errors.New("the requested action is not allowed")
	ErrInternal    = errors.New("something critical went wrong, please try again later")

	// Authentication errors
	ErrUnknownSigningKey = errors.New("unknown signing key")
	ErrNoKeyID           = errors.New("token does not have kid in header")
	ErrInvalidKeyID      = errors.New("invalid key id")
	ErrUnparsableClaims  = errors.New("could not parse or verify claims")
	ErrUnauthenticated   = errors.New("request is unauthenticated")
	ErrNoClaims          = errors.New("no claims found on the request context")
	ErrNoUserInfo        = errors.New("no user info found on the request context")
	ErrInvalidAuthToken  = errors.New("invalid authorization token")
	ErrAuthRequired      = errors.New("this endpoint requires authentication")
	ErrNotAuthorized     = errors.New("user does not have permission to perform this operation")
	ErrNoAuthUser        = errors.New("could not identify authenticated user in request")
	ErrParseBearer       = errors.New("could not parse Bearer token from Authorization header")
	ErrNoAuthorization   = errors.New("no authorization header or cookies in request")
	ErrNoRefreshToken    = errors.New("cannot reauthenticate no refresh token in request")
	ErrNoSigningKeys     = errors.New("claims issuer has no signing keys configured")
)

// Reduce namespacing conflicts by adding error functions from the errors package.
var (
	New = errors.New
	Fmt = fmt.Errorf
	Is  = errors.Is
	As  = errors.As
)
