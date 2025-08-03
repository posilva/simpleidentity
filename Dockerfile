# SimpleIdentity - Multi-stage Dockerfile for Production
# Built with security, performance, and size optimization in mind

# ============================================================================
# Build Stage - Use official Go image for compilation
# ============================================================================
FROM golang:1.23.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create appuser for security (non-root execution)
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies (cached if go.mod/go.sum unchanged)
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build the binary with optimizations
# - Disable CGO for static binary
# - Remove debug info and symbol table for smaller size
# - Set build info for version tracking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o simpleidentity \
    ./cmd/simpleidentity

# Verify binary
RUN ./simpleidentity version

# ============================================================================
# Runtime Stage - Minimal distroless image for security
# ============================================================================
FROM gcr.io/distroless/static:nonroot

# Labels for container metadata
LABEL maintainer="SimpleIdentity Team" \
      description="SimpleIdentity - Secure Gaming Platform Identity Service" \
      version="1.0.0" \
      org.opencontainers.image.title="SimpleIdentity" \
      org.opencontainers.image.description="Enterprise-grade identity management for gaming platforms" \
      org.opencontainers.image.vendor="SimpleIdentity" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/posilva/simpleidentity"

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder stage
COPY --from=builder /build/simpleidentity /usr/local/bin/simpleidentity

# Set user to non-root for security (distroless nonroot user is UID 65532)
USER 65532:65532

# Expose ports (documentation only - actual ports configured via environment)
# Main application ports
EXPOSE 8090/tcp 9090/tcp
# Health check port
EXPOSE 8080/tcp
# pprof debug port (internal only)
EXPOSE 6060/tcp

# Environment variables with secure defaults
ENV SMPIDT_LOG_LEVEL=info \
    SMPIDT_LOG_PRETTY=false \
    SMPIDT_HEALTH_ADDR=:8080 \
    SMPIDT_HTTP_ADDR=:8090 \
    SMPIDT_GRPC_ADDR=:9090 \
    SMPIDT_PPROF_ADDR=:6060 \
    SMPIDT_SHUTDOWN_TIMEOUT=30s 

# Default command
ENTRYPOINT ["/usr/local/bin/simpleidentity"]
CMD ["server"]
