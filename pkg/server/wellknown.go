package server

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/errors"
)

const (
	HeaderExpires      = "Expires"
	HeaderCacheControl = "Cache-Control"
	HeaderIfNoneMatch  = "If-None-Match"
	HeaderEtag         = "ETag"
)

// JWKS returns the JSON web key set for the public keys that are currently being used
// by Quarterdeck to sign JWT access and refresh tokens. External callers can use these
// keys to verify that a JWT token was in fact issued by Quarterdeck and has not been
// tampered with. This endpoint also provides Cache-Control, Expires, and ETag headers
// to allow clients to cache the keys for a period of time, based on key rotation
// periods on the Quarterdeck server.
func (s *Server) JWKS(c *gin.Context) {
	var (
		keys *auth.JWKS
		err  error
	)

	// Get the current version of the keys from the issuer.
	if keys, err = s.issuer.Keys(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(errors.ErrInternal))
		return
	}

	c.JSON(http.StatusOK, keys)
}

// Returns a JSON document with the OpenID configuration as defined by the OpenID
// Connect standard" https://connect2id.com/learn/openid-connect. This document helps
// clients understand how to authenticate with Quarterdeck.
// TODO: once OpenID endpoints have been configured add them to this JSON response
func (s *Server) OpenIDConfiguration(c *gin.Context) {
	// Parse the token issuer for the OpenID configuration
	base, err := url.Parse(s.conf.Auth.Issuer)
	if err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("openid is not configured correctly"))
		return
	}

	openid := &api.OpenIDConfiguration{
		Issuer:                        base.String(),
		AuthorizationEP:               base.ResolveReference(&url.URL{Path: "/login"}).String(),
		TokenEP:                       base.ResolveReference(&url.URL{Path: "/v1/reauthenticate"}).String(),
		JWKSURI:                       base.ResolveReference(&url.URL{Path: "/.well-known/jwks.json"}).String(),
		ScopesSupported:               []string{"openid", "profile", "email"},
		ResponseTypesSupported:        []string{"token", "id_token"},
		CodeChallengeMethodsSupported: []string{"S256", "plain"},
		ResponseModesSupported:        []string{"query", "fragment", "form_post"},
		SubjectTypesSupported:         []string{"public"},
		IDTokenSigningAlgValues:       []string{"HS256", "RS256", "EdDSA"},
		TokenEndpointAuthMethods:      []string{"client_secret_basic", "client_secret_post"},
		ClaimsSupported:               []string{"aud", "email", "exp", "iat", "iss", "sub"},
		RequestURIParameterSupported:  false,
	}

	c.JSON(http.StatusOK, openid)
}

func (s *Server) SecurityTxt(c *gin.Context) {
	// TODO: set Expires and Cache-Control headers for the security.txt file
	// TODO: ensure Content-Type is set to text/plain
	// TODO: generate the security.txt file if it does not exist
	if s.conf.Security.TxtPath == "" {
		c.Status(http.StatusNotFound)
		return
	}
	c.File(s.conf.Security.TxtPath)
}
