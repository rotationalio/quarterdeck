package redirect_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/redirect"
)

func TestNewOrigin(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []string{
			"https://auth.example.com",
			"https://example.com",
			"http://localhost:8000",
			"http://localhost:8888",
			"http://endeavor.local",
			"http://endeavor.local:8000",
		}

		for _, tc := range testCases {
			_, err := redirect.New(tc)
			require.NoError(t, err, "should not error for valid origin: %s", tc)

			// Should not panic!
			require.NotPanics(t, func() {
				redirect.MustNew(tc)
			})
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			origin string
			errs   string
		}{
			{"\x00", `parse "\x00": net/url: invalid control character in URL`},
			{"localhost:9999", `invalid origin: "localhost:9999"`},
			{"http://", `invalid origin: "http://"`},
		}

		for _, tc := range testCases {
			_, err := redirect.New(tc.origin)
			require.Error(t, err, "should error for invalid origin")
			require.EqualError(t, err, tc.errs, "should return expected error for invalid origin")

			require.Panics(t, func() {
				redirect.MustNew(tc.origin)
			}, "should panic for invalid origin")
		}
	})

	t.Run("Normalize", func(t *testing.T) {
		// All path and fragment components should be cleaned up
		testCases := []string{
			"https://example.com",
			"https://example.com/",
			"https://example.com/path/to/nowhere",
			"https://example.com?color=red",
			"https://example.com/foo?color=red",
			"https://example.com#fragment",
		}

		for _, tc := range testCases {
			origin, err := redirect.New(tc)
			require.NoError(t, err, "should not error for valid origin: %s", tc)
			require.Equal(t, "https://example.com", origin.String())
		}
	})

	t.Run("NormalizePort", func(t *testing.T) {
		// All path and fragment components should be cleaned up
		testCases := []string{
			"http://localhost:8000",
			"http://localhost:8000/",
			"http://localhost:8000/path/to/nowhere",
			"http://localhost:8000?color=red",
			"http://localhost:8000/foo?color=red",
			"http://localhost:8000#fragment",
		}

		for _, tc := range testCases {
			origin, err := redirect.New(tc)
			require.NoError(t, err, "should not error for valid origin: %s", tc)
			require.Equal(t, "http://localhost:8000", origin.String())
		}
	})
}

func TestLocation(t *testing.T) {
	origin, err := redirect.New("https://example.com")
	require.NoError(t, err, "should not error creating origin")

	t.Run("Happy", func(t *testing.T) {
		testCases := []struct {
			next     any
			expected string
		}{
			{&url.URL{Scheme: "https", Host: "example.com", Path: "/login"}, "/login"},
			{url.URL{Scheme: "https", Host: "auth.example.com", Path: "/login"}, "https://auth.example.com/login"},
			{"https://example.com/login", "/login"},
			{"https://auth.example.com/login", "https://auth.example.com/login"},
			{"/login", "/login"},
			{"https://example.com/login?next=foo", "/login?next=foo"},
			{"https://auth.example.com/login?next=foo", "https://auth.example.com/login?next=foo"},
			{"/login?next=foo", "/login?next=foo"},
		}

		for i, tc := range testCases {
			actual, err := origin.Location(tc.next)
			require.NoError(t, err, "should not error for valid location in test case %d", i)
			require.Equal(t, tc.expected, actual, "should return expected location in test case %d", i)

			require.NotPanics(t, func() {
				origin.MustLocation(tc.next)
			}, "should not panic for valid location in test case %d", i)
		}
	})

	t.Run("Errors", func(t *testing.T) {
		testCases := []struct {
			next any
			errs string
		}{
			{42, "unsupported type int"},
			{"\x00", `parse "\x00": net/url: invalid control character in URL`},
		}

		for i, tc := range testCases {
			_, err := origin.Location(tc.next)
			require.Error(t, err, "should error for invalid location in test case %d", i)
			require.EqualError(t, err, tc.errs, "should return expected error for invalid location in test case %d", i)

			require.Panics(t, func() {
				origin.MustLocation(tc.next)
			}, "should panic for invalid location in test case %d", i)
		}
	})
}
