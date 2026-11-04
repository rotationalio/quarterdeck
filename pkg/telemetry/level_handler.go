package telemetry

import (
	"context"
	"log/slog"
)

// LevelHandler applies a dynamic minimum level to handlers that do not expose
// their own level configuration, such as the OpenTelemetry slog bridge.
type LevelHandler struct {
	wrapped slog.Handler
	level   func() slog.Level
}

// NewLevelHandler creates a handler that filters records below the configured level.
func NewLevelHandler(wrapped slog.Handler, level func() slog.Level) *LevelHandler {
	return &LevelHandler{wrapped: wrapped, level: level}
}

func (h *LevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level() && h.wrapped.Enabled(ctx, level)
}

func (h *LevelHandler) Handle(ctx context.Context, record slog.Record) error {
	return h.wrapped.Handle(ctx, record)
}

func (h *LevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LevelHandler{wrapped: h.wrapped.WithAttrs(attrs), level: h.level}
}

func (h *LevelHandler) WithGroup(name string) slog.Handler {
	return &LevelHandler{wrapped: h.wrapped.WithGroup(name), level: h.level}
}
