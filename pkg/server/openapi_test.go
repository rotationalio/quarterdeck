package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/web"
)

const docsProductBlurb = "Quarterdeck is a distributed authentication and authorization service that issues JWTs, serves JWKS, and provides user and API key management for Rotational applications."

func TestOpenAPITemplatesRenderVersionAndTitle(t *testing.T) {
	files, err := fs.Sub(web.Templates(), "docs/openapi")
	require.NoError(t, err)

	templates, err := template.ParseFS(files, "*.json", "*.yaml")
	require.NoError(t, err)

	data := map[string]string{
		keyVersion: "9.9.9-test",
		keyOrigin:  "http://example.test",
		keyTitle:   "Quarterdeck API Docs",
	}

	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		t.Run(name, func(t *testing.T) {
			var buf strings.Builder
			require.NoError(t, templates.ExecuteTemplate(&buf, name, data))

			out := buf.String()

			require.Contains(t, out, "9.9.9-test")
			require.Contains(t, out, "Quarterdeck API Docs")
			require.Contains(t, out, docsProductBlurb)
			require.NotContains(t, out, "{{ .Version }}")
			require.NotContains(t, out, "{{ .Title }}")

			if name == "openapi.json" {
				var spec struct {
					Info struct {
						Title       string `json:"title"`
						Version     string `json:"version"`
						Summary     string `json:"summary"`
						Description string `json:"description"`
					} `json:"info"`
				}
				require.NoError(t, json.Unmarshal([]byte(out), &spec))
				require.Equal(t, "9.9.9-test", spec.Info.Version)
				require.Equal(t, "Quarterdeck API Docs", spec.Info.Title)
				require.Equal(t, docsProductBlurb, spec.Info.Summary)
				require.Equal(t, docsProductBlurb, spec.Info.Description)
			}
		})
	}
}

func TestOpenAPIHandlerServesRenderedJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	conf := config.Config{
		DocsName: "Quarterdeck API Docs",
	}
	srv := &Server{conf: conf}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/docs/openapi.json", nil)
	ctx.Params = gin.Params{{Key: "ext", Value: "json"}}

	srv.OpenAPI()(ctx)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), pkg.Version(false))
	require.Contains(t, rec.Body.String(), conf.DocsName)
	require.Contains(t, rec.Body.String(), docsProductBlurb)
	require.NotContains(t, rec.Body.String(), "{{ .Version }}")
	require.NotContains(t, rec.Body.String(), "{{ .Title }}")
}

func TestAPIDocsHTMLRendersTitleLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	render, err := web.HTMLRender(web.Templates())
	require.NoError(t, err)

	rec := httptest.NewRecorder()

	require.NoError(t, render.Instance("docs/openapi/openapi.html", gin.H{
		keyTitle: "Quarterdeck API Docs",
	}).Render(rec))

	out := rec.Body.String()

	require.Contains(t, out, `data-docs-label="Quarterdeck API Docs"`)
	require.Contains(t, out, "dataset.docsLabel")
	require.Contains(t, out, "quarterdeck-home-link")
	require.Contains(t, out, "onLoaded:")
	require.Contains(t, out, "/static/js/scalar/standalone.js")
	require.NotContains(t, out, "rapidoc")
	require.NotContains(t, out, "{{ .Title }}")
}
