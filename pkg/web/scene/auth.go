package scene

import (
	"net/url"

	"github.com/gin-gonic/gin"
)

type LoginScene struct {
	Scene
	LoginURL          string
	ForgotPasswordURL string
	Next              string
}

func (s Scene) Login(c *gin.Context) *LoginScene {
	// Default to the issuer URL
	forgotPasswordURL := issuerForgotPasswordURL

	// If the origin host is different, use its host for the URL
	if origin := c.Request.Header.Get("Origin"); origin != "" {
		if originURL, err := url.Parse(origin); err == nil {
			forgotPasswordURL = originURL.ResolveReference(&url.URL{Path: forgotPasswordURL.Path})
		}
	}

	// Return the login scene with the URLs set.
	return &LoginScene{
		Scene:             s,
		LoginURL:          loginURL.String(),
		ForgotPasswordURL: forgotPasswordURL.String(),
		Next:              c.Query("next"),
	}
}

// Adds the URL to the "forgot password" page to the [Scene] and returns it.
func (s Scene) WithForgotPasswordURL() Scene {
	return s.With("ForgotPasswordURL", issuerForgotPasswordURL)
}
