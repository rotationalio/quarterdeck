package passwords_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/auth/passwords"
)

func TestAPIKeyGeneration(t *testing.T) {
	// These tests indicate our requirements for API key generation rather than testing
	// exactly what the key generation functions return. The requirements are expressed
	// as a regular expression that the client ID and secret must match.
	t.Run("ClientID", func(t *testing.T) {
		// Client IDs must be only be ASCII alpha characters between 16 and
		// 32 characters long. No special characters are allowed.
		requirements := regexp.MustCompile(`^[a-zA-Z]{16,32}$`)
		for i := 0; i < 128; i++ {
			require.Regexp(t, requirements, passwords.ClientID(), "a randomly generated client ID did not meet specifications")
		}
	})

	t.Run("ClientSecret", func(t *testing.T) {
		// Client Secrets must be only be ASCII alphanumeric characters between 32
		// and 64 characters long. No special characters are allowed.
		requirements := regexp.MustCompile(`^[a-zA-Z0-9]{32,64}$`)
		for i := 0; i < 128; i++ {
			require.Regexp(t, requirements, passwords.ClientSecret(), "a randomly generated client secret did not meet specifications")
		}
	})
}
