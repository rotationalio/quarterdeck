package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/quarterdeck/pkg/auth"
)

func TestSubjectString(t *testing.T) {
	tests := []struct {
		subjectType SubjectType
		expected    string
	}{
		{SubjectUser, "user"},
		{SubjectAPIKey, "apikey"},
		{SubjectVero, "vero"},
		{SubjectType('!'), "unknown"},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, test.subjectType.String(), "expected subject type %q to be %q", test.subjectType, test.expected)
	}
}
