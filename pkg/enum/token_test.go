package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

func TestValidTokenTypes(t *testing.T) {
	tests := []struct {
		input  interface{}
		assert require.BoolAssertionFunc
	}{
		{"", require.True},
		{"unknown", require.True},
		{"reset_password", require.True},
		{"verify_email", require.True},
		{"team_invite", require.True},
		{uint8(0), require.True},
		{uint8(1), require.True},
		{uint8(2), require.True},
		{uint8(3), require.True},
		{enum.TokenTypeUnknown, require.True},
		{enum.TokenTypeResetPassword, require.True},
		{enum.TokenTypeVerifyEmail, require.True},
		{enum.TokenTypeTeamInvite, require.True},
		{"foo", require.False},
		{true, require.False},
		{uint8(99), require.False},
	}

	for i, tc := range tests {
		tc.assert(t, enum.ValidTokenType(tc.input), "test case %d failed", i)
	}
}

func TestCheckTokenType(t *testing.T) {
	tests := []struct {
		input   interface{}
		targets []enum.TokenType
		assert  require.BoolAssertionFunc
		err     error
	}{
		{"", []enum.TokenType{enum.TokenTypeUnknown, enum.TokenTypeResetPassword, enum.TokenTypeTeamInvite}, require.True, nil},
		{"unknown", []enum.TokenType{enum.TokenTypeResetPassword, enum.TokenTypeVerifyEmail, enum.TokenTypeUnknown}, require.True, nil},
		{"reset_password", []enum.TokenType{enum.TokenTypeTeamInvite, enum.TokenTypeVerifyEmail}, require.False, nil},
		{"foo", []enum.TokenType{enum.TokenTypeResetPassword, enum.TokenTypeTeamInvite}, require.False, errors.New(`invalid token type: "foo"`)},
		{"", []enum.TokenType{enum.TokenTypeResetPassword, enum.TokenTypeTeamInvite}, require.False, nil},
		{"unknown", []enum.TokenType{enum.TokenTypeTeamInvite, enum.TokenTypeResetPassword}, require.False, nil},
		{"verify_email", []enum.TokenType{enum.TokenTypeResetPassword, enum.TokenTypeTeamInvite}, require.False, nil},
	}

	for i, tc := range tests {
		result, err := enum.CheckTokenType(tc.input, tc.targets...)
		tc.assert(t, result, "test case %d failed", i)

		if tc.err != nil {
			require.Equal(t, tc.err, err, "test case %d failed", i)
		} else {
			require.NoError(t, err, "test case %d failed", i)
		}
	}
}

func TestParseTokenType(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.TokenType
		}{
			{"", enum.TokenTypeUnknown},
			{"unknown", enum.TokenTypeUnknown},
			{"reset_password", enum.TokenTypeResetPassword},
			{"verify_email", enum.TokenTypeVerifyEmail},
			{"team_invite", enum.TokenTypeTeamInvite},
			{uint8(0), enum.TokenTypeUnknown},
			{uint8(1), enum.TokenTypeResetPassword},
			{uint8(2), enum.TokenTypeVerifyEmail},
			{uint8(3), enum.TokenTypeTeamInvite},
			{enum.TokenTypeUnknown, enum.TokenTypeUnknown},
			{enum.TokenTypeResetPassword, enum.TokenTypeResetPassword},
			{enum.TokenTypeVerifyEmail, enum.TokenTypeVerifyEmail},
			{enum.TokenTypeTeamInvite, enum.TokenTypeTeamInvite},
		}

		for i, test := range tests {
			result, err := enum.ParseTokenType(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"foo", "invalid token type: \"foo\""},
			{true, "cannot parse bool into a token type"},
		}

		for i, test := range tests {
			result, err := enum.ParseTokenType(test.input)
			require.Equal(t, enum.TokenTypeUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		tt       enum.TokenType
		expected string
	}{
		{enum.TokenTypeUnknown, "unknown"},
		{enum.TokenTypeResetPassword, "reset_password"},
		{enum.TokenTypeVerifyEmail, "verify_email"},
		{enum.TokenTypeTeamInvite, "team_invite"},
		{enum.TokenType(99), "unknown"},
	}

	for i, test := range tests {
		result := test.tt.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestTokenTypeJSON(t *testing.T) {
	tests := []enum.TokenType{
		enum.TokenTypeUnknown, enum.TokenTypeResetPassword,
		enum.TokenTypeVerifyEmail, enum.TokenTypeTeamInvite,
	}

	for _, tt := range tests {
		data, err := json.Marshal(tt)
		require.NoError(t, err)

		var result enum.TokenType
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, tt, result)
	}
}

func TestTokenTypeScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.TokenType
	}{
		{nil, enum.TokenTypeUnknown},
		{"", enum.TokenTypeUnknown},
		{"unknown", enum.TokenTypeUnknown},
		{"reset_password", enum.TokenTypeResetPassword},
		{"verify_email", enum.TokenTypeVerifyEmail},
		{"team_invite", enum.TokenTypeTeamInvite},
		{[]byte(""), enum.TokenTypeUnknown},
		{[]byte("unknown"), enum.TokenTypeUnknown},
		{[]byte("reset_password"), enum.TokenTypeResetPassword},
		{[]byte("verify_email"), enum.TokenTypeVerifyEmail},
		{[]byte("team_invite"), enum.TokenTypeTeamInvite},
	}

	for i, test := range tests {
		var tt enum.TokenType
		err := tt.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, tt, "test case %d failed", i)
	}

	var d enum.TokenType
	err := d.Scan("foo")
	require.EqualError(t, err, "invalid token type: \"foo\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into a token type")
}

func TestTokenTypeValue(t *testing.T) {
	value, err := enum.TokenTypeVerifyEmail.Value()
	require.NoError(t, err)
	require.Equal(t, "verify_email", value)
}
