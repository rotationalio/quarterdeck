package telemetry

const (
	DefaultServiceName = "quarterdeck"
)

// Returns the service name for use in the otel resource. By default it is "quarterdeck"
// but can be overriden by the `$OTEL_SERVICE_NAME` environment variable. This method
// is used to ensure the service name is consistent across all components including
// logging (which might use a separate resource).
func ServiceName() string {
	conf := config.Get()

	return DefaultServiceName
}
