package enum_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

func TestValidAPIKeyStatuss(t *testing.T) {
	tests := []struct {
		input  interface{}
		assert require.BoolAssertionFunc
	}{
		{"", require.True},
		{"unknown", require.True},
		{"unused", require.True},
		{"active", require.True},
		{"stale", require.True},
		{"revoked", require.True},
		{uint8(0), require.True},
		{uint8(1), require.True},
		{uint8(2), require.True},
		{uint8(3), require.True},
		{uint8(4), require.True},
		{enum.APIKeyStatusUnknown, require.True},
		{enum.APIKeyStatusUnused, require.True},
		{enum.APIKeyStatusActive, require.True},
		{enum.APIKeyStatusStale, require.True},
		{enum.APIKeyStatusRevoked, require.True},
		{"foo", require.False},
		{true, require.False},
		{uint8(99), require.False},
	}

	for i, tc := range tests {
		tc.assert(t, enum.ValidAPIKeyStatus(tc.input), "test case %d failed", i)
	}
}

func TestCheckAPIKeyStatus(t *testing.T) {
	tests := []struct {
		input   interface{}
		targets []enum.APIKeyStatus
		assert  require.BoolAssertionFunc
		err     error
	}{
		{"", []enum.APIKeyStatus{enum.APIKeyStatusUnknown, enum.APIKeyStatusUnused, enum.APIKeyStatusStale}, require.True, nil},
		{"unknown", []enum.APIKeyStatus{enum.APIKeyStatusUnused, enum.APIKeyStatusActive, enum.APIKeyStatusUnknown}, require.True, nil},
		{"revoked", []enum.APIKeyStatus{enum.APIKeyStatusStale, enum.APIKeyStatusActive}, require.False, nil},
		{"foo", []enum.APIKeyStatus{enum.APIKeyStatusUnused, enum.APIKeyStatusStale}, require.False, errors.New(`invalid apikey status: "foo"`)},
		{"", []enum.APIKeyStatus{enum.APIKeyStatusUnused, enum.APIKeyStatusStale}, require.False, nil},
		{"unknown", []enum.APIKeyStatus{enum.APIKeyStatusStale, enum.APIKeyStatusUnused}, require.False, nil},
		{"active", []enum.APIKeyStatus{enum.APIKeyStatusUnused, enum.APIKeyStatusStale}, require.False, nil},
	}

	for i, tc := range tests {
		result, err := enum.CheckAPIKeyStatus(tc.input, tc.targets...)
		tc.assert(t, result, "test case %d failed", i)

		if tc.err != nil {
			require.Equal(t, tc.err, err, "test case %d failed", i)
		} else {
			require.NoError(t, err, "test case %d failed", i)
		}
	}
}

func TestParseAPIKeyStatus(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    interface{}
			expected enum.APIKeyStatus
		}{
			{"", enum.APIKeyStatusUnknown},
			{"unknown", enum.APIKeyStatusUnknown},
			{"unused", enum.APIKeyStatusUnused},
			{"active", enum.APIKeyStatusActive},
			{"stale", enum.APIKeyStatusStale},
			{"revoked", enum.APIKeyStatusRevoked},
			{uint8(0), enum.APIKeyStatusUnknown},
			{uint8(1), enum.APIKeyStatusUnused},
			{uint8(2), enum.APIKeyStatusActive},
			{uint8(3), enum.APIKeyStatusStale},
			{uint8(4), enum.APIKeyStatusRevoked},
			{enum.APIKeyStatusUnknown, enum.APIKeyStatusUnknown},
			{enum.APIKeyStatusUnused, enum.APIKeyStatusUnused},
			{enum.APIKeyStatusActive, enum.APIKeyStatusActive},
			{enum.APIKeyStatusStale, enum.APIKeyStatusStale},
			{enum.APIKeyStatusRevoked, enum.APIKeyStatusRevoked},
		}

		for i, test := range tests {
			result, err := enum.ParseAPIKeyStatus(test.input)
			require.NoError(t, err, "test case %d failed", i)
			require.Equal(t, test.expected, result, "test case %d failed", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		tests := []struct {
			input interface{}
			errs  string
		}{
			{"foo", "invalid apikey status: \"foo\""},
			{true, "cannot parse bool into an apikey status"},
		}

		for i, test := range tests {
			result, err := enum.ParseAPIKeyStatus(test.input)
			require.Equal(t, enum.APIKeyStatusUnknown, result, "test case %d failed", i)
			require.EqualError(t, err, test.errs, "test case %d failed", i)
		}
	})
}

func TestAPIKeyStatusString(t *testing.T) {
	tests := []struct {
		tt       enum.APIKeyStatus
		expected string
	}{
		{enum.APIKeyStatusUnknown, "unknown"},
		{enum.APIKeyStatusUnused, "unused"},
		{enum.APIKeyStatusActive, "active"},
		{enum.APIKeyStatusStale, "stale"},
		{enum.APIKeyStatusRevoked, "revoked"},
		{enum.APIKeyStatus(99), "unknown"},
	}

	for i, test := range tests {
		result := test.tt.String()
		require.Equal(t, test.expected, result, "test case %d failed", i)
	}
}

func TestAPIKeyStatusJSON(t *testing.T) {
	tests := []enum.APIKeyStatus{
		enum.APIKeyStatusUnknown, enum.APIKeyStatusUnused,
		enum.APIKeyStatusActive, enum.APIKeyStatusStale,
		enum.APIKeyStatusRevoked,
	}

	for _, tt := range tests {
		data, err := json.Marshal(tt)
		require.NoError(t, err)

		var result enum.APIKeyStatus
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, tt, result)
	}
}

func TestAPIKeyStatusScan(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected enum.APIKeyStatus
	}{
		{nil, enum.APIKeyStatusUnknown},
		{"", enum.APIKeyStatusUnknown},
		{"unknown", enum.APIKeyStatusUnknown},
		{"unused", enum.APIKeyStatusUnused},
		{"active", enum.APIKeyStatusActive},
		{"stale", enum.APIKeyStatusStale},
		{"revoked", enum.APIKeyStatusRevoked},
		{[]byte(""), enum.APIKeyStatusUnknown},
		{[]byte("unknown"), enum.APIKeyStatusUnknown},
		{[]byte("unused"), enum.APIKeyStatusUnused},
		{[]byte("active"), enum.APIKeyStatusActive},
		{[]byte("stale"), enum.APIKeyStatusStale},
		{[]byte("revoked"), enum.APIKeyStatusRevoked},
	}

	for i, test := range tests {
		var tt enum.APIKeyStatus
		err := tt.Scan(test.input)
		require.NoError(t, err, "test case %d failed", i)
		require.Equal(t, test.expected, tt, "test case %d failed", i)
	}

	var d enum.APIKeyStatus
	err := d.Scan("foo")
	require.EqualError(t, err, "invalid apikey status: \"foo\"")
	err = d.Scan(true)
	require.EqualError(t, err, "cannot scan bool into an apikey status")
}

func TestAPIKeyStatusValue(t *testing.T) {
	value, err := enum.APIKeyStatusActive.Value()
	require.NoError(t, err)
	require.Equal(t, "active", value)
}
