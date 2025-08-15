package redirect

import (
	"fmt"
	"net/url"
)

// A redirect Origin computes the location for a redirect response based on the origin
// of the server serving the request; excluding non-path components of the URL if the
// host and scheme match, otherwise resolving to a full absolute URL.
//
// E.g. if the origin is https://auth.example.com and the redirect is to
// https://auth.example.com/login then the redirect location will be /login. However if
// the redirect is to https://example.com/login then the location will be the full url.
type Origin struct {
	uri *url.URL
}

// Create a new origin for determining redirect locations. The origin should be a valid
// URL with a scheme and a hostname (and optionally a port). Any path components will
// be stripped from the origin so this cannot be used for relative redirects.
func New(origin string) (o *Origin, err error) {
	var uri *url.URL
	if uri, err = url.Parse(origin); err != nil {
		return nil, err
	}

	if uri.Host == "" || uri.Scheme == "" {
		return nil, fmt.Errorf("invalid origin: %q", origin)
	}

	return &Origin{uri: &url.URL{
		Scheme: uri.Scheme,
		Host:   uri.Host,
	}}, nil
}

// Create a new origin for determining redirect locations; panic if origin is invalid.
func MustNew(origin string) *Origin {
	o, err := New(origin)
	if err != nil {
		panic(err)
	}
	return o
}

// Location returns the resolved redirect location. If the next URL shares the same
// host and scheme as the origin, the location will be a path relative to the origin.
// Otherwise, the location will be an absolute URL.
func (o *Origin) Location(next any) (_ string, err error) {
	var to url.URL
	switch n := next.(type) {
	case *url.URL:
		to = *n
	case url.URL:
		to = n
	case string:
		var parsed *url.URL
		if parsed, err = url.Parse(n); err != nil {
			return "", err
		}
		to = *parsed
	default:
		return "", fmt.Errorf("unsupported type %T", n)
	}

	if to.Scheme == o.uri.Scheme && to.Host == o.uri.Host {
		to.Scheme = ""
		to.Host = ""
	}

	return to.String(), nil
}

// MustLocation returns the resolved redirect location. Panics if there is an error.
func (o *Origin) MustLocation(next any) string {
	loc, err := o.Location(next)
	if err != nil {
		panic(err)
	}
	return loc
}

// String returns the origin url
func (o *Origin) String() string {
	return o.uri.String()
}
