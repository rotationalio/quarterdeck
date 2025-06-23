package server

import "go.rtnl.ai/quarterdeck/pkg/config"

type Server struct{}

func New(conf config.Config) (*Server, error) {
	return nil, nil
}

func (s *Server) Serve() error {
	return nil
}
