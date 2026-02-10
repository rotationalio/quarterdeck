package telemetry

import "go.rtnl.ai/quarterdeck/pkg/config"

const (
	DefaultServiceName = "quarterdeck"
)

// Returns the service name for use in the otel resource. By default it is "quarterdeck"
// but can be overriden by the `$OTEL_SERVICE_NAME` environment variable. This method
// is used to ensure the service name is consistent across all components including
// logging (which might use a separate resource).
func ServiceName() string {
	conf, err := config.Get()
	if err != nil {
		return DefaultServiceName
	}
	return conf.ServiceName
}
