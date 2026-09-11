package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/csrf"
)

// Adds the explicit defense-in-depth cookie policy that Gimlet's cookie setter
// does not currently expose. Cookie and header names remain owned by the Gimlet
// namespaced handler.
// TODO: Replace this adapter with Gimlet's SameSite cookie option when available.
type sameSiteCSRF struct {
	csrf.TokenHandler
}

// The namespace is delegated to the underlying Gimlet handler so all callers
// use the same generated cookie and header names.
func (h *sameSiteCSRF) Namespace() csrf.Namespace {
	// HACK: gimlet/csrf has no namespace helper, so we create a throwaway
	// handler to get the names for now.
	// TODO: add `csrf.NewNamespace(namespace string) csrf.Namespace` to gimlet then use here.
	return h.TokenHandler.(csrf.Namespacer).Namespace()
}

// Sets the Gimlet cookie pair and applies the explicit SameSite policy without
// changing the names, values, or other security attributes.
func (h *sameSiteCSRF) SetDoubleCookieToken(c *gin.Context) error {
	setCookies := c.Writer.Header().Values("Set-Cookie")
	before := len(setCookies)
	if err := h.TokenHandler.SetDoubleCookieToken(c); err != nil {
		return err
	}

	setCookies = c.Writer.Header().Values("Set-Cookie")
	for i := before; i < len(setCookies); i++ {
		if isLocalHTTP(c) {
			setCookies[i] = withoutCookieAttribute(setCookies[i], "Secure")
		}
		if !strings.Contains(strings.ToLower(setCookies[i]), "samesite=") {
			setCookies[i] += "; SameSite=Lax"
		}
	}
	c.Writer.Header()["Set-Cookie"] = setCookies
	return nil
}

// withoutCookieAttribute removes one standalone cookie attribute while
// preserving the rest of the Set-Cookie value emitted by Gimlet.
func withoutCookieAttribute(setCookie, attribute string) string {
	parts := strings.Split(setCookie, ";")
	filtered := parts[:1]
	for _, part := range parts[1:] {
		if !strings.EqualFold(strings.TrimSpace(part), attribute) {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, ";")
}

// Local development uses plain HTTP, so Secure cookies would be rejected by
// the browser. Production HTTPS requests and non-local hosts retain Secure.
func isLocalHTTP(c *gin.Context) bool {
	if c.Request.TLS != nil || c.Request.URL.Scheme == "https" {
		return false
	}

	host := c.Request.Host
	if hostname, _, err := net.SplitHostPort(host); err == nil {
		host = hostname
	} else {
		host = strings.Trim(host, "[]")
	}

	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local")
}

// Applies the namespaced double-submit check and also requires the
// JavaScript-readable token cookie to be present on unsafe requests.
func csrfProtection(verifier csrf.TokenHandler) gin.HandlerFunc {
	names := verifier.(csrf.Namespacer).Namespace()
	verify := csrf.DoubleCookie(verifier)
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
		default:
			if _, err := c.Cookie(names.Cookie); err != nil {
				c.Header(names.ErrorHeader, csrf.ErrorTokenInvalid)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if _, err := url.QueryUnescape(c.GetHeader(names.Header)); err != nil {
				c.Header(names.ErrorHeader, csrf.ErrorTokenInvalid)
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			verify(c)
		}
	}
}

// Builds the mutation and page middleware using the configured CSRF policy.
func (s *Server) csrfHandlers() (csrfMiddleware gin.HandlerFunc, csrfPage gin.HandlerFunc) {
	csrfCheck := csrfProtection(s.csrf)
	csrfMiddleware = func(c *gin.Context) {
		if s.conf.CSRF.Disabled {
			c.Next()
			return
		}
		csrfCheck(c)
	}
	csrfPage = func(c *gin.Context) {
		if err := s.setCSRFToken(c); err != nil {
			s.Error(c, err)
			return
		}
		c.Next()
	}
	return csrfMiddleware, csrfPage
}

// Sets the CSRF cookies unless the explicit configuration opt-out is active.
func (s *Server) setCSRFToken(c *gin.Context) error {
	if s.conf.CSRF.Disabled {
		return nil
	}
	return s.csrf.SetDoubleCookieToken(c)
}

// Bootstraps the namespaced double-submit cookies for direct browser clients.
// Existing cookies are preserved so visiting this endpoint does not invalidate
// a token already held by the browser.
func (s *Server) CSRFToken(c *gin.Context) {
	if s.conf.CSRF.Disabled {
		c.Data(http.StatusNoContent, "", nil)
		return
	}

	names := s.csrf.(csrf.Namespacer).Namespace()
	_, tokenErr := c.Cookie(names.Cookie)
	_, referenceErr := c.Cookie(names.ReferenceCookie)
	if tokenErr != nil || referenceErr != nil {
		if err := s.csrf.SetDoubleCookieToken(c); err != nil {
			s.Error(c, err)
			return
		}
	}

	c.Data(http.StatusNoContent, "", nil)
}
