package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"go.rtnl.ai/quarterdeck/pkg/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
)

const (
	DefaultServiceName = "quarterdeck"
)

type shutdownFn func(ctx context.Context) error

var (
	shutdown []shutdownFn
	initmu   sync.Once
	initerr  error
	closemu  sync.Mutex
	closeerr error

	qdResource       *resource.Resource
	qdPropagator     propagation.TextMapPropagator
	qdTracerProvider *trace.TracerProvider
	qdMeterProvider  *metric.MeterProvider
	qdLoggerProvider *log.LoggerProvider
)

// Setup initializes the opentelemetry sdk components and sets them as the global
// providers. In general, the setup is primarily configured via the standard OTEL_*
// environment variables, and to a limited extent via the config package.
//
// The setup function is idempotent and can be called multiple times, but will only be
// configured once; modifying the environment after the first call will have no effect.
func Setup(ctx context.Context) (err error) {
	initmu.Do(func() { setup(ctx) })
	return initerr
}

func setup(ctx context.Context) {
	closemu.Lock()
	defer closemu.Unlock()

	// Setup module telemetry handlers
	shutdown = make([]shutdownFn, 0, 4)
	initerr = nil
	closeerr = nil

	// Cleanup is only called if there is an error during setup; shutting down any
	// open telemetry sdk objects that have been created before the error occurred.
	cleanup := func(ctx context.Context) error {
		for _, fn := range shutdown {
			closeerr = errors.Join(closeerr, fn(ctx))
		}

		shutdown = nil
		return closeerr
	}

	var err error
	if qdResource, err = newResource(ctx); err != nil {
		initerr = errors.Join(err, cleanup(ctx))
		return
	}

	// Set up propagator
	if qdPropagator, err = newPropagator(); err != nil {
		initerr = errors.Join(err, cleanup(ctx))
		return
	}
	otel.SetTextMapPropagator(qdPropagator)

	// Set up tracer provider
	if qdTracerProvider, err = newTracerProvider(ctx, qdResource); err != nil {
		initerr = errors.Join(err, cleanup(ctx))
		return
	}
	shutdown = append(shutdown, qdTracerProvider.Shutdown)
	otel.SetTracerProvider(qdTracerProvider)

	// Set up meter provider
	if qdMeterProvider, err = newMeterProvider(ctx, qdResource); err != nil {
		initerr = errors.Join(err, cleanup(ctx))
		return
	}
	shutdown = append(shutdown, qdMeterProvider.Shutdown)
	otel.SetMeterProvider(qdMeterProvider)

	// Set up logger provider
	if qdLoggerProvider, err = newLoggerProvider(ctx, qdResource); err != nil {
		initerr = errors.Join(err, cleanup(ctx))
		return
	}
	shutdown = append(shutdown, qdLoggerProvider.Shutdown)
	global.SetLoggerProvider(qdLoggerProvider)
}

func Shutdown(ctx context.Context) error {
	closemu.Lock()
	if shutdown == nil {
		closemu.Unlock()
		return closeerr
	}

	for _, fn := range shutdown {
		closeerr = errors.Join(closeerr, fn(ctx))
	}

	shutdown = nil
	closemu.Unlock()

	return closeerr
}

func Propagator() propagation.TextMapPropagator {
	return qdPropagator
}

func TracerProvider() *trace.TracerProvider {
	return qdTracerProvider
}

func MeterProvider() *metric.MeterProvider {
	return qdMeterProvider
}

func LoggerProvider() *log.LoggerProvider {
	return qdLoggerProvider
}

// Returns the service name for use in the otel resource. By default it is "quarterdeck"
// but can be overridden by the `$OTEL_SERVICE_NAME` environment variable. This method
// is used to ensure the service name is consistent across all components including
// logging (which might use a separate resource).
func ServiceName() string {
	conf, err := config.Get()
	if err != nil || conf.Telemetry.ServiceName == "" {
		return DefaultServiceName
	}
	return conf.Telemetry.ServiceName
}

// Returns the service address for use in otel http server tracing. This address
// can be set by the `$GIMLET_OTEL_SERVICE_ADDR` environment variable. If not set by
// this value then it is inferred from the bind address and the name of the pod or the
// hostname of the machine running the service.
//
// The service address must be the primary server name if it is known. E.g. the server
// name directive in an Apache or Nginx configuration. More generically, the primary
// server name would be the host header value that matches the default virtual host of
// an HTTP server. It should include the host identifier and if a port is used to route
// to the server that port identifier should be included as an appropriate port suffix.
// If this nae is not known, server should be an empty string.
func ServiceAddr() string {
	var (
		err  error
		conf config.Config
	)

	// If the configuration is not available then return an empty string.
	if conf, err = config.Get(); err == nil && conf.Telemetry.ServiceAddr != "" {
		return ""
	}

	// If the service address is set in the configuration then return it.
	if conf.Telemetry.ServiceAddr != "" {
		return conf.Telemetry.ServiceAddr
	}

	// Attempt to infer the service address from the bind address.
	var (
		host string
		port string
	)

	if host, port, err = net.SplitHostPort(conf.BindAddr); err != nil {
		return ""
	}

	// If the host is specified in the bind address then return it (normalizing localhost)
	if host != "" {
		if host == "127.0.0.1" {
			return fmt.Sprintf("localhost:%s", port)
		}
		return conf.BindAddr
	}

	// Attempt to get the pod name from the environment.
	if pod, ok := os.LookupEnv("POD_NAME"); ok && pod != "" {
		return fmt.Sprintf("%s:%s", pod, port)
	}

	// Attempt to get the hostname from the environment.
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		return fmt.Sprintf("%s:%s", hostname, port)
	}

	return ""
}
