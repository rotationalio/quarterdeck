package config

import (
	"html/template"
	"net/url"
	"os"
	"sync"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// Configures the details of the application that is utilizing Quarterdeck for auth.
type AppConfig struct {
	Name         string        `split_words:"true" default:"Quarterdeck"  desc:"the descriptive name of the application"`
	LogoURI      string        `split_words:"true" default:"https://rotational.ai/hs-fs/hubfs/Rotational%20Logo%20Hor%201073x280.png"  desc:"the logo for the application"`
	BaseURI      string        `split_words:"true" env:"QD_AUTH_AUDIENCE" desc:"base URL for the application"`
	WelcomeEmail EmailTemplate `split_words:"true"`

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
// EmailTemplate
// ############################################################################

// EmailTemplate allows for custom HTML and Text email template content to be
// loaded from the filesystem. Use [EmailTemplate.LoadTemplateContent] to load
// the template content into memory and then use [EmailTemplate.HTMLContent] and
// [EmailTemplate.TextContent] to get the content that was loaded from the files.
type EmailTemplate struct {
	HTMLPath string `default:"" desc:"specify the file path to a custom html template for the given email"`
	TextPath string `default:"" desc:"specify the file path to a custom text template for the given email"`

	loaded      *sync.Once
	htmlContent string
	textContent string
}

// Returns the HTML template content loaded from the template files. Use
// [EmailTemplate.LoadTemplateContent] to load the template content into memory
// first.
func (p *EmailTemplate) HTMLContent() template.HTML {
	return template.HTML(p.htmlContent)
}

// Returns the Text template content loaded from the template files. Use
// [EmailTemplate.LoadTemplateContent] to load the template content into memory
// first.
func (p *EmailTemplate) TextContent() string {
	return p.textContent
}

// Loads the welcome email template content from the provided paths. Can be used
// concurrently.
func (p *EmailTemplate) LoadTemplateContent() (err error) {
	if p.loaded == nil {
		p.loaded = &sync.Once{}
	}

	p.loaded.Do(func() {
		if p.HTMLPath != "" {
			var data []byte
			if data, err = os.ReadFile(p.HTMLPath); err != nil {
				return
			}
			p.htmlContent = string(data)
		}

		if p.TextPath != "" {
			var data []byte
			if data, err = os.ReadFile(p.TextPath); err != nil {
				return
			}
			p.textContent = string(data)
		}
	})

	if err != nil {
		return err
	}

	return nil
}
