package login

import (
	"net/url"

	"github.com/gin-gonic/gin"
)

// LoginURL computes a redirect URL for the login page, adding query parameters as
// needed to more correctly process the login flow on redirect.
type URL struct {
	url *url.URL
}

func New(uri string) *URL {
	var (
		u   *URL
		err error
	)

	u = &URL{}
	if u.url, err = url.Parse(uri); err != nil {
		panic(err)
	}
	return u
}

func (l *URL) Location(c *gin.Context) string {
	loc := *l.url
	next := c.Request.URL

	if loc.Host == next.Host {
		loc.Scheme = ""
		loc.Host = ""
		next.Scheme = ""
		next.Host = ""
	}

	var query url.Values
	if loc.RawQuery != "" {
		query, _ = url.ParseQuery(loc.RawQuery)
	} else {
		query = make(url.Values)
	}

	query.Set("next", next.String())
	loc.RawQuery = query.Encode()
	return loc.String()
}
