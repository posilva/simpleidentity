package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Instrumenter provides common telemetry instrumentation helpers
type Instrumenter struct {
	tracer trace.Tracer
	meter  metric.Meter
}

// NewInstrumenter creates a new instrumenter for a component
func NewInstrumenter(componentName string) *Instrumenter {
	return &Instrumenter{
		tracer: otel.Tracer(componentName),
		meter:  otel.Meter(componentName),
	}
}

// StartSpan starts a new span with common attributes
func (i *Instrumenter) StartSpan(ctx context.Context, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return i.tracer.Start(ctx, operationName, trace.WithAttributes(attrs...))
}

// RecordError records an error in the current span
func (i *Instrumenter) RecordError(span trace.Span, err error, description string) {
	if err != nil {
		span.SetStatus(codes.Error, description)
		span.RecordError(err, trace.WithAttributes(
			attribute.String("error.type", fmt.Sprintf("%T", err)),
			attribute.String("error.message", err.Error()),
		))
	}
}

// SetSpanAttributes sets attributes on a span
func (i *Instrumenter) SetSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}
