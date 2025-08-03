package pprof

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Import pprof handlers
	"time"

	"github.com/posilva/simpleidentity/pkg/logger"
)

// Server represents the pprof debug server
type Server struct {
	server *http.Server
	logger logger.Logger
}

// NewServer creates a new pprof server
func NewServer(addr string, logger logger.Logger) *Server {
	// Use default pprof mux
	mux := http.DefaultServeMux

	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
			// Security: Add timeouts for the debug server
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
		logger: logger,
	}
}

// Start starts the pprof server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info().
		Str("addr", s.server.Addr).
		Msg("Starting pprof debug server (internal use only)")

	s.logger.Info().
		Str("endpoints", fmt.Sprintf("http://%s/debug/pprof/", s.server.Addr)).
		Msg("pprof endpoints available")

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.logger.Info().Msg("Shutting down pprof debug server")
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error().Err(err).Msg("Error shutting down pprof server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("pprof server error: %w", err)
	}

	return nil
}

// Available endpoints:
// - /debug/pprof/ - Index page with links to all profiles
// - /debug/pprof/profile - CPU profile (30 seconds by default)
// - /debug/pprof/heap - Heap profile
// - /debug/pprof/goroutine - Goroutine profile
// - /debug/pprof/block - Block profile
// - /debug/pprof/mutex - Mutex profile
// - /debug/pprof/allocs - Memory allocation profile
// - /debug/pprof/cmdline - Command line that started the program
// - /debug/pprof/symbol - Symbol table
// - /debug/pprof/trace - Execution tracer (duration can be specified with ?seconds=N)
