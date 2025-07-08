package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	jose "github.com/go-jose/go-jose/v4"

	"go.rtnl.ai/quarterdeck/pkg/api/v1"
)

// JWKS returns the JSON web key set for the public keys that are currently being used
// by Quarterdeck to sign JWT access and refresh tokens. External callers can use these
// keys to verify that a JWT token was in fact issued by Quarterdeck and has not been
// tampered with.
func (s *Server) JWKS(c *gin.Context) {
	// TODO: add Cache-Control or Expires header to the response.
	var (
		keys jose.JSONWebKeySet
		err  error
	)

	if keys, err = s.issuer.Keys(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(http.StatusText(http.StatusInternalServerError)))
		return
	}

	c.JSON(http.StatusOK, keys)
}

// Returns a JSON document with the OpenID configuration as defined by the OpenID
// Connect standard" https://connect2id.com/learn/openid-connect. This document helps
// clients understand how to authenticate with Quarterdeck.
// TODO: once OpenID endpoints have been configured add them to this JSON response
func (s *Server) OpenIDConfiguration(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("todo"))
}

func (s *Server) SecurityTxt(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("todo"))
}
