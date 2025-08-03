package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Common attribute helpers
func UserIDAttr(userID string) attribute.KeyValue {
	return attribute.String("user.id", userID)
}

func ProviderAttr(provider string) attribute.KeyValue {
	return attribute.String("auth.provider", provider)
}

func OperationAttr(operation string) attribute.KeyValue {
	return attribute.String("operation", operation)
}

func StatusAttr(status string) attribute.KeyValue {
	return attribute.String("status", status)
}

func ErrorTypeAttr(err error) attribute.KeyValue {
	return attribute.String("error.type", fmt.Sprintf("%T", err))
}

// TraceOperation is a helper to trace a function execution
func TraceOperation(ctx context.Context, tracer trace.Tracer, operationName string, fn func(context.Context, trace.Span) error) error {
	ctx, span := tracer.Start(ctx, operationName)
	defer span.End()
	
	if err := fn(ctx, span); err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}
	
	span.SetStatus(codes.Ok, "")
	return nil
}