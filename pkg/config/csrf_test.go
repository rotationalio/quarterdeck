package config_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

// Accepts exact trusted origins and safe methods while rejecting unsafe configuration values.
func TestCSRFConfigValidate(t *testing.T) {
	require.NoError(t, (config.CSRFConfig{}).Validate())
	require.NoError(t, (config.CSRFConfig{
		ExpectedOrigins: []string{"https://app.example.com", "http://localhost:8888"},
		SafeHTTPMethods: []string{"GET", "HEAD", "OPTIONS"},
	}).Validate())
	for _, origin := range []string{"*", "https://*.example.com", "example.com", "ftp://example.com", "https://example.com/path", "https://example.com?x=1", "https://example.com#fragment", "https://user@example.com"} {
		t.Run(origin, func(t *testing.T) {
			require.Error(t, (config.CSRFConfig{
				ExpectedOrigins: []string{origin},
			}).Validate())
		})
	}
	require.Error(t, (config.CSRFConfig{
		SafeHTTPMethods: []string{"POST"},
	}).Validate())
}

// Confirms configured error header names match Gimlet's namespace normalization.
func TestCSRFErrorHeader(t *testing.T) {
	for _, namespace := range []string{"", "quarterdeck", " My App! ", "FOO_bar"} {
		t.Run(namespace, func(t *testing.T) {
			conf := config.CSRFConfig{
				Namespace: namespace,
			}
			router := gin.New()
			router.Use(csrf.Middleware(conf.Options()...))
			router.POST("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))
			require.Equal(t, http.StatusForbidden, recorder.Code)
			require.Equal(t, csrf.ErrorRequestRejected, recorder.Header().Get(conf.ErrorHeader()))
		})
	}
}
