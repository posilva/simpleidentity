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
	"github.com/posilva/simpleidentity/pkg/telemetry"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the SimpleIdentity server",
	Long: `Start the SimpleIdentity server with all required services:

- Main API server (gRPC and HTTP)
- Health check server for Kubernetes probes
- pprof debug server for observability (internal only)
- OpenTelemetry for metrics and tracing
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

	// OpenTelemetry configuration flags
	serverCmd.Flags().Bool("telemetry-enabled", false, "Enable OpenTelemetry")
	serverCmd.Flags().String("telemetry-environment", "development", "Environment name for telemetry")
	
	// Tracing configuration
	serverCmd.Flags().Bool("tracing-enabled", false, "Enable distributed tracing")
	serverCmd.Flags().String("tracing-endpoint", "localhost:4318", "OTLP tracing endpoint")
	serverCmd.Flags().String("tracing-protocol", "http", "Tracing protocol (http or grpc)")
	serverCmd.Flags().String("tracing-sampler", "always", "Tracing sampler (always, never, ratio)")
	serverCmd.Flags().Float64("tracing-sample-rate", 1.0, "Tracing sample rate (0.0-1.0)")
	
	// Metrics configuration
	serverCmd.Flags().Bool("metrics-enabled", false, "Enable metrics collection")
	serverCmd.Flags().String("metrics-endpoint", "localhost:4318", "OTLP metrics endpoint")
	serverCmd.Flags().String("metrics-protocol", "http", "Metrics protocol (http or grpc)")
	serverCmd.Flags().Duration("metrics-interval", 30*time.Second, "Metrics collection interval")
	
	// OTLP configuration
	serverCmd.Flags().Bool("otlp-insecure", true, "Use insecure OTLP connection")
	serverCmd.Flags().Duration("otlp-timeout", 10*time.Second, "OTLP connection timeout")
	serverCmd.Flags().String("otlp-compression", "gzip", "OTLP compression (gzip or none)")

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

	// Initialize OpenTelemetry if enabled
	var telemetryProvider *telemetry.Provider
	if viper.GetBool("telemetry-enabled") {
		telemetryConfig := telemetry.Config{
			ServiceName:       "simpleidentity",
			ServiceVersion:    viper.GetString("version"),
			Environment:       viper.GetString("telemetry-environment"),
			TracingEnabled:    viper.GetBool("tracing-enabled"),
			TracingEndpoint:   viper.GetString("tracing-endpoint"),
			TracingProtocol:   viper.GetString("tracing-protocol"),
			TracingSampler:    viper.GetString("tracing-sampler"),
			TracingSampleRate: viper.GetFloat64("tracing-sample-rate"),
			MetricsEnabled:    viper.GetBool("metrics-enabled"),
			MetricsEndpoint:   viper.GetString("metrics-endpoint"),
			MetricsProtocol:   viper.GetString("metrics-protocol"),
			MetricsInterval:   viper.GetDuration("metrics-interval"),
			Insecure:          viper.GetBool("otlp-insecure"),
			Timeout:           viper.GetDuration("otlp-timeout"),
			Compression:       viper.GetString("otlp-compression"),
		}

		var err error
		telemetryProvider, err = telemetry.NewProvider(telemetryConfig, log)
		if err != nil {
			return fmt.Errorf("failed to initialize telemetry: %w", err)
		}

		// Add telemetry shutdown hook
		shutdownMgr.AddHook(shutdown.CustomHook("telemetry", telemetryProvider.Shutdown))
	}

	// Initialize health checker
	version := viper.GetString("version")
	healthChecker := health.NewChecker(log, version)

	// Add basic health checks
	healthChecker.AddCheck("self", func(ctx context.Context) error {
		return nil // Always healthy for now
	})

	// Add telemetry health check if enabled
	if telemetryProvider != nil {
		healthChecker.AddCheck("telemetry", telemetryProvider.HealthCheck)
	}

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
