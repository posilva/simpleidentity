package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/posilva/simpleidentity/pkg/logger"
)

// Hook represents a shutdown hook function
type Hook func(ctx context.Context) error

// Manager manages graceful shutdown
type Manager struct {
	hooks   []Hook
	timeout time.Duration
	logger  logger.Logger
	mutex   sync.Mutex
}

// NewManager creates a new shutdown manager
func NewManager(timeout time.Duration, logger logger.Logger) *Manager {
	return &Manager{
		hooks:   make([]Hook, 0),
		timeout: timeout,
		logger:  logger,
	}
}

// AddHook adds a shutdown hook
func (m *Manager) AddHook(hook Hook) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.hooks = append(m.hooks, hook)
}

// Wait waits for shutdown signals and executes hooks
func (m *Manager) Wait(ctx context.Context) {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)

	// Register the channel to receive specific signals
	signal.Notify(sigChan,
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // Termination signal
		syscall.SIGQUIT, // Quit signal
	)

	select {
	case sig := <-sigChan:
		m.logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal")
		m.shutdown()

	case <-ctx.Done():
		m.logger.Info().Msg("Context cancelled, initiating shutdown")
		m.shutdown()
	}
}

// Shutdown executes all shutdown hooks
func (m *Manager) shutdown() {
	m.logger.Info().
		Dur("timeout", m.timeout).
		Msg("Starting graceful shutdown")

	// Create a context with timeout for shutdown operations
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	// Execute hooks in reverse order (LIFO)
	m.mutex.Lock()
	hooks := make([]Hook, len(m.hooks))
	copy(hooks, m.hooks)
	m.mutex.Unlock()

	var wg sync.WaitGroup
	errors := make(chan error, len(hooks))

	// Execute all hooks concurrently
	for i := len(hooks) - 1; i >= 0; i-- {
		wg.Add(1)
		go func(hook Hook, index int) {
			defer wg.Done()

			hookCtx, hookCancel := context.WithTimeout(ctx, m.timeout/2)
			defer hookCancel()

			m.logger.Debug().
				Int("hook_index", index).
				Msg("Executing shutdown hook")

			if err := hook(hookCtx); err != nil {
				m.logger.Error().
					Err(err).
					Int("hook_index", index).
					Msg("Shutdown hook failed")
				errors <- err
			} else {
				m.logger.Debug().
					Int("hook_index", index).
					Msg("Shutdown hook completed successfully")
			}
		}(hooks[i], i)
	}

	// Wait for all hooks to complete or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		m.logger.Info().Msg("All shutdown hooks completed")
	case <-ctx.Done():
		m.logger.Warn().Msg("Shutdown timeout reached, forcing exit")
	}

	// Collect any errors
	close(errors)
	var shutdownErrors []error
	for err := range errors {
		shutdownErrors = append(shutdownErrors, err)
	}

	if len(shutdownErrors) > 0 {
		m.logger.Error().
			Int("error_count", len(shutdownErrors)).
			Msg("Some shutdown hooks failed")
		for _, err := range shutdownErrors {
			m.logger.Error().Err(err).Msg("Shutdown error")
		}
		os.Exit(1)
	}

	m.logger.Info().Msg("Graceful shutdown completed")
	os.Exit(0)
}

// ServerShutdownHook creates a shutdown hook for HTTP servers
func ServerShutdownHook(server interface{ Shutdown(context.Context) error }, name string) Hook {
	return func(ctx context.Context) error {
		return server.Shutdown(ctx)
	}
}

// ContextCancelHook creates a shutdown hook that cancels a context
func ContextCancelHook(cancel context.CancelFunc, name string) Hook {
	return func(ctx context.Context) error {
		cancel()
		return nil
	}
}

// DatabaseCloseHook creates a shutdown hook for database connections
func DatabaseCloseHook(closer interface{ Close() error }, name string) Hook {
	return func(ctx context.Context) error {
		return closer.Close()
	}
}

// CustomHook creates a custom shutdown hook
func CustomHook(name string, fn func(context.Context) error) Hook {
	return fn
}
