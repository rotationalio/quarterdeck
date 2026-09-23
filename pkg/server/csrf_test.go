package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/csrf"
)

// Verifies that the CSRF bootstrap endpoint issues the namespaced cookie pair
// once and preserves an existing valid pair on subsequent requests.
func TestCSRFTokenBootstrapsAndPreservesCookies(t *testing.T) {
	handler, err := csrf.NewTokenHandlerWithNamespace(
		5*time.Minute, "/", nil, make([]byte, 32), "quarterdeck",
	)
	require.NoError(t, err)

	s := &Server{csrf: &sameSiteCSRF{TokenHandler: handler}}
	// HACK: gimlet/csrf has no namespace helper, so we create a throwaway
	// handler to get the names for now.
	// TODO: add `csrf.NewNamespace(namespace string) csrf.Namespace` to gimlet then use here.
	names := handler.(csrf.Namespacer).Namespace()

	first := httptest.NewRecorder()
	firstContext, _ := gin.CreateTestContext(first)
	firstContext.Request = httptest.NewRequest(http.MethodGet, "http://localhost/csrf", nil)
	s.CSRFToken(firstContext)

	require.Equal(t, http.StatusNoContent, first.Code)
	cookies := first.Result().Cookies()
	require.Len(t, cookies, 2)
	var tokenCookie, referenceCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case names.Cookie:
			tokenCookie = cookie
		case names.ReferenceCookie:
			referenceCookie = cookie
		}
	}
	require.NotNil(t, tokenCookie)
	require.NotNil(t, referenceCookie)
	require.NotEmpty(t, tokenCookie.Value)
	require.Equal(t, tokenCookie.Value, referenceCookie.Value)
	require.False(t, tokenCookie.HttpOnly)
	require.False(t, tokenCookie.Secure)
	require.True(t, referenceCookie.HttpOnly)
	require.False(t, referenceCookie.Secure)
	require.Equal(t, http.SameSiteLaxMode, tokenCookie.SameSite)
	require.Equal(t, http.SameSiteLaxMode, referenceCookie.SameSite)

	// HTTPS requests must retain Secure on both cookies, including for a local
	// host. This protects deployments that terminate TLS at the application.
	httpsRecorder := httptest.NewRecorder()
	httpsContext, _ := gin.CreateTestContext(httpsRecorder)
	httpsContext.Request = httptest.NewRequest(http.MethodGet, "https://localhost/csrf", nil)
	s.CSRFToken(httpsContext)

	httpsCookies := httpsRecorder.Result().Cookies()
	require.Len(t, httpsCookies, 2)
	for _, cookie := range httpsCookies {
		require.True(t, cookie.Secure, "HTTPS CSRF cookie %q should be Secure", cookie.Name)
	}

	second := httptest.NewRecorder()
	secondContext, _ := gin.CreateTestContext(second)
	secondContext.Request = httptest.NewRequest(http.MethodGet, "http://localhost/csrf", nil)
	secondContext.Request.AddCookie(tokenCookie)
	secondContext.Request.AddCookie(referenceCookie)
	s.CSRFToken(secondContext)

	require.Equal(t, http.StatusNoContent, second.Code)
	require.Empty(t, second.Result().Cookies())
}
