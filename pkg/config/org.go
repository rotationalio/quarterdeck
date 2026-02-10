package config

import (
	"net/url"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

// Configures the details of the organization that is utilizing Quarterdeck for auth.
type OrgConfig struct {
	Name          string `split_words:"true" default:"Rotational Labs" desc:"the name of the organization"`
	StreetAddress string `split_words:"true" default:"202 N Cedar Ave, Suite #1, Owatonna, MN 55060, USA" desc:"the street address for the organization"`
	HomepageURI   string `split_words:"true" default:"https://www.rotational.io" desc:"the homepage URI for the organization"`
	SupportEmail  string `split_words:"true" default:"support@rotational.io" desc:"an email address that a user may email for technical support"`
}

func (c OrgConfig) Validate() (err error) {
	if _, err := url.Parse(c.HomepageURI); err != nil {
		return errors.ConfigError(err, errors.InvalidConfig("orgConfig", "homepageURI", "url '%s' is unparseable", c.HomepageURI))
	}

	return nil
}

func (c OrgConfig) HomepageURL() *url.URL {
	// Ignore errors because we have already validated the config
	u, _ := url.Parse(c.HomepageURI)
	return u
}
