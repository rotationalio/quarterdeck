package errors

import (
	"errors"
	"fmt"
)

var (
	// Database DSN errors
	ErrDSNParse      = errors.New("could not parse dsn")
	ErrInvalidDSN    = errors.New("could not parse DSN, critical component missing")
	ErrPathRequired  = errors.New("a path is required for this database scheme")
	ErrUnknownScheme = errors.New("database scheme not handled by this package")

	// Database errors
	ErrDatabase          = errors.New("database error")
	ErrSQLiteForeignKeys = errors.New("could not enable sqlite foreign keys")
	ErrSQLiteQueryOnly   = errors.New("could not set sqlite to query_only mode")

	// Database constraint errors
	ErrAlreadyExists      = errors.New("record already exists in database")
	ErrAmbiguous          = errors.New("ambiguous query: more than one result returned")
	ErrIDMismatch         = errors.New("resource id does not match target")
	ErrMissingAssociation = errors.New("associated record(s) not cached on model")
	ErrMissingID          = errors.New("id required for this resource")
	ErrMissingReference   = errors.New("missing id of foreign key reference")
	ErrNoIDOnCreate       = errors.New("cannot create a resource with an id")
	ErrNotFound           = errors.New("record not found")
	ErrNotImplemented     = errors.New("method not implemented")
	ErrReadOnly           = errors.New("cannot perform operation in read-only mode")
	ErrTooSoon            = errors.New("a previous record has not expired yet")
	ErrTypeMismatch       = errors.New("record type does not match target")
	ErrZeroValuedNotNull  = errors.New("query contains a not null field with a zero valued parameter")

	// Mock scanner errors
	ErrNilScanDestination = errors.New("destination pointer is nil")
	ErrUnknownScanType    = errors.New("unknown type for conversion: reflection required")

	// Server related errors
	ErrCouldNotCompleteResetPasswordRequest = errors.New("could not complete reset password request")
	ErrCouldNotCreateUser                   = errors.New("could not create user")
	ErrCouldNotParseResetPasswordRequest    = errors.New("could not parse reset password request")
	ErrCouldNotUpsertExistingUser           = errors.New("could not upsert existing user")
	ErrCSRFRequestRejected                  = errors.New("CSRF request rejected")
	ErrInternal                             = errors.New("could not complete request due to an internal error: admins have been notified")
	ErrInvalidOrExpiredResetPasswordLink    = errors.New("reset password link is invalid or expired")
	ErrInvalidResetPasswordRequest          = errors.New("reset password request is invalid")
	ErrNotAccepted                          = errors.New("the accepted formats are not offered by the server")
	ErrNotAllowed                           = errors.New("the requested action is not allowed")
	ErrRejectedResetPasswordLink            = errors.New("reset password link was rejected")
	ErrResetPasswordCookieMissing           = errors.New("reset password token cookie is missing")

	// Parsing errors
	ErrBindJSON = errors.New("could not parse application/json request body")

	// Authentication errors
	ErrAuthRequired              = errors.New("this endpoint requires authentication")
	ErrEmailNotVerified          = errors.New("please check your email to verify your account before logging in")
	ErrEmptyDerivedKeyOrPassword = errors.New("cannot verify empty derived key or password")
	ErrEmptyDerivedKeyPassword   = errors.New("cannot create derived key for empty password")
	ErrExpiredToken              = errors.New("verification token is expired")
	ErrFailedAuthentication      = errors.New("login failed, please check your credentials")
	ErrInvalidAuthToken          = errors.New("invalid authorization token")
	ErrInvalidKeyID              = errors.New("invalid key id")
	ErrLoginEmailNotVerified     = errors.New("login failed: email is not verified")
	ErrLoginFailed               = errors.New("login failed")
	ErrMalformedDerivedKey       = errors.New("cannot parse encoded derived key, does not match regular expression")
	ErrMalformedDerivedKeyGroups = errors.New("cannot parse encoded derived key, matched expression does not contain enough subgroups")
	ErrNoAuthorization           = errors.New("no authorization header or cookies in request")
	ErrNoAuthUser                = errors.New("could not identify authenticated user in request")
	ErrNoClaims                  = errors.New("no claims found on the request context")
	ErrNoKeyID                   = errors.New("token does not have kid in header")
	ErrNoLoginURL                = errors.New("no login URL configured to redirect the user to")
	ErrNoRefreshToken            = errors.New("cannot reauthenticate: no refresh token in request")
	ErrNoSigningKeys             = errors.New("claims issuer has no signing keys configured")
	ErrNotAuthorized             = errors.New("user does not have permission to perform this operation")
	ErrNoUserInfo                = errors.New("no user info found on the request context")
	ErrParseBearer               = errors.New("could not parse Bearer token from Authorization header")
	ErrPasswordEmpty             = errors.New("password must not be an empty string")
	ErrPasswordStrength          = errors.New("password must contain uppercase letters, lowercase letters, numbers, and special characters")
	ErrPasswordTooShort          = errors.New("password must be at least 8 characters")
	ErrPasswordWhitespace        = errors.New("password must not start or end with whitespace")
	ErrUnauthenticated           = errors.New("request is unauthenticated")
	ErrUnknownSigningKey         = errors.New("unknown signing key")
	ErrUnparsableClaims          = errors.New("could not parse or verify claims")

	// Cache errors
	ErrComputeJWKSCacheETag = errors.New("cannot compute etag on JWKS object only edit keys")
	ErrSetJWKSCacheETag     = errors.New("cannot set etag on JWKS object only edit keys")
	ErrSetJWKSMaxAge        = errors.New("cannot set max age on JWKS object only edit keys")
	ErrSetJWKSModifiedTime  = errors.New("cannot set modified timestamps on JWKS object only edit keys")
	ErrSetJWKSSharedMaxAge  = errors.New("cannot set s-max-age on JWKS object only edit keys")

	// API payload errors
	ErrInvalidRejection     = errors.New("envelope does not contain a rejection/repair error")
	ErrInvalidTimestamp     = errors.New("payload timestamp has invalid string format")
	ErrParsingIVMS101Person = errors.New("unable to parse IVMS101 person record as json or protocol buffers")
	ErrUnknownEncoding      = errors.New("could not detect data encoding please specify and try again")

	// Email errors
	ErrEmptyWelcomeEmailBody = errors.New("welcome email body text or html is empty")
)

// UnhandledProvider is returned when Open or LoadMigrations is called with an
// unsupported database provider.
type UnhandledProvider string

func (e UnhandledProvider) Error() string {
	return fmt.Sprintf("unhandled database provider %q", string(e))
}

// Reduce namespacing conflicts by adding error functions from the errors package.
var (
	New  = errors.New
	Fmt  = fmt.Errorf
	Is   = errors.Is
	As   = errors.As
	Join = errors.Join
)
