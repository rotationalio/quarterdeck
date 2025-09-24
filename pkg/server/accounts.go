package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
)

func (s *Server) ListAccounts(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) CreateAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) AccountDetail(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) UpdateAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}

func (s *Server) DeleteAccount(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("this endpoint not implemented yet"))
}
