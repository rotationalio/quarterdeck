package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// TokenType identifies the purpose of the vero token being sent such as password reset,
// email verification, or team invitation. The token type also determines what model
// the resource ID is associated with; e.g. password reset and email verification
// tokens are associated with a User.
type TokenType uint8

const (
	TokenTypeUnknown TokenType = iota
	TokenTypeResetPassword
	TokenTypeVerifyEmail
	TokenTypeTeamInvite

	// The terminator is used to determine the last value of the enum. It should be
	// the last value in the list and is automatically incremented when enums are
	// added above it.
	// NOTE: you should not reorder the enums, just append them to the list above
	// to add new values.
	tokenTypeTerminator
)

var tokenTypeNames = [4]string{
	"unknown", "reset_password", "verify_email", "team_invite",
}

// Returns true if the provided token type is valid (e.g. parseable), false otherwise.
func ValidTokenType(t interface{}) bool {
	if tt, err := ParseTokenType(t); err != nil || tt >= tokenTypeTerminator {
		return false
	}
	return true
}

// Returns true if the token type is equal to one of the target token types. Any parse
// errors for the token type are returned.
func CheckTokenType(t interface{}, targets ...TokenType) (_ bool, err error) {
	var tt TokenType
	if tt, err = ParseTokenType(t); err != nil {
		return false, err
	}

	for _, target := range targets {
		if tt == target {
			return true, nil
		}
	}

	return false, nil
}

// Parse the token type from the provided value.
func ParseTokenType(t interface{}) (TokenType, error) {
	switch t := t.(type) {
	case string:
		t = strings.ToLower(t)
		if t == "" {
			return TokenTypeUnknown, nil
		}

		for i, name := range tokenTypeNames {
			if name == t {
				return TokenType(i), nil
			}
		}
		return TokenTypeUnknown, errors.Fmt("invalid token type: %q", t)
	case uint8:
		return TokenType(t), nil
	case TokenType:
		return t, nil
	default:
		return TokenTypeUnknown, errors.Fmt("cannot parse %T into a token type", t)
	}
}

// Return a string representation of the token type.
func (tt TokenType) String() string {
	if tt >= tokenTypeTerminator {
		return tokenTypeNames[0]
	}
	return tokenTypeNames[tt]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (tt TokenType) MarshalJSON() ([]byte, error) {
	return json.Marshal(tt.String())
}

func (tt *TokenType) UnmarshalJSON(b []byte) (err error) {
	var src string
	if err = json.Unmarshal(b, &src); err != nil {
		return err
	}
	if *tt, err = ParseTokenType(src); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (tt *TokenType) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*tt, err = ParseTokenType(x)
		return err
	case []byte:
		*tt, err = ParseTokenType(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into a token type", src)
}

func (tt TokenType) Value() (driver.Value, error) {
	return tt.String(), nil
}
