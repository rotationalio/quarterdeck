package redirect_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/redirect"
)

func TestLogin(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []string{
			"https://example.com",
			"https://example.com/login",
			"http://localhost:8000",
			"http://localhost:8000/login",
			"http://endeavor.local",
			"http://endeavor.local:8000",
			"http://example.com/accounts/login",
			"http://localhost:8000/accounts/login",
			"https://example.com/login?callback=token",
			"http://localhost:8000/login?callback=token",
		}

		for _, tc := range testCases {
			login, err := redirect.Login(tc)
			require.NoError(t, err, "should not error for valid login url: %s", tc)
			require.Equal(t, tc, login.String(), "login URL should match original in test case: %s", tc)

			// Should not panic!
			require.NotPanics(t, func() {
				redirect.MustLogin(tc)
			})
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			origin string
			errs   string
		}{
			{"\x00", `parse "\x00": net/url: invalid control character in URL`},
			{"localhost:9999", `invalid login url: "localhost:9999"`},
			{"http://", `invalid login url: "http://"`},
		}

		for _, tc := range testCases {
			_, err := redirect.Login(tc.origin)
			require.Error(t, err, "should error for invalid login url")
			require.EqualError(t, err, tc.errs, "should return expected error for invalid login url")

			require.Panics(t, func() {
				redirect.MustLogin(tc.origin)
			}, "should panic for invalid login url")
		}
	})

	t.Run("Normalize", func(t *testing.T) {

	})

	t.Run("NormalizePort", func(t *testing.T) {

	})
}

func TestLoginURL(t *testing.T) {
	t.Run("Location", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			login    string
			next     string
			expected string
		}{
			{
				"https://auth.rotational.app/login",
				"https://rotational.app/dashboard",
				"https://auth.rotational.app/login?next=https%3A%2F%2Frotational.app%2Fdashboard",
			},
			{
				"https://auth.rotational.app/login?apikey=12345",
				"https://rotational.app/dashboard",
				"https://auth.rotational.app/login?apikey=12345&next=https%3A%2F%2Frotational.app%2Fdashboard",
			},
			{
				"https://auth.rotational.app/login",
				"https://rotational.app/dashboard?color=blue",
				"https://auth.rotational.app/login?next=https%3A%2F%2Frotational.app%2Fdashboard%3Fcolor%3Dblue",
			},
			{
				"https://rotational.app/login",
				"https://rotational.app/dashboard",
				"/login?next=%2Fdashboard",
			},
			{
				"https://rotational.app/login?apikey=12345",
				"https://rotational.app/dashboard",
				"/login?apikey=12345&next=%2Fdashboard",
			},
			{
				"https://rotational.app/login",
				"https://rotational.app/dashboard?color=blue",
				"/login?next=%2Fdashboard%3Fcolor%3Dblue",
			},
			{
				"https://localhost:8000/login",
				"https://localhost:8888/dashboard",
				"https://localhost:8000/login?next=https%3A%2F%2Flocalhost%3A8888%2Fdashboard",
			},
			{
				"https://localhost:8000/login?apikey=12345",
				"https://localhost:8888/dashboard",
				"https://localhost:8000/login?apikey=12345&next=https%3A%2F%2Flocalhost%3A8888%2Fdashboard",
			},
			{
				"https://localhost:8000/login",
				"https://localhost:8888/dashboard?color=blue",
				"https://localhost:8000/login?next=https%3A%2F%2Flocalhost%3A8888%2Fdashboard%3Fcolor%3Dblue",
			},
			{
				"https://localhost:8000/login",
				"https://localhost:8000/dashboard",
				"/login?next=%2Fdashboard",
			},
			{
				"https://localhost:8000/login?apikey=12345",
				"https://localhost:8000/dashboard",
				"/login?apikey=12345&next=%2Fdashboard",
			},
			{
				"https://localhost:8000/login",
				"https://localhost:8000/dashboard?color=blue",
				"/login?next=%2Fdashboard%3Fcolor%3Dblue",
			},
		}

		t.Run("Gin", func(t *testing.T) {
			for i, tc := range testCases {
				loginURL, err := redirect.Login(tc.login)
				require.NoError(t, err, "should not error creating login URL in test case %d", i)

				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest(http.MethodGet, tc.next, nil)

				actual, err := loginURL.Location(c)
				require.NoError(t, err, "should not error getting login URL location in test case %d", i)
				require.Equal(t, tc.expected, actual, "login URL location should match expected in test case %d", i)

				require.NotPanics(t, func() {
					loginURL.MustLocation(c)
				}, "should not panic for valid login URL in test case %d", i)
			}
		})

		t.Run("Request", func(t *testing.T) {
			for i, tc := range testCases {
				loginURL, err := redirect.Login(tc.login)
				require.NoError(t, err, "should not error creating login URL in test case %d", i)

				c := httptest.NewRequest(http.MethodGet, tc.next, nil)

				actual, err := loginURL.Location(c)
				require.NoError(t, err, "should not error getting login URL location in test case %d", i)
				require.Equal(t, tc.expected, actual, "login URL location should match expected in test case %d", i)

				require.NotPanics(t, func() {
					loginURL.MustLocation(c)
				}, "should not panic for valid login URL in test case %d", i)

				actual, err = loginURL.Location(*c)
				require.NoError(t, err, "should not error getting login URL location in test case %d", i)
				require.Equal(t, tc.expected, actual, "login URL location should match expected in test case %d", i)

				require.NotPanics(t, func() {
					loginURL.MustLocation(*c)
				}, "should not panic for valid login URL in test case %d", i)
			}
		})

		t.Run("URL", func(t *testing.T) {
			for i, tc := range testCases {
				loginURL, err := redirect.Login(tc.login)
				require.NoError(t, err, "should not error creating login URL in test case %d", i)

				c, err := url.Parse(tc.next)
				require.NoError(t, err, "should not error parsing URL in test case %d", i)

				actual, err := loginURL.Location(c)
				require.NoError(t, err, "should not error getting login URL location in test case %d", i)
				require.Equal(t, tc.expected, actual, "login URL location should match expected in test case %d", i)

				require.NotPanics(t, func() {
					loginURL.MustLocation(c)
				}, "should not panic for valid login URL in test case %d", i)

				actual, err = loginURL.Location(*c)
				require.NoError(t, err, "should not error getting login URL location in test case %d", i)
				require.Equal(t, tc.expected, actual, "login URL location should match expected in test case %d", i)

				require.NotPanics(t, func() {
					loginURL.MustLocation(*c)
				}, "should not panic for valid login URL in test case %d", i)
			}
		})

		t.Run("String", func(t *testing.T) {
			for i, tc := range testCases {
				loginURL, err := redirect.Login(tc.login)
				require.NoError(t, err, "should not error creating login URL in test case %d", i)

				actual, err := loginURL.Location(tc.next)
				require.NoError(t, err, "should not error getting login URL location in test case %d", i)
				require.Equal(t, tc.expected, actual, "login URL location should match expected in test case %d", i)

				require.NotPanics(t, func() {
					loginURL.MustLocation(tc.next)
				}, "should not panic for valid login URL in test case %d", i)
			}
		})
	})
}
