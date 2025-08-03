package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// HTTPMiddleware creates HTTP middleware for automatic instrumentation
type HTTPMiddleware struct {
	tracer  trace.Tracer
	metrics *ServiceMetrics
}

// NewHTTPMiddleware creates a new HTTP middleware
func NewHTTPMiddleware(serviceName string) (*HTTPMiddleware, error) {
	instrumenter := NewInstrumenter(fmt.Sprintf("%s_http", serviceName))
	
	metrics, err := instrumenter.NewServiceMetrics("http")
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP metrics: %w", err)
	}
	
	return &HTTPMiddleware{
		tracer:  instrumenter.tracer,
		metrics: metrics,
	}, nil
}

// Handler wraps an HTTP handler with telemetry
func (m *HTTPMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tracing context from headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		
		// Start span
		spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		ctx, span := m.tracer.Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.route", r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.remote_addr", r.RemoteAddr),
			),
		)
		defer span.End()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Record active request
		start := time.Now()
		m.metrics.ActiveRequests.Add(ctx, 1)
		defer m.metrics.ActiveRequests.Add(ctx, -1)
		
		// Call next handler
		next.ServeHTTP(wrapped, r.WithContext(ctx))
		
		// Record metrics and span attributes
		duration := time.Since(start).Seconds()
		statusCode := wrapped.statusCode
		
		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int64("http.response_size", wrapped.bytesWritten),
		)
		
		// Set span status based on HTTP status code
		if statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
			m.metrics.ErrorCount.Add(ctx, 1, metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.Int("http.status_code", statusCode),
			))
		} else {
			span.SetStatus(codes.Ok, "")
		}
		
		// Record metrics
		attrs := []attribute.KeyValue{
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
			attribute.Int("http.status_code", statusCode),
		}
		
		m.metrics.RequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		m.metrics.RequestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)
	return n, err
}

// GRPCInterceptors provides gRPC interceptors for telemetry
type GRPCInterceptors struct {
	tracer  trace.Tracer
	metrics *ServiceMetrics
}

// NewGRPCInterceptors creates new gRPC interceptors
func NewGRPCInterceptors(serviceName string) (*GRPCInterceptors, error) {
	instrumenter := NewInstrumenter(fmt.Sprintf("%s_grpc", serviceName))
	
	metrics, err := instrumenter.NewServiceMetrics("grpc")
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC metrics: %w", err)
	}
	
	return &GRPCInterceptors{
		tracer:  instrumenter.tracer,
		metrics: metrics,
	}, nil
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor for tracing
func (i *GRPCInterceptors) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract tracing context from gRPC metadata
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataCarrier{md: md})
		
		// Start span
		ctx, span := i.tracer.Start(ctx, info.FullMethod,
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", info.FullMethod),
				attribute.String("rpc.method", info.FullMethod),
			),
		)
		defer span.End()
		
		// Record active request
		start := time.Now()
		i.metrics.ActiveRequests.Add(ctx, 1)
		defer i.metrics.ActiveRequests.Add(ctx, -1)
		
		// Call handler
		resp, err := handler(ctx, req)
		
		// Record metrics and span attributes
		duration := time.Since(start).Seconds()
		
		if err != nil {
			// Extract gRPC status
			grpcStatus := status.Convert(err)
			
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", grpcStatus.Code().String()),
				attribute.String("error.message", grpcStatus.Message()),
			)
			
			i.metrics.ErrorCount.Add(ctx, 1, metric.WithAttributes(
				attribute.String("rpc.method", info.FullMethod),
				attribute.String("rpc.grpc.status_code", grpcStatus.Code().String()),
			))
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", "OK"),
			)
		}
		
		// Record metrics
		statusCode := "OK"
		if err != nil {
			statusCode = status.Convert(err).Code().String()
		}
		
		attrs := []attribute.KeyValue{
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.grpc.status_code", statusCode),
		}
		
		i.metrics.RequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		i.metrics.RequestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		
		return resp, err
	}
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor for tracing
func (i *GRPCInterceptors) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		
		// Extract tracing context from gRPC metadata
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataCarrier{md: md})
		
		// Start span
		ctx, span := i.tracer.Start(ctx, info.FullMethod,
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", info.FullMethod),
				attribute.String("rpc.method", info.FullMethod),
				attribute.Bool("rpc.streaming", true),
			),
		)
		defer span.End()
		
		// Wrap stream with context
		wrappedStream := &serverStreamWrapper{
			ServerStream: stream,
			ctx:          ctx,
		}
		
		// Record active request
		start := time.Now()
		i.metrics.ActiveRequests.Add(ctx, 1)
		defer i.metrics.ActiveRequests.Add(ctx, -1)
		
		// Call handler
		err := handler(srv, wrappedStream)
		
		// Record metrics and span attributes
		duration := time.Since(start).Seconds()
		
		if err != nil {
			grpcStatus := status.Convert(err)
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", grpcStatus.Code().String()),
			)
			
			i.metrics.ErrorCount.Add(ctx, 1, metric.WithAttributes(
				attribute.String("rpc.method", info.FullMethod),
				attribute.String("rpc.grpc.status_code", grpcStatus.Code().String()),
			))
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", "OK"),
			)
		}
		
		// Record metrics
		statusCode := "OK"
		if err != nil {
			statusCode = status.Convert(err).Code().String()
		}
		
		attrs := []attribute.KeyValue{
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.grpc.status_code", statusCode),
		}
		
		i.metrics.RequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		i.metrics.RequestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		
		return err
	}
}

// metadataCarrier implements propagation.TextMapCarrier for gRPC metadata
type metadataCarrier struct {
	md metadata.MD
}

func (c *metadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *metadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

func (c *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for key := range c.md {
		keys = append(keys, key)
	}
	return keys
}

// serverStreamWrapper wraps grpc.ServerStream to inject context
type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *serverStreamWrapper) Context() context.Context {
	return w.ctx
}

// UnaryClientInterceptor returns a grpc.UnaryClientInterceptor for tracing outgoing calls
func (i *GRPCInterceptors) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Start span for outgoing call
		ctx, span := i.tracer.Start(ctx, method,
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", method),
				attribute.String("rpc.method", method),
				attribute.String("rpc.target", cc.Target()),
			),
		)
		defer span.End()
		
		// Inject tracing context into gRPC metadata
		md, _ := metadata.FromOutgoingContext(ctx)
		if md == nil {
			md = metadata.New(nil)
		}
		carrier := &metadataCarrier{md: md}
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, carrier.md)
		
		// Call the RPC
		err := invoker(ctx, method, req, reply, cc, opts...)
		
		if err != nil {
			grpcStatus := status.Convert(err)
			span.SetStatus(codes.Error, grpcStatus.Message())
			span.RecordError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", grpcStatus.Code().String()),
			)
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", "OK"),
			)
		}
		
		return err
	}
}
