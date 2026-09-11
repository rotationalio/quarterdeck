package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/commo"
	"go.rtnl.ai/gimlet/csrf"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/v1"
	"go.rtnl.ai/quarterdeck/pkg/telemetry"
	"go.rtnl.ai/x/probez"
	"go.rtnl.ai/x/rlog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

const (
	ServiceName       = "quarterdeck"
	ReadHeaderTimeout = 20 * time.Second
	WriteTimeout      = 20 * time.Second
	IdleTimeout       = 180 * time.Second
	ShutdownTimeout   = 45 * time.Second
)

// The Quarterdeck server implements both a web based UI and a REST API for managing
// authentication and authorization for Rotational applications. The server also runs
// background routines that manage Quarterdeck's internal lifecycle. This is the main
// entry point for the Quarterdeck application.
type Server struct {
	sync.RWMutex
	probez.Handler
	conf    config.Config
	store   store.Store
	srv     *http.Server
	router  *gin.Engine
	issuer  *auth.Issuer
	csrf    csrf.TokenHandler
	url     *url.URL
	started time.Time
	errc    chan error
}

func New() (s *Server, err error) {
	// Create a new server instance and prepare to serve.
	s = &Server{
		errc: make(chan error, 1),
	}

	// Load the default configuration from the environment if it is not set.
	// Ensure that the global config is set when loading from the environment.
	if s.conf, err = config.Get(); err != nil {
		return nil, err
	}

	// Initialize telemetry and logging before any other initialization.
	if err = telemetry.Setup(context.Background()); err != nil {
		return nil, err
	}

	// NOTE: telemetry must be initialized before logging to ensure the otelslog
	// handler is bound to the real (or noop) LoggerProvider.
	ConfigureLogging(&s.conf)

	// If in maintenance mode, stop the server configuration (no database connections,
	// no setup of subcomponents, no gin routers or middleware) and run only a
	// maintenance server with probez routes and and a 503 response.
	if s.conf.Maintenance {
		if err = s.Maintenance(); err != nil {
			return nil, fmt.Errorf("could not initialize maintenance mode: %w", err)
		}
		return s, nil
	}

	// NOTE: anything below this line assumes the server is not in maintenance mode.

	// Initialize the commo module for email sending and load welcome email
	// template content from filesystem
	if err = commo.Initialize(s.conf.Email, emails.LoadTemplates()); err != nil {
		return nil, fmt.Errorf("could not initialize commo: %w", err)
	}

	if err = s.conf.App.WelcomeEmail.LoadTemplateContent(); err != nil {
		return nil, fmt.Errorf("could not load welcome email template content: %w", err)
	}

	// Connect to the configured database store.
	if s.store, err = store.Open(s.conf.Database); err != nil {
		return nil, fmt.Errorf("could not open database store: %w", err)
	}

	// Initialize the claims issuer for JWT tokens.
	if s.issuer, err = auth.NewIssuer(s.conf.Auth); err != nil {
		return nil, fmt.Errorf("could not initialize claims issuer: %w", err)
	}

	// Initialize the namespaced, signed CSRF token handler. Cookie scope is
	// explicitly configured; it is never inferred from CORS origins.
	var csrfHandler csrf.TokenHandler
	if csrfHandler, err = csrf.NewTokenHandlerWithNamespace(
		s.conf.CSRF.CookieTTL,
		"/",
		s.conf.CSRF.CookieDomains(),
		s.conf.CSRF.GetSecret(),
		s.conf.CSRF.Namespace,
	); err != nil {
		return nil, fmt.Errorf("could not initialize CSRF token handler: %w", err)
	}
	s.csrf = &sameSiteCSRF{TokenHandler: csrfHandler}

	// Configure the gin router
	gin.SetMode(s.conf.Mode)
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
func Debug(srv *http.Server) (s *Server, err error) {
	if s, err = New(); err != nil {
		return nil, err
	}

	// Replace the http server with the one provided
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}

func (s *Server) Serve() (err error) {
	// Catch OS signals for graceful shutdowns
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		s.errc <- s.Shutdown()
	}()

	// Catch fatal log errors and shutdown the server
	// Runs after rlog.Fatal output; hook replaces os.Exit(1), so we must exit.
	rlog.SetFatalHook(func() {
		_ = s.Shutdown()
		os.Exit(1)
	})

	// Attempt to bootstrap the server if enabled.
	if err = s.Bootstrap(); err != nil {
		return err
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
	s.Healthy()
	s.started = time.Now()

	// Listen for HTTP requests and handle them.
	go func() {
		// Make sure we don't use the external err to avoid data races.
		if serr := s.serve(sock); !errors.Is(serr, http.ErrServerClosed) {
			s.errc <- serr
		}
	}()

	s.Ready()
	rlog.InfoAttrs(context.Background(), "quarterdeck server started",
		slog.String("url", s.URL()),
		slog.Bool("maintenance", s.conf.Maintenance),
		slog.String("version", pkg.Version(false)),
		slog.String("issuer", s.conf.Auth.Issuer),
		slog.Any("audience", s.conf.Auth.Audience),
	)
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
	rlog.InfoAttrs(context.Background(), "gracefully shutting down quarterdeck server")
	s.NotReady()
	defer s.Unhealthy()

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	s.srv.SetKeepAlivesEnabled(false)
	if serr := s.srv.Shutdown(ctx); serr != nil {
		err = errors.Join(err, fmt.Errorf("could not shutdown http server: %w", serr))
	}

	if telErr := telemetry.Shutdown(ctx); telErr != nil {
		err = errors.Join(err, fmt.Errorf("could not shutdown telemetry: %w", telErr))
	}

	rlog.DebugAttrs(context.Background(), "quarterdeck server shutdown complete", slog.Any("error", err))
	return err
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

// ConfigureLogging sets the default global logger and log level based on the
// configuration. If telemetry is enabled, the logger will be configured to fan-out
// to the OpenTelemetry logger. Can be called multiple times to reconfigure the
// logger.
func ConfigureLogging(conf *config.Config) {
	rlog.SetLevel(conf.GetLogLevel())

	opts := rlog.MergeWithCustomLevels(rlog.WithGlobalLevel(nil))
	var console slog.Handler
	if conf.ConsoleLog {
		console = slog.NewTextHandler(os.Stdout, opts)
	} else {
		console = slog.NewJSONHandler(os.Stdout, opts)
	}

	var handler slog.Handler = console
	if conf.Telemetry.Enabled {
		otelHandler := otelslog.NewHandler(ServiceName, otelslog.WithLoggerProvider(telemetry.LoggerProvider()))
		handler = slog.NewMultiHandler(console, otelHandler)
	}

	rlog.SetDefault(rlog.New(slog.New(handler)))
}
