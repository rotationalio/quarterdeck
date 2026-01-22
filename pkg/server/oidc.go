package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gimauth "go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
)

// OIDC UserInfo endpoint.
// See: https://openid.net/specs/openid-connect-core-1_0.html#UserInfo
func (s *Server) UserInfo(c *gin.Context) {
	var (
		err    error
		claims *gimauth.Claims
	)

	if claims, err = gimauth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusNotFound, api.Error("user not found"))
		return
	}

	c.JSON(http.StatusOK, api.NewUserInfo(claims))
}
