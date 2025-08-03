# SimpleIdentity Foundation

This document describes the foundational infrastructure components implemented for the SimpleIdentity service.

## 🏗️ Foundation Components

### 1. Abstracted Logger Layer (zerolog)
- **Location**: `pkg/logger/`
- **Features**:
  - Interface-based abstraction for easy testing and swapping
  - Structured logging with JSON output
  - Pretty printing for development
  - Context support for request tracing
  - Global convenience functions

**Usage:**
```go
import "github.com/posilva/simpleidentity/pkg/logger"

// Initialize logger
log := logger.New("info", false)

// Use logger
log.Info().Str("component", "auth").Msg("User authenticated")
log.Error().Err(err).Str("user_id", "123").Msg("Authentication failed")

// Use global logger
logger.InitGlobal("debug", true)
logger.Info().Msg("Global logger message")
```

### 2. Health Check Server
- **Location**: `pkg/health/`
- **Endpoints**:
  - `/health` - Comprehensive health check with all dependencies
  - `/health/live` - Liveness probe (simple ping)
  - `/health/ready` - Readiness probe (dependencies check)
- **Features**:
  - Kubernetes-compatible probes
  - Concurrent health check execution
  - Extensible check system
  - JSON responses with detailed status

**Usage:**
```go
import "github.com/posilva/simpleidentity/pkg/health"

// Create checker
checker := health.NewChecker(logger, "v1.0.0")

// Add custom checks
checker.AddCheck("database", health.DatabaseCheck(db.Ping))
checker.AddCheck("redis", health.HTTPCheck("http://redis:6379", 5*time.Second))

// Start server
server := health.NewServer(":8080", checker, logger)
server.Start(ctx)
```

### 3. pprof Debug Server
- **Location**: `pkg/pprof/`
- **Port**: 6060 (internal, not publicly exposed)
- **Endpoints**:
  - `/debug/pprof/` - Index with all available profiles
  - `/debug/pprof/profile` - CPU profile
  - `/debug/pprof/heap` - Memory heap profile
  - `/debug/pprof/goroutine` - Goroutine profile
  - `/debug/pprof/block` - Block profile
  - `/debug/pprof/mutex` - Mutex profile

**Usage:**
```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Web interface
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile
```

### 4. Graceful Shutdown Manager
- **Location**: `pkg/shutdown/`
- **Features**:
  - Signal handling (SIGINT, SIGTERM, SIGQUIT)
  - Configurable shutdown timeout
  - Concurrent hook execution
  - LIFO execution order
  - Error collection and reporting

**Usage:**
```go
import "github.com/posilva/simpleidentity/pkg/shutdown"

// Create manager
shutdownMgr := shutdown.NewManager(30*time.Second, logger)

// Add shutdown hooks
shutdownMgr.AddHook(shutdown.ServerShutdownHook(httpServer, "http-server"))
shutdownMgr.AddHook(shutdown.DatabaseCloseHook(dbConn, "database"))
shutdownMgr.AddHook(shutdown.CustomHook("cleanup", func(ctx context.Context) error {
    // Custom cleanup logic
    return nil
}))

// Wait for shutdown
shutdownMgr.Wait(ctx)
```

## 🚀 Quick Start

### Build and Run
```bash
# Using Makefile
make run

# Using go run
go run ./cmd/simpleidentity server --log-pretty --log-level debug

# Using environment variables
export SMPIDT_LOG_LEVEL=debug
export SMPIDT_LOG_PRETTY=true
go run ./cmd/simpleidentity server
```

### Available Commands
```bash
# Start server
./simpleidentity server

# Show version
./simpleidentity version

# Show help
./simpleidentity --help
./simpleidentity server --help
```

### Environment Configuration (12-Factor)
All configuration can be set via environment variables:

```bash
export SMPIDT_LOG_LEVEL=info          # Log level
export SMPIDT_LOG_PRETTY=false        # Pretty logging
export SMPIDT_HEALTH_ADDR=:8080       # Health server address
export SMPIDT_PPROF_ADDR=:6060        # pprof server address
export SMPIDT_GRPC_ADDR=:9090         # gRPC server address (future)
export SMPIDT_HTTP_ADDR=:8090         # HTTP server address (future)
export SMPIDT_SHUTDOWN_TIMEOUT=30s    # Graceful shutdown timeout
export SMPIDT_VERSION=v1.0.0          # Service version
```

## 🔍 Monitoring and Observability

### Health Checks
```bash
# Check overall health
curl http://localhost:8080/health

# Liveness probe (for Kubernetes)
curl http://localhost:8080/health/live

# Readiness probe (for Kubernetes)
curl http://localhost:8080/health/ready
```

### Performance Profiling
```bash
# Open pprof web interface
make pprof

# Or manually
go tool pprof -http=:8081 http://localhost:6060/debug/pprof/profile
```

### Logging
Structured JSON logs with configurable levels:
- `debug` - Detailed debugging information
- `info` - General operational messages
- `warn` - Warning conditions
- `error` - Error conditions

## 🐳 Kubernetes Integration

The service is designed for Kubernetes deployment with:

```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: simpleidentity
    image: simpleidentity:latest
    ports:
    - containerPort: 8080  # Health checks
    - containerPort: 9090  # gRPC API
    - containerPort: 8090  # HTTP API
    # pprof port 6060 should NOT be exposed publicly
    livenessProbe:
      httpGet:
        path: /health/live
        port: 8080
      initialDelaySeconds: 10
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health/ready
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
    env:
    - name: SMPIDT_LOG_LEVEL
      value: "info"
    - name: SMPIDT_HEALTH_ADDR
      value: ":8080"
```

## 🔧 Development

### Adding New Health Checks
```go
// In your service initialization
healthChecker.AddCheck("my-service", func(ctx context.Context) error {
    // Check your service health
    return myService.Ping(ctx)
})
```

### Adding Shutdown Hooks
```go
// Add to shutdown manager
shutdownMgr.AddHook(shutdown.CustomHook("my-cleanup", func(ctx context.Context) error {
    // Your cleanup logic here
    return nil
}))
```

### Custom Logging
```go
// Create component-specific logger
log := logger.New("info", false).With().Str("component", "auth").Logger()
log.Info().Str("user_id", userID).Msg("User authenticated successfully")
```

## 📝 Next Steps

The foundation is now ready for:
1. **gRPC Server** - Add your gRPC API handlers
2. **HTTP Server** - Add your REST API handlers  
3. **Database Integration** - Add database health checks
4. **Metrics** - Add Prometheus metrics
5. **Tracing** - Add distributed tracing support
6. **Authentication Logic** - Implement the actual auth providers
