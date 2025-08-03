package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/posilva/simpleidentity/pkg/logger"
)

// Config holds the OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	
	// Tracing configuration
	TracingEnabled   bool
	TracingEndpoint  string
	TracingProtocol  string // "http" or "grpc"
	TracingHeaders   map[string]string
	TracingSampler   string // "always", "never", "ratio"
	TracingSampleRate float64
	
	// Metrics configuration
	MetricsEnabled   bool
	MetricsEndpoint  string
	MetricsProtocol  string // "http" or "grpc"
	MetricsHeaders   map[string]string
	MetricsInterval  time.Duration
	
	// Common configuration
	Insecure    bool
	Timeout     time.Duration
	Compression string // "gzip" or ""
}

// Provider manages OpenTelemetry providers
type Provider struct {
	config          Config
	logger          logger.Logger
	tracerProvider  *sdktrace.TracerProvider
	meterProvider   *sdkmetric.MeterProvider
	shutdown        []func(context.Context) error
}

// NewProvider creates a new OpenTelemetry provider
func NewProvider(config Config, logger logger.Logger) (*Provider, error) {
	p := &Provider{
		config:   config,
		logger:   logger,
		shutdown: make([]func(context.Context) error, 0),
	}

	// Create resource
	res, err := p.createResource()
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize tracing
	if config.TracingEnabled {
		if err := p.initTracing(res); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
	}

	// Initialize metrics
	if config.MetricsEnabled {
		if err := p.initMetrics(res); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	// Set up propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	p.logger.Info().
		Str("service", config.ServiceName).
		Str("version", config.ServiceVersion).
		Bool("tracing_enabled", config.TracingEnabled).
		Bool("metrics_enabled", config.MetricsEnabled).
		Msg("OpenTelemetry initialized")

	return p, nil
}

// createResource creates a resource with service information
func (p *Provider) createResource() (*resource.Resource, error) {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(p.config.ServiceName),
		semconv.ServiceVersion(p.config.ServiceVersion),
		semconv.DeploymentEnvironment(p.config.Environment),
	), nil
}

// initTracing initializes the tracing provider
func (p *Provider) initTracing(res *resource.Resource) error {
	// Create exporter based on protocol
	var exporter sdktrace.SpanExporter
	var err error

	switch p.config.TracingProtocol {
	case "grpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(p.config.TracingEndpoint),
			otlptracegrpc.WithTimeout(p.config.Timeout),
		}
		
		if p.config.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		
		if p.config.Compression != "" {
			opts = append(opts, otlptracegrpc.WithCompressor(p.config.Compression))
		}
		
		if len(p.config.TracingHeaders) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(p.config.TracingHeaders))
		}
		
		exporter, err = otlptracegrpc.New(context.Background(), opts...)
		
	case "http":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(p.config.TracingEndpoint),
			otlptracehttp.WithTimeout(p.config.Timeout),
		}
		
		if p.config.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		
		if p.config.Compression != "" && p.config.Compression != "none" {
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		}
		
		if len(p.config.TracingHeaders) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(p.config.TracingHeaders))
		}
		
		exporter, err = otlptracehttp.New(context.Background(), opts...)
		
	default:
		return fmt.Errorf("unsupported tracing protocol: %s", p.config.TracingProtocol)
	}

	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create sampler
	var sampler sdktrace.Sampler
	switch p.config.TracingSampler {
	case "always":
		sampler = sdktrace.AlwaysSample()
	case "never":
		sampler = sdktrace.NeverSample()
	case "ratio":
		sampler = sdktrace.TraceIDRatioBased(p.config.TracingSampleRate)
	default:
		sampler = sdktrace.AlwaysSample()
	}

	// Create tracer provider
	p.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(p.tracerProvider)

	// Add shutdown function
	p.shutdown = append(p.shutdown, p.tracerProvider.Shutdown)

	p.logger.Info().
		Str("endpoint", p.config.TracingEndpoint).
		Str("protocol", p.config.TracingProtocol).
		Str("sampler", p.config.TracingSampler).
		Msg("Tracing initialized")

	return nil
}

// initMetrics initializes the metrics provider
func (p *Provider) initMetrics(res *resource.Resource) error {
	// Create exporter based on protocol
	var exporter sdkmetric.Exporter
	var err error

	switch p.config.MetricsProtocol {
	case "grpc":
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(p.config.MetricsEndpoint),
			otlpmetricgrpc.WithTimeout(p.config.Timeout),
		}
		
		if p.config.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		
		if p.config.Compression != "" {
			opts = append(opts, otlpmetricgrpc.WithCompressor(p.config.Compression))
		}
		
		if len(p.config.MetricsHeaders) > 0 {
			opts = append(opts, otlpmetricgrpc.WithHeaders(p.config.MetricsHeaders))
		}
		
		exporter, err = otlpmetricgrpc.New(context.Background(), opts...)
		
	case "http":
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(p.config.MetricsEndpoint),
			otlpmetrichttp.WithTimeout(p.config.Timeout),
		}
		
		if p.config.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		
		if p.config.Compression != "" && p.config.Compression != "none" {
			opts = append(opts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
		}
		
		if len(p.config.MetricsHeaders) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(p.config.MetricsHeaders))
		}
		
		exporter, err = otlpmetrichttp.New(context.Background(), opts...)
		
	default:
		return fmt.Errorf("unsupported metrics protocol: %s", p.config.MetricsProtocol)
	}

	if err != nil {
		return fmt.Errorf("failed to create metrics exporter: %w", err)
	}

	// Create meter provider
	p.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(
			exporter,
			sdkmetric.WithInterval(p.config.MetricsInterval),
		)),
	)

	// Set global meter provider
	otel.SetMeterProvider(p.meterProvider)

	// Add shutdown function
	p.shutdown = append(p.shutdown, p.meterProvider.Shutdown)

	p.logger.Info().
		Str("endpoint", p.config.MetricsEndpoint).
		Str("protocol", p.config.MetricsProtocol).
		Dur("interval", p.config.MetricsInterval).
		Msg("Metrics initialized")

	return nil
}

// Tracer returns a tracer for the given name
func (p *Provider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return otel.Tracer(name, opts...)
}

// Meter returns a meter for the given name
func (p *Provider) Meter(name string, opts ...metric.MeterOption) metric.Meter {
	return otel.Meter(name, opts...)
}

// TracerProvider returns the tracer provider
func (p *Provider) TracerProvider() trace.TracerProvider {
	return p.tracerProvider
}

// MeterProvider returns the meter provider
func (p *Provider) MeterProvider() metric.MeterProvider {
	return p.meterProvider
}

// Shutdown gracefully shuts down the telemetry providers
func (p *Provider) Shutdown(ctx context.Context) error {
	var errors []error
	
	for i := len(p.shutdown) - 1; i >= 0; i-- {
		if err := p.shutdown[i](ctx); err != nil {
			errors = append(errors, err)
			p.logger.Error().Err(err).Msg("Error shutting down telemetry component")
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("telemetry shutdown errors: %v", errors)
	}
	
	p.logger.Info().Msg("OpenTelemetry shutdown completed")
	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig(serviceName, version, environment string) Config {
	return Config{
		ServiceName:       serviceName,
		ServiceVersion:    version,
		Environment:       environment,
		TracingEnabled:    false,
		TracingEndpoint:   "localhost:4318",
		TracingProtocol:   "http",
		TracingSampler:    "always",
		TracingSampleRate: 1.0,
		MetricsEnabled:    false,
		MetricsEndpoint:   "localhost:4318",
		MetricsProtocol:   "http",
		MetricsInterval:   30 * time.Second,
		Insecure:          true,
		Timeout:           10 * time.Second,
		Compression:       "gzip",
	}
}

// Health check for telemetry
func (p *Provider) HealthCheck(ctx context.Context) error {
	// Simple health check - could be enhanced to actually test connectivity
	if p.config.TracingEnabled && p.tracerProvider == nil {
		return fmt.Errorf("tracing is enabled but tracer provider is nil")
	}
	if p.config.MetricsEnabled && p.meterProvider == nil {
		return fmt.Errorf("metrics is enabled but meter provider is nil")
	}
	return nil
}
