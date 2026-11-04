package telemetry

import (
	"bytes"

	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

// Ensures that the LevelHandler filters log messages at the correct level.
func TestLevelHandlerFiltersMultiHandler(t *testing.T) {
	var stdout, otel bytes.Buffer
	var level slog.LevelVar
	level.Set(slog.LevelInfo)
	options := &slog.HandlerOptions{Level: &level}
	console := slog.NewJSONHandler(&stdout, options)
	otelHandler := &LevelHandler{
		wrapped: slog.NewJSONHandler(&otel, options),
		level:   level.Level,
	}
	logger := slog.New(slog.NewMultiHandler(console, otelHandler))

	logger.Debug("filtered")
	logger.Info("visible")
	require.NotContains(t, stdout.String(), "filtered")
	require.NotContains(t, otel.String(), "filtered")
	require.Contains(t, stdout.String(), "visible")
	require.Contains(t, otel.String(), "visible")

	level.Set(slog.LevelDebug)
	logger.Debug("debug-visible")
	require.Contains(t, stdout.String(), "debug-visible")
	require.Contains(t, otel.String(), "debug-visible")

}
