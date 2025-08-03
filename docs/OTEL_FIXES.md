# 🔧 OTEL Collector Configuration Fixes

## Issues Fixed

### 1. Invalid Jaeger Exporter ❌ → ✅ Fixed
**Error**: `unknown type: "jaeger" for id: "jaeger"`

**Solution**: Replaced `jaeger` exporter with `otlp/jaeger` exporter
```yaml
# Before (broken):
exporters:
  jaeger:
    endpoint: jaeger:14250

# After (fixed):
exporters:
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true
```

### 2. Missing memory_limiter check_interval ❌ → ✅ Fixed
**Error**: `checkInterval must be greater than zero`

**Solution**: Added required `check_interval` parameter
```yaml
# Before (broken):
memory_limiter:
  limit_mib: 256

# After (fixed):
memory_limiter:
  limit_mib: 256
  check_interval: 1s  # <- This was missing
```

## Complete Fixed Configuration

The `/deployments/otel-collector/config.yaml` now has:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024
    send_batch_max_size: 2048

  memory_limiter:
    limit_mib: 256
    check_interval: 1s  # ✅ Fixed: Added required parameter

  resource:
    attributes:
      - key: service.namespace
        value: simpleidentity
        action: upsert
      - key: deployment.environment
        value: development
        action: upsert

exporters:
  otlp/jaeger:  # ✅ Fixed: Using OTLP exporter instead of deprecated jaeger
    endpoint: jaeger:4317
    tls:
      insecure: true

  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: simpleidentity
    const_labels:
      environment: development

  logging:
    loglevel: info

service:
  telemetry:
    logs:
      level: info
    metrics:
      address: 0.0.0.0:8888

  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, resource, batch]
      exporters: [otlp/jaeger, logging]  # ✅ Fixed: Updated exporter name

    metrics:
      receivers: [otlp]
      processors: [memory_limiter, resource, batch]
      exporters: [prometheus, logging]
```

## How to Apply the Fix

### Option 1: Restart with Docker Compose
```bash
# Stop the stack
sudo docker-compose down

# Start with fixed configuration
sudo docker-compose up -d

# Check logs
sudo docker-compose logs otel-collector
```

### Option 2: Test Configuration First
```bash
# Validate the configuration
docker-compose config --quiet

# Check if OTEL Collector starts without errors
sudo docker-compose up otel-collector
```

## Expected Results After Fix

### ✅ OTEL Collector Should Start Successfully
- No more "unknown type: jaeger" errors
- No more "checkInterval must be greater than zero" errors
- Clean startup logs

### ✅ Telemetry Flow Should Work
```
SimpleIdentity → OTEL Collector → Jaeger (traces)
                                → Prometheus (metrics)
```

### ✅ Test the Fix
```bash
# 1. Start the stack
sudo docker-compose up -d

# 2. Check OTEL Collector logs
sudo docker-compose logs otel-collector

# 3. Test endpoints
curl http://localhost:4319/v1/traces  # OTEL Collector HTTP
curl http://localhost:8889/metrics    # OTEL Collector Prometheus metrics

# 4. Generate telemetry data
curl http://localhost:8080/health     # This should create traces and metrics

# 5. Verify in UIs
# - Jaeger: http://localhost:16686 (should show simpleidentity traces)
# - Prometheus: http://localhost:9090 (should show simpleidentity metrics)
```

## Architecture After Fix

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────┐
│ SimpleIdentity  │───▶│ OTEL Collector   │───▶│ Jaeger      │
│                 │    │                  │    │ (traces)    │
│ - Metrics  ────────▶│ - Receives OTLP  │    └─────────────┘
│ - Traces  ──────────│ - Routes data    │    
└─────────────────┘    │ - Exports        │    ┌─────────────┐
                       └──────────────────┘───▶│ Prometheus  │
                                              │ (metrics)   │
                                              └─────────────┘
```

The OTEL Collector configuration is now correct and should work without errors! 🎉
