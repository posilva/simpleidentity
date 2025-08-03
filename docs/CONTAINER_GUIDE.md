# SimpleIdentity Container Guide 🐳

## Overview
This guide covers the robust containerization of the SimpleIdentity service, including Docker image creation, orchestration, and observability stack deployment.

## 🏗️ Container Architecture

### Multi-Stage Dockerfile Features

#### Build Stage (golang:1.23.4-alpine)
- **Optimized Caching**: Dependencies downloaded separately for better layer caching
- **Security**: Non-root user creation for runtime security
- **Static Binary**: CGO disabled for portable, static binary
- **Size Optimization**: Debug symbols and symbol table removed (`-w -s`)
- **Verification**: Binary tested during build process

#### Runtime Stage (gcr.io/distroless/static:nonroot)
- **Minimal Attack Surface**: Distroless base image with only essential components
- **Security**: Runs as non-root user by default
- **SSL Support**: CA certificates included for external connections
- **Timezone Support**: Full timezone data available

### Container Security Features

```dockerfile
# Non-root execution
USER nonroot:nonroot

# Minimal base image (distroless)
FROM gcr.io/distroless/static:nonroot

# Static binary (no external dependencies)
CGO_ENABLED=0
```

## 🚀 Build & Run

### Quick Start

```bash
# Build the container
docker build -t simpleidentity:latest .

# Run with basic configuration
docker run -p 8080:8080 -p 8090:8090 -p 9090:9090 \
  -e SMPIDT_LOG_LEVEL=info \
  simpleidentity:latest

# Check health
curl http://localhost:8080/health
```

### Development with Observability Stack

```bash
# Start full development environment
docker-compose up -d

# View logs
docker-compose logs -f simpleidentity

# Access services
# - SimpleIdentity Health: http://localhost:8080/health
# - Jaeger UI: http://localhost:16686
# - Grafana: http://localhost:3000 (admin/admin123)
# - Prometheus: http://localhost:9090
```

## 📊 Container Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SMPIDT_LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `SMPIDT_LOG_PRETTY` | `false` | Pretty logging (set true for development) |
| `SMPIDT_HEALTH_ADDR` | `:8080` | Health check server address |
| `SMPIDT_HTTP_ADDR` | `:8090` | HTTP API server address |
| `SMPIDT_GRPC_ADDR` | `:9090` | gRPC API server address |
| `SMPIDT_PPROF_ADDR` | `:6060` | pprof debug server (internal) |
| `SMPIDT_TELEMETRY_ENABLED` | `false` | Enable OpenTelemetry |
| `SMPIDT_METRICS_ENABLED` | `false` | Enable metrics collection |
| `SMPIDT_TRACING_ENABLED` | `false` | Enable distributed tracing |

### Port Configuration

| Port | Protocol | Purpose | Exposure |
|------|----------|---------|----------|
| 8080 | HTTP | Health checks | Public |
| 8090 | HTTP | REST API | Public |
| 9090 | gRPC | gRPC API | Public |
| 6060 | HTTP | pprof debug | Internal only |

## 🔍 Health Checks

### Container Health Check
```bash
# Built-in Docker health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/simpleidentity", "health", "--addr", "localhost:8080"]
```

### Manual Health Verification
```bash
# Basic health check
curl http://localhost:8080/health

# Detailed health status
curl http://localhost:8080/health/ready
curl http://localhost:8080/health/live
```

### Kubernetes Probes
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## 📈 Observability Stack

### Included Services

#### Jaeger (Distributed Tracing)
- **UI**: http://localhost:16686
- **OTLP Endpoint**: http://localhost:4318
- **Purpose**: Trace request flows and performance bottlenecks

#### Prometheus (Metrics)
- **UI**: http://localhost:9090
- **Scrape Config**: Auto-configured for SimpleIdentity
- **Purpose**: Metrics collection and alerting

#### Grafana (Visualization)
- **UI**: http://localhost:3000
- **Credentials**: admin/admin123
- **Purpose**: Dashboards and metrics visualization

### OpenTelemetry Configuration

```bash
# Enable full observability
docker run -p 8080:8080 \
  -e SMPIDT_TELEMETRY_ENABLED=true \
  -e SMPIDT_METRICS_ENABLED=true \
  -e SMPIDT_TRACING_ENABLED=true \
  -e SMPIDT_METRICS_ENDPOINT=jaeger:4318 \
  -e SMPIDT_TRACING_ENDPOINT=jaeger:4318 \
  simpleidentity:latest
```

## 🏃‍♂️ Production Deployment

### Production Environment Variables

```bash
# Production configuration
SMPIDT_LOG_LEVEL=info
SMPIDT_LOG_PRETTY=false
SMPIDT_TELEMETRY_ENVIRONMENT=production
SMPIDT_VERSION=1.0.0

# External observability endpoints
SMPIDT_METRICS_ENDPOINT=https://metrics.company.com:4318
SMPIDT_TRACING_ENDPOINT=https://traces.company.com:4318
SMPIDT_OTLP_INSECURE=false
```

### Resource Limits

```yaml
resources:
  limits:
    memory: "256Mi"
    cpu: "200m"
  requests:
    memory: "128Mi"
    cpu: "100m"
```

### Security Considerations

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532  # nonroot user
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
```

## 🔧 Container Optimization

### Image Size Optimization
- **Multi-stage build**: Separate build and runtime environments
- **Distroless base**: Minimal runtime dependencies
- **Static binary**: No external library dependencies
- **Debug symbols removed**: Smaller binary size

### Build Performance
- **Layer caching**: Dependencies cached separately
- **Parallel builds**: Multi-stage concurrent execution
- **Build context**: Optimized with `.dockerignore`

### Runtime Performance
- **Minimal overhead**: Distroless base image
- **Static binary**: No dynamic linking overhead
- **Resource efficient**: Low memory and CPU footprint

## 📝 Troubleshooting

### Common Issues

#### Health Check Failures
```bash
# Check if health endpoint is accessible
docker exec simpleidentity-container curl localhost:8080/health

# Check container logs
docker logs simpleidentity-container
```

#### Port Conflicts
```bash
# Use alternative ports
docker run -p 18080:8080 -p 18090:8090 -p 19090:9090 simpleidentity:latest
```

#### Permission Issues
```bash
# Verify non-root execution
docker exec simpleidentity-container id
# Should show: uid=65532(nonroot) gid=65532(nonroot)
```

### Debug Mode

```bash
# Run with debug logging
docker run -e SMPIDT_LOG_LEVEL=debug \
           -e SMPIDT_LOG_PRETTY=true \
           simpleidentity:latest
```

## 🎯 Next Steps

1. **Kubernetes Deployment**: Create Helm charts for production deployment
2. **CI/CD Integration**: Automated builds and security scanning
3. **Monitoring**: Set up alerting rules and SLOs
4. **Load Testing**: Container performance under load
5. **Security Scanning**: Regular vulnerability assessments

The containerized SimpleIdentity service is now production-ready with enterprise-grade security, observability, and operational features.
