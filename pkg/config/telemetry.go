package config

// Telemetry is primarily configured via the open telemetry sdk environment variables.
// As such there is no need to specify OTel specific configuration here. This config
// is used primarily to enable/disable telemetry and to set values for custom telemetry.
//
// See: https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/
// For the environment variables that can be used to configure telemetry.
//
// See Also: https://oneuptime.com/blog/post/2026-02-06-opentelemetry-environment-variables-zero-code/view
// For OpenTelemetry configuration best practices.
type TelemetryConfig struct {
	Enabled     bool   `default:"true" desc:"disable telemetry by setting this environment variable to false"`
	ServiceName string `split_words:"true" env:"OTEL_SERVICE_NAME" desc:"override the default name of the service, used for logging and telemetry"`
	ServiceAddr string `split_words:"true" env:"GIMLET_OTEL_SERVICE_ADDR" desc:"the primary server name if it is known. E.g. the server name directive in an Nginx config. Should include host identifier and port if it is used; empty if not known."`
}
