package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/commo"
	"go.rtnl.ai/gimlet/csrf"
	"go.rtnl.ai/gimlet/logger"
	"go.rtnl.ai/gimlet/o11y"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store"
)

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFieldName = logger.GCPFieldKeyTime
	zerolog.MessageFieldName = logger.GCPFieldKeyMsg

	// Add the severity hook for GCP logging
	var gcpHook logger.SeverityHook
	log.Logger = zerolog.New(os.Stdout).Hook(gcpHook).With().Timestamp().Logger()
}

const (
	ServiceName       = "quarterdeck"
	ReadHeaderTimeout = 20 * time.Second
	WriteTimeout      = 20 * time.Second
	IdleTimeout       = 120 * time.Second
)

// The Quarterdeck server implements both a web based UI and a REST API for managing
// authentication and authorization for Rotational applications. The server also runs
// background routines that manage Quarterdeck's internal lifecycle. This is the main
// entry point for the Quarterdeck application.
type Server struct {
	sync.RWMutex
	conf    config.Config
	store   store.Store
	srv     *http.Server
	router  *gin.Engine
	issuer  *auth.Issuer
	csrf    csrf.TokenHandler
	url     *url.URL
	started time.Time
	errc    chan error
	healthy bool
	ready   bool
}

func New(conf *config.Config) (s *Server, err error) {
	// Create a new server instance and prepare to serve.
	s = &Server{
		errc: make(chan error, 1),
	}

	if conf == nil {
		// Load the default configuration from the environment if it is not set.
		// Ensure that the global config is set when loading from the environment.
		if s.conf, err = config.Get(); err != nil {
			return nil, err
		}
	} else {
		// Set the global configuration from the user-specificed config.
		s.conf = *conf
		if err = config.Set(s.conf); err != nil {
			return nil, err
		}
	}

	// Set the global level
	zerolog.SetGlobalLevel(conf.GetLogLevel())

	// Set human readable logging if configured
	if conf.ConsoleLog {
		console := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		log.Logger = zerolog.New(console).With().Timestamp().Logger()
	}

	// Initialize the commo module for email sending and load welcome email
	// template content from filesystem
	if err = commo.Initialize(s.conf.Email, emails.LoadTemplates()); err != nil {
		return nil, err
	}

	if err = s.conf.App.WelcomeEmail.LoadTemplateContent(); err != nil {
		return nil, err
	}

	// Connect to the configured database store.
	if s.store, err = store.Open(conf.Database); err != nil {
		return nil, err
	}

	// Initialize the claims issuer for JWT tokens.
	if s.issuer, err = auth.NewIssuer(conf.Auth); err != nil {
		return nil, err
	}

	// Initialize the CSRF token handler if enabled.
	if s.csrf, err = csrf.NewTokenHandler(s.conf.CSRF.CookieTTL, "/", s.conf.CookieDomains(), s.conf.CSRF.GetSecret()); err != nil {
		return nil, err
	}

	// Configure the gin router
	gin.SetMode(conf.Mode)
	s.router = gin.New()
	s.router.RedirectTrailingSlash = true
	s.router.RedirectFixedPath = false
	s.router.HandleMethodNotAllowed = true
	s.router.ForwardedByClientIP = true
	s.router.UseRawPath = false
	s.router.UnescapePathValues = true
	if err = s.setupRoutes(); err != nil {
		return nil, err
	}

	// Create the http server if enabled
	s.srv = &http.Server{
		Addr:              s.conf.BindAddr,
		Handler:           s.router,
		ErrorLog:          nil,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}

	return s, nil
}

// Debug returns a server that uses the specified http server instead of creating one.
// This is primarily used to create test servers that can be used in unit tests.
func Debug(conf *config.Config, srv *http.Server) (s *Server, err error) {
	if s, err = New(conf); err != nil {
		return nil, err
	}

	// Replace the http server with the one provided
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}

func (s *Server) Serve() (err error) {
	// If we're not in maintenance mode; connect to database and prepare the service.
	if !s.conf.Maintenance {
		// Register prometheus metrics (ok to call multiple times)
		if err = o11y.Setup(); err != nil {
			return err
		}
	}

	// Create a socket to listen on and infer the final URL.
	// NOTE: if the bindaddr is 127.0.0.1:0 for testing, a random port will be assigned,
	// manually creating the listener will allow us to determine which port.
	// When we start listening all incoming requests will be buffered until the server
	// actually starts up in its own go routine below.
	var sock net.Listener
	if sock, err = net.Listen("tcp", s.srv.Addr); err != nil {
		return errors.Fmt("could not listen on bind addr %s: %s", s.srv.Addr, err)
	}

	s.setURL(sock.Addr())
	s.SetStatus(true, true)
	s.started = time.Now()

	// Listen for HTTP requests and handle them.
	go func() {
		// Make sure we don't use the external err to avoid data races.
		if serr := s.serve(sock); !errors.Is(serr, http.ErrServerClosed) {
			s.errc <- serr
		}
	}()

	log.Info().
		Str("url", s.URL()).
		Bool("maintenance", s.conf.Maintenance).
		Str("version", pkg.Version(false)).
		Str("issuer", s.conf.Auth.Issuer).
		Strs("audience", s.conf.Auth.Audience).
		Msg("quarterdeck server started")
	return <-s.errc
}

// ServeTLS if a tls configuration is provided, otherwise Serve.
func (s *Server) serve(sock net.Listener) error {
	if s.srv.TLSConfig != nil {
		return s.srv.ServeTLS(sock, "", "")
	}
	return s.srv.Serve(sock)
}

// Shutdown the web server gracefully.
func (s *Server) Shutdown() (err error) {
	log.Info().Msg("gracefully shutting down quarterdeck server")
	s.SetStatus(false, false)

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	s.srv.SetKeepAlivesEnabled(false)
	if err = s.srv.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// SetStatus sets the health and ready status on the server, modifying the behavior of
// the kubernetes probe responses.
func (s *Server) SetStatus(health, ready bool) {
	s.Lock()
	s.healthy = health
	s.ready = ready
	s.Unlock()
	log.Debug().Bool("health", health).Bool("ready", ready).Msg("server status set")
}

// URL returns the endpoint of the server as determined by the configuration and the
// socket address and port (if specified).
func (s *Server) URL() string {
	s.RLock()
	defer s.RUnlock()
	return s.url.String()
}

func (s *Server) setURL(addr net.Addr) {
	s.Lock()
	defer s.Unlock()

	s.url = &url.URL{
		Scheme: "http",
		Host:   addr.String(),
	}

	if s.srv.TLSConfig != nil {
		s.url.Scheme = "https"
		if s.srv.TLSConfig.ServerName != "" {
			s.url.Host = s.srv.TLSConfig.ServerName
		}
	}

	if tcp, ok := addr.(*net.TCPAddr); ok && tcp.IP.IsUnspecified() && s.url.Host == "" {
		s.url.Host = fmt.Sprintf("127.0.0.1:%d", tcp.Port)
	}
}
