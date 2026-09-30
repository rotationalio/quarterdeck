package config

import (
	"net/url"
	"strings"

	csrf "go.rtnl.ai/gimlet/csrf/secfetch"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

type CSRFConfig struct {
	Disabled                 bool     `default:"false" desc:"log CSRF failures without rejecting requests"`
	Namespace                string   `default:"quarterdeck" desc:"the namespace for the CSRF error header"`
	SafeHTTPMethods          []string `split_words:"true" default:"GET,HEAD,OPTIONS" desc:"safe HTTP methods that bypass CSRF checks"`
	ExpectedOrigins          []string `split_words:"true" desc:"exact trusted HTTP(S) origins for same-site and Origin/Referer checks"`
	AllowMissingMetadata     bool     `split_words:"true" default:"false" desc:"allow requests missing Fetch Metadata even without trusted origins"`
	AllowUnknownSite         bool     `split_words:"true" default:"false" desc:"allow unknown Sec-Fetch-Site values"`
	AllowSiteNone            bool     `split_words:"true" default:"false" desc:"allow unsafe requests with Sec-Fetch-Site none"`
	AllowedFetchModes        []string `split_words:"true" desc:"allowed Sec-Fetch-Mode values; empty is unrestricted"`
	RequireFetchMode         bool     `split_words:"true" default:"false" desc:"require Sec-Fetch-Mode"`
	AllowedFetchDestinations []string `split_words:"true" desc:"allowed Sec-Fetch-Dest values; empty is unrestricted"`
	RequireFetchDestination  bool     `split_words:"true" default:"false" desc:"require Sec-Fetch-Dest"`
}

func (c CSRFConfig) Validate() (err error) {
	for _, origin := range c.ExpectedOrigins {
		u, perr := url.Parse(origin)
		if perr != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(origin, "*") {
			err = errors.ConfigError(err, errors.InvalidConfig("csrf", "expectedOrigins", "origin %q must be an exact http(s) origin", origin))
		}
	}
	for _, method := range c.SafeHTTPMethods {
		switch strings.ToUpper(strings.TrimSpace(method)) {
		case "GET", "HEAD", "OPTIONS":
		default:
			err = errors.ConfigError(err, errors.InvalidConfig("csrf", "safeHTTPMethods", "method %q must be GET, HEAD, or OPTIONS", method))
		}
	}
	return err
}

// Converts configuration into Fetch Metadata policy options, using log-only mode when disabled.
func (c CSRFConfig) Options() []csrf.Option {
	opts := []csrf.Option{
		csrf.WithLogOnly(c.Disabled),
		csrf.WithNamespace(c.Namespace),
		csrf.WithExpectedOrigins(c.ExpectedOrigins),
		csrf.WithAllowMissingMetadata(c.AllowMissingMetadata),
		csrf.WithAllowUnknownSite(c.AllowUnknownSite),
		csrf.WithAllowSiteNone(c.AllowSiteNone),
		csrf.WithAllowedFetchModes(c.AllowedFetchModes),
		csrf.WithAllowedFetchDestinations(c.AllowedFetchDestinations),
		csrf.WithRequireFetchMode(c.RequireFetchMode),
		csrf.WithRequireFetchDestination(c.RequireFetchDestination),
	}
	// Preserve Gimlet's defaults for programmatically constructed configurations.
	// An explicitly empty (non-nil) list checks even safe methods.
	if c.SafeHTTPMethods != nil {
		opts = append(opts, csrf.WithSafeHTTPMethods(c.SafeHTTPMethods))
	}
	return opts
}

// Mirrors Gimlet's namespace normalization until it exports a header-name helper.
func (c CSRFConfig) ErrorHeader() string {
	namespace := strings.ToLower(strings.TrimSpace(c.Namespace))
	if namespace == "" {
		return csrf.HeaderError
	}
	var normalized strings.Builder
	for _, char := range namespace {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9', char == '-', char == '_':
			normalized.WriteRune(char)
		default:
			normalized.WriteByte('_')
		}
	}
	name := normalized.String()
	return "X-" + strings.ToUpper(name[:1]) + name[1:] + "-CSRF-Error"
}
