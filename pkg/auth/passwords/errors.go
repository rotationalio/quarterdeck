package passwords

import "go.rtnl.ai/quarterdeck/pkg/errors"

// Password Strength Errors
var (
	ErrPasswordEmpty      = errors.ErrPasswordEmpty
	ErrPasswordTooShort   = errors.ErrPasswordTooShort
	ErrPasswordWhitespace = errors.ErrPasswordWhitespace
	ErrPasswordStrength   = errors.ErrPasswordStrength
)
