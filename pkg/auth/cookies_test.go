package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/auth"
)

func TestIsLocalhost(t *testing.T) {
	testCases := []struct {
		domain string
		assert require.BoolAssertionFunc
	}{
		{
			"localhost",
			require.True,
		},
		{
			"endeavor.local",
			require.True,
		},
		{
			"honu.local",
			require.True,
		},
		{
			"quarterdeck",
			require.False,
		},
		{
			"rotational.app",
			require.False,
		},
		{
			"auth.rotational.app",
			require.False,
		},
		{
			"quarterdeck.local.example.io",
			require.False,
		},
	}

	for i, tc := range testCases {
		tc.assert(t, auth.IsLocalhost(tc.domain), "test case %d failed", i)
	}
}
