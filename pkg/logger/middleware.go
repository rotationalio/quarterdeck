package logger

import (
	"fmt"
	"time"

	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/metrics"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/ulid"
)

// GinLogger returns a new Gin middleware that performs logging for our JSON APIs using
// zerolog rather than the default Gin logger which is a standard HTTP logger.
// NOTE: we previously used github.com/dn365/gin-zerolog but wanted more customization.
func GinLogger(server string) gin.HandlerFunc {
	version := pkg.Version(false)

	// Initialize prometheus collectors (safe to call multiple times)
	metrics.Setup()

	return func(c *gin.Context) {
		// Before request
		started := time.Now()

		path := c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			path = path + "?" + c.Request.URL.RawQuery
		}

		// Create a request ID for tracing purposes and add to context
		// HACK: this creates a shallow copy of the request, which might cause issues?
		requestID := ulid.Make().String()
		c.Request = c.Request.WithContext(WithRequestID(c.Request.Context(), requestID))

		// Handle the request
		c.Next()

		// After request
		status := c.Writer.Status()
		logctx := log.With().
			Str("path", path).
			Str("ser_name", server).
			Str("version", version).
			Str("method", c.Request.Method).
			Dur("resp_time", time.Since(started)).
			Int("resp_bytes", c.Writer.Size()).
			Int("status", status).
			Str("client_ip", c.ClientIP()).
			Str("request_id", requestID).
			Logger()

		// Log any errors that were added to the context
		if len(c.Errors) > 0 {
			errs := make([]error, 0, len(c.Errors))
			for _, err := range c.Errors {
				errs = append(errs, err)
			}
			logctx = logctx.With().Errs("errors", errs).Logger()
		}

		// Create the message to send to the logger.
		var msg string
		switch len(c.Errors) {
		case 0:
			msg = fmt.Sprintf("%s %s %s %d", server, c.Request.Method, c.Request.URL.Path, status)
		case 1:
			msg = c.Errors.String()
		default:
			msg = fmt.Sprintf("%s %s %s [%d] %d errors occurred", server, c.Request.Method, c.Request.URL.Path, status, len(c.Errors))
		}

		switch {
		case status >= 400 && status < 500:
			logctx.Warn().Msg(msg)
		case status >= 500:
			logctx.Error().Msg(msg)
		default:
			logctx.Info().Msg(msg)
		}

		// prometheus metrics - log request duration and type
		duration := time.Since(started)
		statusCode := fmt.Sprintf("%d", status)
		metrics.RequestDuration.WithLabelValues(server, statusCode, path).Observe(duration.Seconds())
		metrics.RequestsHandled.WithLabelValues(server, statusCode, path).Inc()
	}
}
