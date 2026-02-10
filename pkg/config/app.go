package config

import (
	"net/url"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// Configures the details of the application that is utilizing Quarterdeck for auth.
type AppConfig struct {
	Name             string           `split_words:"true" default:"Quarterdeck"  desc:"the descriptive name of the application (default: 'Quarterdeck')"`
	LogoURI          string           `split_words:"true" default:"https://rotational.ai/hs-fs/hubfs/Rotational%20Logo%20Hor%201073x280.png"  desc:"the logo for the application (default: a Rotational Labs logo)"`
	BaseURI          string           `split_words:"true" default:"http://localhost:8888"  desc:"base URL for the application (default: 'http://localhost:8888)"`
	WelcomeEmailBody WelcomeEmailBody `split_words:"true"`

	// Configures user syncing. Quarterdeck will attempt to post any new or modified
	// users to each of the endpoints provided. A create/update for a user will be
	// sent via an HTTP POST with the [api.User] JSON to the endpoints, and when a
	// user is deleted the user's ID will be appended to each endpoint as a path
	// parameter and sent via an HTTP DELETE. This is a "best effort" functionality,
	// so failures will be logged but not handled at that time.
	//
	// TODO: this endpoint config and callback code can be removed in favor of the
	// OIDC callback method in the future once OIDC is implemented more completely.
	WebhookURI string `split_words:"true" required:"false" desc:"webhook endpoint for the application to recieve user sync events"`
}

func (c AppConfig) Validate() (err error) {
	if _, err := url.Parse(c.LogoURI); err != nil {
		return errors.ConfigError(err, errors.InvalidConfig("appConfig", "logoURI", "url '%s' is unparseable", c.LogoURI))
	}

	if _, err := url.Parse(c.BaseURI); err != nil {
		return errors.ConfigError(err, errors.InvalidConfig("appConfig", "baseURI", "url '%s' is unparseable", c.BaseURI))
	}

	if _, err := url.Parse(c.WebhookURI); err != nil {
		return errors.ConfigError(err, errors.InvalidConfig("appConfig", "webhookURI", "url '%s' is unparseable", c.WebhookURI))
	}

	return nil
}

func (c AppConfig) LogoURL() *url.URL {
	// Ignore errors because we have already validated the config
	u, _ := url.Parse(c.LogoURI)
	return u
}

func (c AppConfig) BaseURL() *url.URL {
	// Ignore errors because we have already validated the config
	u, _ := url.Parse(c.BaseURI)
	return u
}

func (c AppConfig) WebhookURL() *url.URL {
	// Ignore errors because we have already validated the config
	u, _ := url.Parse(c.WebhookURI)
	return u
}

// ############################################################################
// Welcome Email Body
// ############################################################################

type WelcomeEmailBody struct {
	HTML string `split_words:"true" default:"" desc:"welcome email body as HTML (default is generic)"`
	Text string `split_words:"true" default:"" desc:"welcome email body as text (default is generic)"`
}
