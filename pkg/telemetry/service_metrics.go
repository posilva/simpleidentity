package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Common metric instruments for services
type ServiceMetrics struct {
	RequestDuration metric.Float64Histogram
	RequestCount    metric.Int64Counter
	ActiveRequests  metric.Int64UpDownCounter
	ErrorCount      metric.Int64Counter
}

// NewServiceMetrics creates common service metrics
func (i *Instrumenter) NewServiceMetrics(serviceName string) (*ServiceMetrics, error) {
	requestDuration, err := i.meter.Float64Histogram(
		fmt.Sprintf("%s_request_duration_seconds", serviceName),
		metric.WithDescription("Duration of requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration histogram: %w", err)
	}

	requestCount, err := i.meter.Int64Counter(
		fmt.Sprintf("%s_requests_total", serviceName),
		metric.WithDescription("Total number of requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request count counter: %w", err)
	}

	activeRequests, err := i.meter.Int64UpDownCounter(
		fmt.Sprintf("%s_active_requests", serviceName),
		metric.WithDescription("Number of active requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active requests gauge: %w", err)
	}

	errorCount, err := i.meter.Int64Counter(
		fmt.Sprintf("%s_errors_total", serviceName),
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create error count counter: %w", err)
	}

	return &ServiceMetrics{
		RequestDuration: requestDuration,
		RequestCount:    requestCount,
		ActiveRequests:  activeRequests,
		ErrorCount:      errorCount,
	}, nil
}

// Middleware creates a middleware function for HTTP handlers
func (sm *ServiceMetrics) Middleware(next func(ctx context.Context) error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		start := time.Now()
		
		// Increment active requests
		sm.ActiveRequests.Add(ctx, 1)
		defer sm.ActiveRequests.Add(ctx, -1)
		
		// Execute the handler
		err := next(ctx)
		
		// Record metrics
		duration := time.Since(start).Seconds()
		
		attrs := []attribute.KeyValue{}
		if err != nil {
			attrs = append(attrs, attribute.String("status", "error"))
			sm.ErrorCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		} else {
			attrs = append(attrs, attribute.String("status", "success"))
		}
		
		sm.RequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		sm.RequestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
		
		return err
	}
}
