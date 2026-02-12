package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	nooplog "go.opentelemetry.io/otel/log/noop"
	noopmeter "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

func disableTelemetry(ctx context.Context) {
	var err error
	if qdResource, err = resource.New(ctx); err != nil {
		initerr = errors.Join(initerr, err)
	}

	qdPropagator = propagation.TraceContext{}
	otel.SetTextMapPropagator(qdPropagator)

	otel.SetTracerProvider(nooptrace.NewTracerProvider())
	otel.SetMeterProvider(noopmeter.NewMeterProvider())
	global.SetLoggerProvider(nooplog.NewLoggerProvider())

	disabled = true
}
