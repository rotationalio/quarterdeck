package server_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type serverTestSuite struct {
	suite.Suite
}

func TestServer(t *testing.T) {
	suite.Run(t, &serverTestSuite{})
}
