# 🧪 Quick Docker Compose Testing Commands

## 🚀 **Start & Basic Testing**

```bash
# 1. Start the complete stack
docker-compose up -d

# 2. Check all containers are running
docker-compose ps

# 3. Test SimpleIdentity health
curl http://localhost:8080/health

# 4. View logs
docker-compose logs -f simpleidentity
```

## 🏥 **Health Check Tests**

```bash
# Main health endpoint
curl http://localhost:8080/health

# Kubernetes-style probes
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Container health check command
docker-compose exec simpleidentity /usr/local/bin/simpleidentity health
```

## 📊 **Observability Stack Tests**

```bash
# Jaeger (Distributed Tracing)
curl http://localhost:16686/api/services
echo "Jaeger UI: http://localhost:16686"

# Prometheus (Metrics)
curl http://localhost:9090/api/v1/targets
echo "Prometheus UI: http://localhost:9090"

# Grafana (Dashboards)
curl -u admin:admin123 http://localhost:3000/api/health
echo "Grafana UI: http://localhost:3000 (admin/admin123)"
```

## 🔧 **Debug & Development**

```bash
# pprof debug endpoints
curl http://localhost:6060/debug/pprof/

# View service logs
docker-compose logs simpleidentity | tail -20

# Check resource usage
docker stats --no-stream

# Network connectivity
docker-compose exec simpleidentity ping jaeger
```

## 🧪 **Automated Testing**

```bash
# Run comprehensive test script
./test-stack.sh

# Or run specific tests manually:
# Test all health endpoints
for endpoint in health health/live health/ready; do
  echo "Testing /$endpoint"
  curl -f http://localhost:8080/$endpoint && echo " ✅" || echo " ❌"
done
```

## 🔄 **Load Testing (Optional)**

```bash
# Simple load test (requires 'hey' tool)
hey -n 100 -c 10 http://localhost:8080/health

# Or use curl in a loop
for i in {1..20}; do
  curl -s http://localhost:8080/health > /dev/null &
done
wait
echo "Load test completed"
```

## 🧹 **Cleanup**

```bash
# Stop services
docker-compose stop

# Stop and remove containers
docker-compose down

# Remove volumes too (if needed)
docker-compose down -v

# View what's been removed
docker-compose ps
```

## 📋 **Quick Validation Checklist**

- [ ] `docker-compose ps` shows all containers running
- [ ] `curl http://localhost:8080/health` returns JSON with "status":"ok"
- [ ] Jaeger UI loads at http://localhost:16686
- [ ] Prometheus UI loads at http://localhost:9090
- [ ] Grafana UI loads at http://localhost:3000
- [ ] pprof debug available at http://localhost:6060/debug/pprof/
- [ ] No error messages in `docker-compose logs`

## 🎯 **Expected Results**

### Healthy Response Example:
```json
{
  "status": "ok",
  "timestamp": "2025-08-03T22:30:15Z",
  "checks": {
    "startup": {
      "status": "pass"
    }
  }
}
```

### Container Status Example:
```
NAME               COMMAND                  SERVICE          STATUS          PORTS
grafana            "/run.sh"                grafana          running         0.0.0.0:3000->3000/tcp
jaeger             "/go/bin/all-in-one"     jaeger           running         0.0.0.0:4317-4318->4317-4318/tcp, 0.0.0.0:16686->16686/tcp
prometheus         "/bin/prometheus --c…"   prometheus       running         0.0.0.0:9090->9090/tcp
simpleidentity     "/usr/local/bin/simp…"   simpleidentity   running         0.0.0.0:6060->6060/tcp, 0.0.0.0:8080->8080/tcp, 0.0.0.0:8090->8090/tcp, 0.0.0.0:9090->9090/tcp
```

## 🚨 **Common Issues & Solutions**

### Port Conflicts
```bash
# Check what's using the ports
netstat -tulpn | grep -E '(8080|9090|16686|3000|6060)'

# Use alternative ports if needed
docker-compose down
# Edit docker-compose.yml port mappings
docker-compose up -d
```

### Service Not Starting
```bash
# Check logs for specific service
docker-compose logs [service-name]

# Restart specific service
docker-compose restart [service-name]

# Force rebuild if needed
docker-compose build --no-cache [service-name]
```

### Connectivity Issues
```bash
# Check network
docker network ls
docker network inspect accounts-service_simpleidentity-net

# Test internal connectivity
docker-compose exec simpleidentity nslookup jaeger
```

This provides both automated testing via the script and manual commands for development workflow! 🎯
