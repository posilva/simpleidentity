package cmd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/posilva/simpleidentity/pkg/health"
	"github.com/posilva/simpleidentity/pkg/logger"
	"github.com/posilva/simpleidentity/pkg/pprof"
	"github.com/posilva/simpleidentity/pkg/shutdown"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the SimpleIdentity server",
	Long: `Start the SimpleIdentity server with all required services:

- Main API server (gRPC and HTTP)
- Health check server for Kubernetes probes
- pprof debug server for observability (internal only)
- Graceful shutdown handling

The server follows 12-factor app principles and can be configured
entirely through environment variables.`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// Server configuration flags
	serverCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")
	serverCmd.Flags().Bool("log-pretty", false, "Enable pretty logging for development")
	serverCmd.Flags().String("health-addr", ":8080", "Health check server address")
	serverCmd.Flags().String("pprof-addr", ":6060", "pprof debug server address")
	serverCmd.Flags().String("grpc-addr", ":9090", "gRPC server address")
	serverCmd.Flags().String("http-addr", ":8090", "HTTP server address")
	serverCmd.Flags().Duration("shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")
	serverCmd.Flags().String("version", "dev", "Service version")

	// Bind flags to viper for environment variable support
	viper.BindPFlags(serverCmd.Flags())
}

func runServer(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logLevel := viper.GetString("log-level")
	logPretty := viper.GetBool("log-pretty")
	log := logger.New(logLevel, logPretty)

	log.Info().
		Str("version", viper.GetString("version")).
		Str("log_level", logLevel).
		Msg("Starting SimpleIdentity server")

	// Create contexts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize shutdown manager
	shutdownTimeout := viper.GetDuration("shutdown-timeout")
	shutdownMgr := shutdown.NewManager(shutdownTimeout, log)

	// Initialize health checker
	version := viper.GetString("version")
	healthChecker := health.NewChecker(log, version)

	// Add basic health checks
	healthChecker.AddCheck("self", func(ctx context.Context) error {
		return nil // Always healthy for now
	})

	// Create servers
	healthAddr := viper.GetString("health-addr")
	healthServer := health.NewServer(healthAddr, healthChecker, log)

	pprofAddr := viper.GetString("pprof-addr")
	pprofServer := pprof.NewServer(pprofAddr, log)

	// Start servers concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// Start health server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := healthServer.Start(ctx); err != nil {
			errChan <- fmt.Errorf("health server error: %w", err)
		}
	}()

	// Start pprof server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pprofServer.Start(ctx); err != nil {
			errChan <- fmt.Errorf("pprof server error: %w", err)
		}
	}()

	// TODO: Start main application servers (gRPC, HTTP)
	// This will be implemented when we add the actual API handlers
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("Main application servers will be started here")
		// For now, just wait for context cancellation
		<-ctx.Done()
	}()

	// Add shutdown hooks
	shutdownMgr.AddHook(shutdown.ContextCancelHook(cancel, "main-context"))

	log.Info().
		Str("health_addr", healthAddr).
		Str("pprof_addr", pprofAddr).
		Msg("All servers started successfully")

	// Wait for shutdown signal or server errors
	go func() {
		for err := range errChan {
			log.Error().Err(err).Msg("Server error occurred")
			cancel()
			return
		}
	}()

	// Wait for shutdown
	shutdownMgr.Wait(ctx)

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	return nil
}
