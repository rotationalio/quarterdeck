package redirect

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

// Computes a redirect URL for the login page by taking the URL requested by the user
// and adding it as a ?next= query parameter to the login URL. Like origin, it strips
// scheme and host if the requested URL matches the login URL; unlike origin, it
// maintains the path and query params for the login URL and adds additional params.
type LoginURL struct {
	uri *url.URL
}

// Create a new LoginURL for determining the redirect location when incoming requests
// need to be authenticated. The redirect will have a next= query parameter with the
// request intended URL normalized to the host and scheme.
func Login(loginURL string) (_ *LoginURL, err error) {
	var uri *url.URL
	if uri, err = url.Parse(loginURL); err != nil {
		return nil, err
	}

	if uri.Host == "" || uri.Scheme == "" {
		return nil, fmt.Errorf("invalid login url: %q", loginURL)
	}

	return &LoginURL{uri: &url.URL{
		Scheme:   uri.Scheme,
		Host:     uri.Host,
		Path:     uri.Path,
		RawQuery: uri.RawQuery,
	}}, nil
}

func MustLogin(loginURL string) *LoginURL {
	login, err := Login(loginURL)
	if err != nil {
		panic(err)
	}
	return login
}

func (l *LoginURL) Location(req any) (_ string, err error) {
	// Create a copy of the originally requested URL based on the input
	var next url.URL
	switch r := req.(type) {
	case *gin.Context:
		next = *r.Request.URL
	case http.Request:
		next = *r.URL
	case *http.Request:
		next = *r.URL
	case *url.URL:
		next = *r
	case url.URL:
		next = r
	case string:
		var nextURL *url.URL
		if nextURL, err = url.Parse(r); err != nil {
			return "", fmt.Errorf("invalid URL string: %q", r)
		}
		next = *nextURL
	default:
		return "", fmt.Errorf("unsupported type %T", r)
	}

	// Create a copy of the internal URL so it is not modified
	loc := *l.uri

	// If the hosts match, strip the scheme and host from both URLs
	if loc.Host == next.Host {
		loc.Scheme = ""
		loc.Host = ""
		next.Scheme = ""
		next.Host = ""
	}

	// Maintain the original query parameters on the loginURL, adding a next= parameter
	var query url.Values
	if loc.RawQuery != "" {
		if query, err = url.ParseQuery(loc.RawQuery); err != nil {
			return "", err
		}
	} else {
		query = make(url.Values)
	}

	query.Set("next", next.String())
	loc.RawQuery = query.Encode()
	return loc.String(), nil
}

func (l *LoginURL) MustLocation(req any) string {
	loc, err := l.Location(req)
	if err != nil {
		panic(err)
	}
	return loc
}

func (l *LoginURL) String() string {
	return l.uri.String()
}
