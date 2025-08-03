# 🧪 Docker Compose Stack Testing Guide

## Overview
This guide provides comprehensive testing procedures for the SimpleIdentity Docker Compose stack, including all services and observability components.

## 🚀 **Quick Start Testing**

### 1. Start the Complete Stack
```bash
# Build and start all services
docker-compose up -d

# Follow logs in real-time
docker-compose logs -f

# Check service status
docker-compose ps
```

### 2. Verify All Services are Running
```bash
# Check container status
docker-compose ps

# Expected output:
# NAME               COMMAND                  SERVICE          STATUS          PORTS
# grafana            "/run.sh"                grafana          running         0.0.0.0:3000->3000/tcp
# jaeger             "/go/bin/all-in-one"     jaeger           running         0.0.0.0:4317-4318->4317-4318/tcp, 0.0.0.0:16686->16686/tcp
# prometheus         "/bin/prometheus --c…"   prometheus       running         0.0.0.0:9090->9090/tcp
# simpleidentity     "/usr/local/bin/simp…"   simpleidentity   running         0.0.0.0:6060->6060/tcp, 0.0.0.0:8080->8080/tcp, 0.0.0.0:8090->8090/tcp, 0.0.0.0:9090->9090/tcp
```

## 🏥 **Health Check Testing**

### SimpleIdentity Service Health
```bash
# Basic health check
curl http://localhost:8080/health

# Expected response:
{"status":"ok","timestamp":"2025-08-03T22:30:15Z","checks":{"startup":{"status":"pass"}}}

# Detailed health endpoints
curl http://localhost:8080/health/live    # Liveness probe
curl http://localhost:8080/health/ready   # Readiness probe

# Container health check
docker-compose exec simpleidentity /usr/local/bin/simpleidentity health --addr localhost:8080
```

### Service Connectivity Test
```bash
# Test internal service communication
docker-compose exec simpleidentity ping jaeger
docker-compose exec simpleidentity ping prometheus
docker-compose exec simpleidentity ping grafana

# Test external connectivity (if needed)
docker-compose exec simpleidentity curl -f http://jaeger:4318/v1/traces || echo "OTLP HTTP endpoint test"
```

## 📊 **Observability Stack Testing**

### Jaeger (Distributed Tracing)
```bash
# Access Jaeger UI
echo "Jaeger UI: http://localhost:16686"

# Test OTLP endpoints
curl -I http://localhost:4318/v1/traces  # HTTP OTLP
curl -I http://localhost:4317            # gRPC OTLP (will show connection refused, which is expected for HTTP curl)

# Generate test trace (when tracing is enabled)
curl http://localhost:8090/health  # This should generate traces
```

### Prometheus (Metrics)
```bash
# Access Prometheus UI
echo "Prometheus UI: http://localhost:9090"

# Test Prometheus API
curl http://localhost:9090/api/v1/status/config
curl http://localhost:9090/api/v1/targets

# Check if SimpleIdentity metrics are being scraped
curl 'http://localhost:9090/api/v1/query?query=up{job="simpleidentity-health"}'
```

### Grafana (Visualization)
```bash
# Access Grafana UI
echo "Grafana UI: http://localhost:3000"
echo "Default credentials: admin/admin123"

# Test Grafana API
curl -u admin:admin123 http://localhost:3000/api/health

# Test datasource connectivity
curl -u admin:admin123 http://localhost:3000/api/datasources
```

## 🔍 **Service Integration Testing**

### End-to-End OpenTelemetry Flow
```bash
# Enable telemetry in the service (should be enabled by default in compose)
curl http://localhost:8080/health  # Generate some activity

# Check traces in Jaeger
# 1. Open http://localhost:16686
# 2. Search for service "simpleidentity"
# 3. Look for health check traces

# Check metrics in Prometheus
# 1. Open http://localhost:9090
# 2. Query: up{job="simpleidentity-health"}
# 3. Should show 1 for healthy service
```

### Network Testing
```bash
# Test internal network connectivity
docker network inspect accounts-service_simpleidentity-net

# Test service-to-service communication
docker-compose exec simpleidentity nslookup jaeger
docker-compose exec simpleidentity nslookup prometheus
docker-compose exec simpleidentity nslookup grafana
```

## 🐛 **Debug and Troubleshooting Tests**

### pprof Debug Endpoints
```bash
# Access pprof endpoints (development only)
curl http://localhost:6060/debug/pprof/
curl http://localhost:6060/debug/pprof/heap
curl http://localhost:6060/debug/pprof/goroutine

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap
```

### Log Analysis
```bash
# View service logs
docker-compose logs simpleidentity
docker-compose logs jaeger
docker-compose logs prometheus
docker-compose logs grafana

# Follow specific service logs
docker-compose logs -f simpleidentity

# Search for errors
docker-compose logs simpleidentity | grep -i error
docker-compose logs simpleidentity | grep -i warning
```

### Container Resource Usage
```bash
# Check resource usage
docker stats

# Expected low resource usage:
# CONTAINER      CPU %     MEM USAGE / LIMIT     MEM %     NET I/O           BLOCK I/O
# simpleidentity  0.01%     10MiB / 1.944GiB     0.50%     1.2kB / 850B      0B / 0B
```

## 🧪 **Load Testing**

### Basic Load Test
```bash
# Install hey (HTTP load testing tool) if not available
# go install github.com/rakyll/hey@latest

# Simple load test on health endpoint
hey -n 100 -c 10 http://localhost:8080/health

# Monitor during load test
watch -n 1 'curl -s http://localhost:8080/health'
```

### Health Check Load Test
```bash
# Stress test health endpoints
for i in {1..50}; do
  curl -s http://localhost:8080/health &
  curl -s http://localhost:8080/health/live &
  curl -s http://localhost:8080/health/ready &
done
wait

echo "Load test completed"
```

## 📝 **Automated Testing Script**

### Create a comprehensive test script:
```bash
#!/bin/bash
# save as test-stack.sh

set -e

echo "🧪 Starting Docker Compose Stack Testing..."

# Start stack
echo "📦 Starting services..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 30

# Test SimpleIdentity
echo "🏥 Testing SimpleIdentity health..."
curl -f http://localhost:8080/health || { echo "❌ Health check failed"; exit 1; }
echo "✅ Health check passed"

# Test Jaeger
echo "🔍 Testing Jaeger..."
curl -f http://localhost:16686/api/services || { echo "❌ Jaeger API failed"; exit 1; }
echo "✅ Jaeger is accessible"

# Test Prometheus
echo "📊 Testing Prometheus..."
curl -f http://localhost:9090/api/v1/status/config || { echo "❌ Prometheus API failed"; exit 1; }
echo "✅ Prometheus is accessible"

# Test Grafana
echo "📈 Testing Grafana..."
curl -f -u admin:admin123 http://localhost:3000/api/health || { echo "❌ Grafana API failed"; exit 1; }
echo "✅ Grafana is accessible"

echo "🎉 All tests passed!"
echo ""
echo "🌐 Access URLs:"
echo "- SimpleIdentity Health: http://localhost:8080/health"
echo "- Jaeger UI: http://localhost:16686"
echo "- Prometheus UI: http://localhost:9090"
echo "- Grafana UI: http://localhost:3000 (admin/admin123)"
echo "- pprof Debug: http://localhost:6060/debug/pprof/"
```

## 🔄 **Performance Testing**

### Startup Time Test
```bash
# Measure startup time
time docker-compose up -d

# Check when services become healthy
start_time=$(date +%s)
while ! curl -f -s http://localhost:8080/health > /dev/null; do
  sleep 1
done
end_time=$(date +%s)
echo "SimpleIdentity ready in $((end_time - start_time)) seconds"
```

### Memory Usage Test
```bash
# Monitor memory usage during operation
docker stats --no-stream simpleidentity

# Expected results:
# - SimpleIdentity: ~10-20MB memory usage
# - CPU usage should be minimal when idle
```

## 🧹 **Cleanup Testing**

### Graceful Shutdown Test
```bash
# Test graceful shutdown
docker-compose stop simpleidentity

# Check logs for graceful shutdown messages
docker-compose logs simpleidentity | tail -10

# Should see:
# "Received shutdown signal"
# "Starting graceful shutdown"
# "Graceful shutdown completed"
```

### Complete Stack Cleanup
```bash
# Stop all services
docker-compose down

# Remove volumes (if needed)
docker-compose down -v

# Clean up images (optional)
docker-compose down --rmi all
```

## 📋 **Testing Checklist**

### ✅ **Basic Functionality**
- [ ] All containers start successfully
- [ ] Health endpoints respond correctly
- [ ] Services can communicate internally
- [ ] External ports are accessible

### ✅ **Observability**
- [ ] Jaeger UI loads and shows services
- [ ] Prometheus targets are up
- [ ] Grafana dashboards load
- [ ] Metrics are being collected

### ✅ **Performance**
- [ ] Startup time is reasonable (<30s)
- [ ] Memory usage is low (<50MB per service)
- [ ] CPU usage is minimal when idle
- [ ] Load testing shows stable performance

### ✅ **Security**
- [ ] Services run as non-root users
- [ ] Internal network isolation works
- [ ] No sensitive data in logs
- [ ] Debug endpoints only available internally

### ✅ **Reliability**
- [ ] Services restart on failure
- [ ] Graceful shutdown works correctly
- [ ] Health checks detect failures
- [ ] Log rotation works properly

## 🎯 **Expected Results**

When all tests pass, you should have:
- **SimpleIdentity**: Healthy and responding on all endpoints
- **Jaeger**: Collecting and displaying traces
- **Prometheus**: Scraping metrics from all targets  
- **Grafana**: Connected to Prometheus with dashboards
- **Network**: All services communicating properly
- **Performance**: Low resource usage and fast response times

This comprehensive testing ensures your Docker Compose stack is production-ready and all observability components are working correctly! 🚀
