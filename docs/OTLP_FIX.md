# 🔧 Fixed: OTLP Metrics 404 Error

## Problem Summary
The error `failed to upload metrics: failed to send metrics to http://jaeger:4318/v1/metrics: 404 Not Found` occurred because:

**Jaeger only accepts traces, not metrics!** 

The SimpleIdentity service was trying to send both metrics and traces to Jaeger's OTLP endpoint, but Jaeger only handles distributed tracing data.

## ✅ Solution Implemented

### 1. Added OpenTelemetry Collector
```yaml
otel-collector:
  image: otel/opentelemetry-collector-contrib:0.91.0
  # Receives both metrics and traces
  # Routes traces to Jaeger
  # Exposes metrics for Prometheus
```

### 2. Separated Endpoints
```yaml
# In docker-compose.yml SimpleIdentity service:
SMPIDT_METRICS_ENDPOINT: "otel-collector:4318"  # Metrics -> OTEL Collector
SMPIDT_TRACING_ENDPOINT: "jaeger:4318"          # Traces -> Jaeger directly
```

### 3. Updated Architecture
```
SimpleIdentity Service
├── Metrics  ──→ OpenTelemetry Collector ──→ Prometheus (scraping)
└── Traces   ──→ Jaeger (directly)
```

## 🏗️ New Architecture Flow

### Before (Broken):
```
SimpleIdentity ──[metrics + traces]──→ Jaeger ❌
                                        │
                                        └── 404 Not Found (metrics not supported)
```

### After (Fixed):
```
SimpleIdentity ──[metrics]──→ OTEL Collector ──[expose]──→ Prometheus ✅
               │                                            
               └──[traces]──→ Jaeger ✅
```

## 📊 Services & Ports

| Service | Port | Purpose |
|---------|------|---------|
| SimpleIdentity | 8080 | Health checks |
| SimpleIdentity | 8090 | HTTP API |
| SimpleIdentity | 9091 | gRPC API |
| SimpleIdentity | 6060 | pprof debug |
| Jaeger | 16686 | Jaeger UI |
| Jaeger | 4318 | OTLP HTTP (traces only) |
| OTEL Collector | 4319 | OTLP HTTP (metrics + traces) |
| OTEL Collector | 8889 | Prometheus metrics endpoint |
| Prometheus | 9090 | Prometheus UI |
| Grafana | 3000 | Grafana UI |

## 🧪 Testing the Fix

### 1. Start the updated stack:
```bash
docker-compose down  # Stop old stack
docker-compose up -d # Start with OTEL Collector
```

### 2. Test metrics flow:
```bash
# Check OTEL Collector is receiving metrics
curl http://localhost:4319/v1/metrics

# Check Prometheus is scraping OTEL Collector
curl http://localhost:9090/api/v1/targets
```

### 3. Test traces flow:
```bash
# Generate some activity
curl http://localhost:8080/health

# Check traces in Jaeger UI
echo "Open http://localhost:16686 and search for 'simpleidentity' service"
```

### 4. Run comprehensive test:
```bash
./test-stack.sh
```

## 🎯 Expected Results

### ✅ No More 404 Errors
- Metrics sent to OTEL Collector (port 4318)
- Traces sent to Jaeger (port 4318)
- Both work without conflicts

### ✅ Working Observability
- **Prometheus**: Scrapes metrics from OTEL Collector
- **Jaeger**: Receives traces directly from SimpleIdentity
- **Grafana**: Visualizes metrics from Prometheus

### ✅ Complete Telemetry Pipeline
- Metrics: SimpleIdentity → OTEL Collector → Prometheus → Grafana
- Traces: SimpleIdentity → Jaeger → Jaeger UI

## 🔍 Troubleshooting

### Check OTEL Collector logs:
```bash
docker-compose logs otel-collector
```

### Verify endpoints:
```bash
# OTEL Collector health
curl http://localhost:4319/v1/traces

# Prometheus targets
curl http://localhost:9090/api/v1/targets | grep otel-collector

# Jaeger traces
curl http://localhost:16686/api/services
```

### Validate configuration:
```bash
docker-compose config --quiet
```

## ✨ Benefits of the Fix

1. **Proper Separation**: Metrics and traces use appropriate backends
2. **Scalable Architecture**: OTEL Collector can handle multiple services
3. **Standards Compliant**: Uses OTLP protocol correctly
4. **Future Ready**: Easy to add more exporters (e.g., external monitoring)
5. **No More Errors**: Clean logs without 404 failures

The SimpleIdentity service now has a **production-ready observability stack** with proper telemetry data routing! 🚀
