package models_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/quarterdeck/pkg/store/models"
)

func TestRoles(t *testing.T) {
	t.Run("List", func(t *testing.T) {
		testCases := []struct {
			roles    Roles
			expected []string
		}{
			{
				roles:    nil,
				expected: nil,
			},
			{
				roles:    Roles{},
				expected: []string{},
			},
			{
				roles: Roles{
					{Title: "role1"},
					{Title: "role2"},
				},
				expected: []string{"role1", "role2"},
			},
		}

		for _, tc := range testCases {
			require.Equal(t, tc.expected, tc.roles.List())
		}
	})

	t.Run("Load", func(t *testing.T) {
		t.Run("Nil", func(t *testing.T) {
			var roles Roles
			roles.Load(nil)
			require.Nil(t, roles)
		})

		t.Run("Empty", func(t *testing.T) {
			var roles Roles
			roles.Load([]string{})
			require.Nil(t, roles)

		})
	})
}
