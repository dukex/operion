// Package otelhelper provides distributed tracing functionality for workflow monitoring.
package otelhelper

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otlptracehttp "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Common attribute keys.
	WorkflowIDKey   = "operion.workflow.id"
	WorkflowNameKey = "operion.workflow.name"
	TriggerIDKey    = "operion.trigger.id"
	TriggerTypeKey  = "operion.trigger.type"
	ActionIDKey     = "operion.action.id"
	ActionTypeKey   = "operion.action.type"
	StepIDKey       = "operion.step.id"
	StepNameKey     = "operion.step.name"
	ExecutionIDKey  = "operion.execution.id"
	EventIDKey      = "operion.event.id"
	ServiceIDKey    = "operion.service.id"
	WorkerIDKey     = "operion.worker.id"
)

// nolint:ireturn // Returning interface is intentional for OpenTelemetry tracing
func NewTracer(ctx context.Context, serviceName string) (trace.Tracer, error) {
	provider, err := newTracerProvider(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	return provider.Tracer(serviceName), nil
}

// nolint:ireturn,spancheck // Returning interface is intentional for OpenTelemetry tracing
func StartSpan(ctx context.Context, tracer trace.Tracer, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

func newTracerProvider(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(r),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}))

	return tp, nil
}
