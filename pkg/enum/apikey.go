package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// APIKeyStatus gives an indication of how the API key is being used to help admins
// better manage API key access to the Endeavor and Quarterdeck systems.
type APIKeyStatus uint8

const (
	APIKeyStatusUnknown APIKeyStatus = iota
	APIKeyStatusUnused
	APIKeyStatusActive
	APIKeyStatusStale
	APIKeyStatusRevoked

	// The terminator is used to determine the last value of the enum. It should be
	// the last value in the list and is automatically incremented when enums are
	// added above it.
	// NOTE: you should not reorder the enums, just append them to the list above
	// to add new values.
	apiKeyStatusTerminator
)

var apiKeyStatusNames = [5]string{
	"unknown", "unused", "active", "stale", "revoked",
}

// Returns true if the provided apikey status is valid (e.g. parseable), false otherwise.
func ValidAPIKeyStatus(s interface{}) bool {
	if status, err := ParseAPIKeyStatus(s); err != nil || status >= apiKeyStatusTerminator {
		return false
	}
	return true
}

// Returns true if the apikey status is equal to one of the target statuses. Any parse
// errors for the apikey status are returned.
func CheckAPIKeyStatus(s interface{}, targets ...APIKeyStatus) (_ bool, err error) {
	var status APIKeyStatus
	if status, err = ParseAPIKeyStatus(s); err != nil {
		return false, err
	}

	for _, target := range targets {
		if status == target {
			return true, nil
		}
	}

	return false, nil
}

// Parse the apikey status from the provided value.
func ParseAPIKeyStatus(s interface{}) (APIKeyStatus, error) {
	switch s := s.(type) {
	case string:
		s = strings.ToLower(s)
		if s == "" {
			return APIKeyStatusUnknown, nil
		}

		for i, name := range apiKeyStatusNames {
			if name == s {
				return APIKeyStatus(i), nil
			}
		}
		return APIKeyStatusUnknown, fmt.Errorf("invalid apikey status: %q", s)
	case uint8:
		return APIKeyStatus(s), nil
	case APIKeyStatus:
		return s, nil
	default:
		return APIKeyStatusUnknown, fmt.Errorf("cannot parse %T into an apikey status", s)
	}
}

// Return a string representation of the apikey status.
func (s APIKeyStatus) String() string {
	if s >= apiKeyStatusTerminator {
		return apiKeyStatusNames[0]
	}
	return apiKeyStatusNames[s]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (s APIKeyStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *APIKeyStatus) UnmarshalJSON(b []byte) (err error) {
	var src string
	if err = json.Unmarshal(b, &src); err != nil {
		return err
	}
	if *s, err = ParseAPIKeyStatus(src); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (s *APIKeyStatus) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*s, err = ParseAPIKeyStatus(x)
		return err
	case []byte:
		*s, err = ParseAPIKeyStatus(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into an apikey status", src)
}

func (s APIKeyStatus) Value() (driver.Value, error) {
	return s.String(), nil
}
