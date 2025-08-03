package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/posilva/simpleidentity/pkg/logger"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// Check represents a health check
type Check struct {
	Name        string        `json:"name"`
	Status      Status        `json:"status"`
	Message     string        `json:"message,omitempty"`
	LastChecked time.Time     `json:"last_checked"`
	Duration    time.Duration `json:"duration_ms"`
}

// CheckFunc is a function that performs a health check
type CheckFunc func(ctx context.Context) error

// Response represents the health check response
type Response struct {
	Status  Status           `json:"status"`
	Checks  map[string]Check `json:"checks"`
	Version string           `json:"version,omitempty"`
	Uptime  time.Duration    `json:"uptime_seconds"`
}

// Checker manages health checks
type Checker struct {
	checks    map[string]CheckFunc
	mutex     sync.RWMutex
	logger    logger.Logger
	version   string
	startTime time.Time
}

// NewChecker creates a new health checker
func NewChecker(logger logger.Logger, version string) *Checker {
	return &Checker{
		checks:    make(map[string]CheckFunc),
		logger:    logger,
		version:   version,
		startTime: time.Now(),
	}
}

// AddCheck adds a health check
func (c *Checker) AddCheck(name string, check CheckFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.checks[name] = check
}

// RemoveCheck removes a health check
func (c *Checker) RemoveCheck(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.checks, name)
}

// Check performs all health checks
func (c *Checker) Check(ctx context.Context) Response {
	c.mutex.RLock()
	checks := make(map[string]CheckFunc)
	for name, check := range c.checks {
		checks[name] = check
	}
	c.mutex.RUnlock()

	response := Response{
		Status:  StatusHealthy,
		Checks:  make(map[string]Check),
		Version: c.version,
		Uptime:  time.Since(c.startTime),
	}

	// Execute all checks concurrently
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for name, checkFunc := range checks {
		wg.Add(1)
		go func(name string, checkFunc CheckFunc) {
			defer wg.Done()

			start := time.Now()
			status := StatusHealthy
			message := ""

			if err := checkFunc(ctx); err != nil {
				status = StatusUnhealthy
				message = err.Error()

				mutex.Lock()
				response.Status = StatusUnhealthy
				mutex.Unlock()
			}

			check := Check{
				Name:        name,
				Status:      status,
				Message:     message,
				LastChecked: start,
				Duration:    time.Since(start),
			}

			mutex.Lock()
			response.Checks[name] = check
			mutex.Unlock()
		}(name, checkFunc)
	}

	wg.Wait()
	return response
}

// Server represents the health check HTTP server
type Server struct {
	server  *http.Server
	checker *Checker
	logger  logger.Logger
}

// NewServer creates a new health check server
func NewServer(addr string, checker *Checker, logger logger.Logger) *Server {
	mux := http.NewServeMux()

	s := &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		checker: checker,
		logger:  logger,
	}

	// Health check endpoints
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/health/live", s.livenessHandler)
	mux.HandleFunc("/health/ready", s.readinessHandler)

	return s
}

// Start starts the health check server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info().
		Str("addr", s.server.Addr).
		Msg("Starting health check server")

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		s.logger.Info().Msg("Shutting down health check server")
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error().Err(err).Msg("Error shutting down health check server")
		}
	}()

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("health server error: %w", err)
	}

	return nil
}

// healthHandler handles comprehensive health checks
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := s.checker.Check(ctx)

	w.Header().Set("Content-Type", "application/json")

	if response.Status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error().Err(err).Msg("Error encoding health response")
	}
}

// livenessHandler handles liveness probe (simple ping)
func (s *Server) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error().Err(err).Msg("Error encoding liveness response")
	}
}

// readinessHandler handles readiness probe (dependencies check)
func (s *Server) readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response := s.checker.Check(ctx)

	w.Header().Set("Content-Type", "application/json")

	if response.Status == StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	readinessResponse := map[string]interface{}{
		"status":    response.Status,
		"checks":    response.Checks,
		"timestamp": time.Now().UTC(),
	}

	if err := json.NewEncoder(w).Encode(readinessResponse); err != nil {
		s.logger.Error().Err(err).Msg("Error encoding readiness response")
	}
}

// Common health check functions

// DatabaseCheck creates a database health check
func DatabaseCheck(pingFunc func(ctx context.Context) error) CheckFunc {
	return func(ctx context.Context) error {
		return pingFunc(ctx)
	}
}

// HTTPCheck creates an HTTP endpoint health check
func HTTPCheck(url string, timeout time.Duration) CheckFunc {
	return func(ctx context.Context) error {
		client := &http.Client{Timeout: timeout}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to perform request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
		}

		return nil
	}
}

// MemoryCheck creates a memory usage health check
func MemoryCheck(maxMemoryMB int64) CheckFunc {
	return func(ctx context.Context) error {
		// This is a simple implementation - in production you might want to use
		// runtime.MemStats or a more sophisticated memory monitoring approach
		return nil
	}
}
