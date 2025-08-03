# OpenTelemetry Integration

SimpleIdentity now includes comprehensive OpenTelemetry (OTEL) support for metrics and distributed tracing, providing enterprise-grade observability.

## 🔭 Features

### Distributed Tracing
- **Protocols**: HTTP and gRPC OTLP exporters
- **Sampling**: Configurable sampling strategies (always, never, ratio-based)
- **Context Propagation**: Automatic trace context propagation across service boundaries
- **Instrumentation**: Automatic HTTP and gRPC middleware, plus manual instrumentation helpers

### Metrics Collection
- **Protocols**: HTTP and gRPC OTLP exporters
- **Service Metrics**: Request duration, count, active requests, error rates
- **Auth Metrics**: Authentication attempts, success/failure rates, token operations
- **Database Metrics**: Connection pools, query performance, error tracking
- **Custom Metrics**: Easy-to-use helpers for application-specific metrics

### Middleware & Instrumentation
- **HTTP Middleware**: Automatic tracing and metrics for HTTP endpoints
- **gRPC Interceptors**: Server and client interceptors for gRPC services
- **Database Tracing**: Helpers for database operation instrumentation
- **Authentication Tracing**: Built-in instrumentation for auth operations

## 🚀 Quick Start

### Basic Usage
```bash
# Enable telemetry with default settings
./simpleidentity server --telemetry-enabled --tracing-enabled --metrics-enabled

# Use environment variables (12-factor compliant)
export SMPIDT_TELEMETRY_ENABLED=true
export SMPIDT_TRACING_ENABLED=true
export SMPIDT_METRICS_ENABLED=true
./simpleidentity server
```

### Configuration Options

#### Core Telemetry Settings
```bash
--telemetry-enabled              # Enable OpenTelemetry
--telemetry-environment string   # Environment name (dev, staging, prod)
```

#### Tracing Configuration
```bash
--tracing-enabled                # Enable distributed tracing
--tracing-endpoint string        # OTLP endpoint (default: localhost:4318)
--tracing-protocol string        # Protocol: http or grpc (default: http)
--tracing-sampler string         # Sampler: always, never, ratio (default: always)
--tracing-sample-rate float      # Sample rate 0.0-1.0 (default: 1.0)
```

#### Metrics Configuration
```bash
--metrics-enabled                # Enable metrics collection
--metrics-endpoint string        # OTLP endpoint (default: localhost:4318)
--metrics-protocol string        # Protocol: http or grpc (default: http)
--metrics-interval duration      # Collection interval (default: 30s)
```

#### OTLP Transport Settings
```bash
--otlp-insecure                  # Use insecure connection (default: true)
--otlp-timeout duration          # Connection timeout (default: 10s)
--otlp-compression string        # Compression: gzip or none (default: gzip)
```

## 🔧 Integration Examples

### Jaeger (Tracing)
```bash
# Start Jaeger (all-in-one)
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14250:14250 \
  jaegertracing/all-in-one:latest

# Run service with Jaeger
./simpleidentity server \
  --telemetry-enabled \
  --tracing-enabled \
  --tracing-endpoint localhost:14250 \
  --tracing-protocol grpc
```

### Prometheus + Grafana (Metrics)
```bash
# Using OpenTelemetry Collector
./simpleidentity server \
  --telemetry-enabled \
  --metrics-enabled \
  --metrics-endpoint localhost:4318 \
  --metrics-protocol http
```

### OTLP Collector
```yaml
# otel-collector.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [jaeger]
    metrics:
      receivers: [otlp]
      exporters: [prometheus]
```

## 📊 Available Metrics

### Service Metrics
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - HTTP request duration
- `http_active_requests` - Active HTTP requests
- `http_errors_total` - HTTP error count

- `grpc_requests_total` - Total gRPC requests  
- `grpc_request_duration_seconds` - gRPC request duration
- `grpc_active_requests` - Active gRPC requests
- `grpc_errors_total` - gRPC error count

### Authentication Metrics
- `auth_attempts_total` - Authentication attempts
- `auth_success_total` - Successful authentications
- `auth_failures_total` - Failed authentications
- `auth_duration_seconds` - Authentication duration
- `tokens_issued_total` - Tokens issued
- `tokens_validated_total` - Tokens validated

### Database Metrics
- `db_{name}_connections_active` - Active database connections
- `db_{name}_connections_opened_total` - Total connections opened
- `db_{name}_query_duration_seconds` - Database query duration
- `db_{name}_queries_total` - Total database queries
- `db_{name}_query_errors_total` - Database query errors

## 🛠️ Development Usage

### Adding Custom Tracing
```go
import "github.com/posilva/simpleidentity/pkg/telemetry"

// Create instrumenter for your component
instrumenter := telemetry.NewInstrumenter("my_component")

// Manual tracing
ctx, span := instrumenter.StartSpan(ctx, "my_operation",
    telemetry.UserIDAttr(userID),
    telemetry.ProviderAttr("google"),
)
defer span.End()

// Record error if needed
if err != nil {
    instrumenter.RecordError(span, err, "operation failed")
    return err
}

// Set success status
span.SetStatus(codes.Ok, "operation completed")
```

### Adding Custom Metrics
```go
// Create service metrics
serviceMetrics, err := instrumenter.NewServiceMetrics("my_service")
if err != nil {
    return err
}

// Use middleware pattern
myHandler := serviceMetrics.Middleware(func(ctx context.Context) error {
    // Your business logic here
    return doSomething(ctx)
})
```

### HTTP Middleware
```go
import "github.com/posilva/simpleidentity/pkg/telemetry"

// Create HTTP middleware
httpMiddleware, err := telemetry.NewHTTPMiddleware("simpleidentity")
if err != nil {
    return err
}

// Wrap your HTTP handlers
http.Handle("/api/auth", httpMiddleware.Handler(authHandler))
```

### gRPC Interceptors
```go
import "github.com/posilva/simpleidentity/pkg/telemetry"

// Create gRPC interceptors
grpcInterceptors, err := telemetry.NewGRPCInterceptors("simpleidentity")
if err != nil {
    return err
}

// Add to gRPC server
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcInterceptors.UnaryServerInterceptor()),
    grpc.StreamInterceptor(grpcInterceptors.StreamServerInterceptor()),
)
```

## 🌍 Environment Variables

All telemetry configuration supports environment variables with the `SMPIDT_` prefix:

```bash
# Core settings
export SMPIDT_TELEMETRY_ENABLED=true
export SMPIDT_TELEMETRY_ENVIRONMENT=production

# Tracing
export SMPIDT_TRACING_ENABLED=true
export SMPIDT_TRACING_ENDPOINT=jaeger:4317
export SMPIDT_TRACING_PROTOCOL=grpc
export SMPIDT_TRACING_SAMPLER=ratio
export SMPIDT_TRACING_SAMPLE_RATE=0.1

# Metrics
export SMPIDT_METRICS_ENABLED=true
export SMPIDT_METRICS_ENDPOINT=otel-collector:4318
export SMPIDT_METRICS_PROTOCOL=http
export SMPIDT_METRICS_INTERVAL=15s

# Transport
export SMPIDT_OTLP_INSECURE=false
export SMPIDT_OTLP_TIMEOUT=30s
export SMPIDT_OTLP_COMPRESSION=gzip
```

## 🐳 Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: simpleidentity
spec:
  template:
    spec:
      containers:
      - name: simpleidentity
        image: simpleidentity:latest
        env:
        - name: SMPIDT_TELEMETRY_ENABLED
          value: "true"
        - name: SMPIDT_TRACING_ENABLED
          value: "true"  
        - name: SMPIDT_METRICS_ENABLED
          value: "true"
        - name: SMPIDT_TRACING_ENDPOINT
          value: "jaeger-collector:14250"
        - name: SMPIDT_TRACING_PROTOCOL
          value: "grpc"
        - name: SMPIDT_METRICS_ENDPOINT
          value: "otel-collector:4318"
        - name: SMPIDT_TELEMETRY_ENVIRONMENT
          value: "production"
        - name: SMPIDT_OTLP_INSECURE
          value: "false"
        ports:
        - containerPort: 8080  # Health checks
        - containerPort: 9090  # gRPC API
        - containerPort: 8090  # HTTP API
```

## 🔍 Trace Context

The service automatically propagates trace context using:
- **W3C Trace Context** - Standard HTTP header propagation
- **Baggage** - Additional context data
- **gRPC Metadata** - For gRPC service calls

## 📈 Performance Impact

- **Tracing**: Minimal overhead with sampling
- **Metrics**: Low overhead with periodic collection
- **No Telemetry**: Zero overhead when disabled
- **Graceful Degradation**: Service continues if telemetry backend is unavailable

## 🔧 Troubleshooting

### Connection Issues
```bash
# Test with insecure connection
--otlp-insecure=true

# Increase timeout
--otlp-timeout=30s

# Check endpoint format (no protocol prefix for gRPC)
--tracing-endpoint=localhost:4317  # gRPC
--tracing-endpoint=localhost:4318  # HTTP
```

### Performance Tuning
```bash
# Reduce sampling rate
--tracing-sampler=ratio --tracing-sample-rate=0.1

# Increase metrics interval  
--metrics-interval=60s

# Disable compression if needed
--otlp-compression=none
```

## 🎯 Next Steps

The telemetry foundation is now ready for:
1. **Custom Business Metrics** - Add domain-specific metrics
2. **Alerting Rules** - Configure alerts based on metrics
3. **Dashboards** - Create Grafana dashboards
4. **SLI/SLO Monitoring** - Track service level objectives
5. **Distributed Tracing Analysis** - Debug complex request flows
