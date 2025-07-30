package htmx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/web/htmx"
)

func TestRedirect(t *testing.T) {
	t.Run("HTMX", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request

		htmx.Redirect(c, http.StatusFound, "/new-location")

		require.Equal(t, http.StatusNoContent, w.Code)
		require.Equal(t, "/new-location", w.Header().Get(htmx.HXRedirect))
		require.Empty(t, w.Header().Get("Location"))

	})

	t.Run("Web", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		htmx.Redirect(c, http.StatusFound, "/new-location")

		require.Equal(t, http.StatusFound, w.Code)
		require.Equal(t, "/new-location", w.Header().Get("Location"))
		require.Empty(t, w.Header().Get(htmx.HXRedirect))
	})
}

func TestIsWebRequest(t *testing.T) {
	t.Run("HTMX", func(t *testing.T) {
		t.Run("JSON", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request
			c.Request.Header.Set("Accept", "application/json")

			require.True(t, htmx.IsWebRequest(c))
		})

		t.Run("HTML", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request
			c.Request.Header.Set("Accept", "text/html")

			require.True(t, htmx.IsWebRequest(c))
		})
		t.Run("ANY", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request
			c.Request.Header.Set("Accept", "*/*")

			require.True(t, htmx.IsWebRequest(c))
		})
		t.Run("XHTML", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request
			c.Request.Header.Set("Accept", "text/xhtml+xml")

			require.True(t, htmx.IsWebRequest(c))
		})
		t.Run("YAML", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request
			c.Request.Header.Set("Accept", "application/yaml")

			require.True(t, htmx.IsWebRequest(c))
		})
	})

	t.Run("Web", func(t *testing.T) {
		t.Run("JSON", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Accept", "application/json")

			require.False(t, htmx.IsWebRequest(c))
		})

		t.Run("HTML", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Accept", "text/html")

			require.True(t, htmx.IsWebRequest(c))
		})
		t.Run("ANY", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Accept", "*/*")

			require.False(t, htmx.IsWebRequest(c))
		})
		t.Run("XHTML", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Accept", "text/xhtml+xml")

			// Very surprising that this is false, but the gin parser says that this
			// is not a mime html binding.
			require.False(t, htmx.IsWebRequest(c))
		})
		t.Run("YAML", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)
			c.Request.Header.Set("Accept", "application/yaml")

			require.False(t, htmx.IsWebRequest(c))
		})
	})
}

func TestTrigger(t *testing.T) {
	t.Run("HTMX", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)
		c.Request.Header.Set(htmx.HXRequest, "true") // Simulate an HTMX request

		htmx.Trigger(c, "test-event")

		require.Equal(t, http.StatusNoContent, w.Code)
		require.Equal(t, "test-event", w.Header().Get(htmx.HXTrigger))
	})

	t.Run("Web", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/test", nil)

		htmx.Trigger(c, "test-event")

		require.Equal(t, http.StatusOK, w.Code)
		require.Empty(t, w.Header().Get(htmx.HXTrigger))
	})
}
