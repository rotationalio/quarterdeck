package docs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/quarterdeck/pkg/docs"
	"go.rtnl.ai/quarterdeck/pkg/web"
)

func TestRoutes(t *testing.T) {
	// Discard logging
	logger.Discard()
	defer logger.ResetLogger()

	// Setup a gin router and httptest server
	gin.SetMode("test")
	router := gin.Default()

	// Setup HTML template renderer.
	var err error
	router.HTMLRender, err = web.HTMLRender(web.Templates())
	require.NoError(t, err, "could not create HTML renderer")

	// Add documentation routes
	docs.Routes(router.Group("/docs"))

	srv := httptest.NewServer(router)
	defer srv.Close()

	for _, page := range docs.Docs() {
		t.Run(page.Title, func(t *testing.T) {
			rep, err := srv.Client().Get(srv.URL + page.Href)
			require.NoError(t, err, "could not fetch page or render template")
			require.Equal(t, http.StatusOK, rep.StatusCode, "expected a Ok response with the rendered template")

			// Test all subpages
			for _, subpage := range page.Pages {
				t.Run(subpage.Title, func(t *testing.T) {
					rep, err := srv.Client().Get(srv.URL + subpage.Href())
					require.NoError(t, err, "could not fetch subpage or render template")
					require.Equal(t, http.StatusOK, rep.StatusCode, "expected a Ok response with the rendered template")
				})
			}
		})
	}
}
