package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.rtnl.ai/gimlet/o11y"
)

// Verifies that successful responses are marked as OK and all others as errors
// with no unset responses.
func TestMiddlewareSuccessfulSpanStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		addError   bool
		wantStatus codes.Code
	}{
		{name: "success", statusCode: http.StatusOK, wantStatus: codes.Ok},
		{name: "no content", statusCode: http.StatusNoContent, wantStatus: codes.Ok},
		{name: "client error", statusCode: http.StatusBadRequest, wantStatus: codes.Error},
		{name: "server error", statusCode: http.StatusInternalServerError, wantStatus: codes.Error},
		{name: "success with context error", statusCode: http.StatusOK, addError: true, wantStatus: codes.Error},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })

			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.Use(o11y.Middleware("quarterdeck", o11y.WithTracerProvider(provider)), SetSuccessfulSpanStatus())
			router.GET("/", func(c *gin.Context) {
				if test.addError {
					c.Error(errors.New("handler error"))
				}
				c.Status(test.statusCode)
			})

			request := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			require.Equal(t, test.statusCode, response.Code)
			spans := recorder.Ended()
			require.Len(t, spans, 1)
			require.Equal(t, test.wantStatus, spans[0].Status().Code)
		})
	}
}
