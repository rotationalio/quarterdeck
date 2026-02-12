package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"

	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/config"
)

func newResource(ctx context.Context) (*resource.Resource, error) {
	conf, _ := config.Get()
	return resource.New(ctx,
		// Resource Detectors
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithContainer(),

		// NOTE: these custom attributes should go last so they override any
		// attributes set by the resource detectors above.
		resource.WithAttributes(
			semconv.ServiceName(ServiceName()),
			semconv.ServiceVersion(pkg.Version(false)),
			semconv.ServiceInstanceID(conf.Org.Name),
		),
	)
}

// Uses autoprop to automatically create the correct propagator based on the
// environment, defaulting to a combined trace context and baggage text map propagator.
func newPropagator() (propagation.TextMapPropagator, error) {
	return autoprop.NewTextMapPropagator(), nil
}

// Uses autoexport to automatically create the correct tracer exporter based on the
// environment, defaulting to a console exporter if $OTEL_TRACES_EXPORTER is empty.
// The traces sampler is also configured from the environment, with the default
// AlwaysSample as the sampler if no environment variables are set.
func newTracerProvider(ctx context.Context, resource *resource.Resource) (tracerProvider *trace.TracerProvider, err error) {
	// Default to a console exporter if $OTEL_TRACES_EXPORTER is empty.
	opts := []autoexport.SpanOption{
		autoexport.WithFallbackSpanExporter(func(context.Context) (trace.SpanExporter, error) {
			return stdouttrace.New(stdouttrace.WithPrettyPrint())
		}),
	}

	var tracerExporter trace.SpanExporter
	if tracerExporter, err = autoexport.NewSpanExporter(ctx, opts...); err != nil {
		return nil, err
	}

	// NOTE: do not specify sampler in order for it to be configured from the environment.
	tracerProvider = trace.NewTracerProvider(
		trace.WithResource(resource),
		trace.WithBatcher(tracerExporter),
	)
	return tracerProvider, nil
}

// Uses autoexport to automatically create the correct meter exporter based on the
// environment, defaulting to a console exporter if $OTEL_METRICS_EXPORTER is empty.
func newMeterProvider(ctx context.Context, resource *resource.Resource) (meterProvider *metric.MeterProvider, err error) {
	// Default to a console exporter if $OTEL_METRICS_EXPORTER is empty.
	opts := []autoexport.MetricOption{
		autoexport.WithFallbackMetricReader(func(context.Context) (_ metric.Reader, err error) {
			var exporter metric.Exporter
			if exporter, err = stdoutmetric.New(stdoutmetric.WithPrettyPrint()); err != nil {
				return nil, err
			}
			return metric.NewPeriodicReader(exporter, metric.WithInterval(10*time.Second)), nil
		}),
	}

	var metricsReader metric.Reader
	if metricsReader, err = autoexport.NewMetricReader(ctx, opts...); err != nil {
		return nil, err
	}

	// TODO: add custom metrics for Quarterdeck here.
	meterProvider = metric.NewMeterProvider(
		metric.WithResource(resource),
		metric.WithReader(metricsReader),
	)
	return meterProvider, nil
}

// Uses autoexport to automatically create the correct logger exporter based on the
// environment, defaulting to a console exporter if $OTEL_LOGS_EXPORTER is empty.
func newLoggerProvider(ctx context.Context, resource *resource.Resource) (loggerProvider *log.LoggerProvider, err error) {

	opts := []autoexport.LogOption{}

	var loggerExporter log.Exporter
	if loggerExporter, err = autoexport.NewLogExporter(ctx, opts...); err != nil {
		return nil, err
	}

	loggerProvider = log.NewLoggerProvider(
		log.WithResource(resource),
		log.WithProcessor(log.NewBatchProcessor(loggerExporter)),
	)
	return loggerProvider, nil
}
