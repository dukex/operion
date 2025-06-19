package tracer

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
	// Common attribute keys
	WorkflowIDKey     = "operion.workflow.id"
	WorkflowNameKey   = "operion.workflow.name"
	TriggerIDKey      = "operion.trigger.id"
	TriggerTypeKey    = "operion.trigger.type"
	ActionIDKey       = "operion.action.id"
	ActionTypeKey     = "operion.action.type"
	StepIDKey         = "operion.step.id"
	StepNameKey       = "operion.step.name"
	ExecutionIDKey    = "operion.execution.id"
	EventIDKey        = "operion.event.id"
	ServiceIDKey      = "operion.service.id"
	WorkerIDKey       = "operion.worker.id"
)

var globalTracerProvider *sdktrace.TracerProvider

func InitTracer(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	// Return existing global tracer provider if already initialized
	if globalTracerProvider != nil {
		return globalTracerProvider, nil
	}

	tp, err := createTracerProvider(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	// Set this as the global tracer provider
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	globalTracerProvider = tp
	return tp, nil
}

func createTracerProvider(ctx context.Context, serviceName string) (*sdktrace.TracerProvider, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
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

	return tp, nil
}

// StartSpan starts a new span with common attributes
func StartSpan(ctx context.Context, tracer trace.Tracer, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// GetTracer returns a tracer with a standardized name
func GetTracer(component string) trace.Tracer {
	return otel.Tracer("operion-" + component)
}

// WorkflowAttributes creates common workflow attributes
func WorkflowAttributes(id, name string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(WorkflowIDKey, id),
		attribute.String(WorkflowNameKey, name),
	}
}

// TriggerAttributes creates common trigger attributes
func TriggerAttributes(id, triggerType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(TriggerIDKey, id),
		attribute.String(TriggerTypeKey, triggerType),
	}
}

// ActionAttributes creates common action attributes
func ActionAttributes(id, actionType string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(ActionIDKey, id),
		attribute.String(ActionTypeKey, actionType),
	}
}

// StepAttributes creates common step attributes
func StepAttributes(id, name string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(StepIDKey, id),
		attribute.String(StepNameKey, name),
	}
}

// ExecutionAttributes creates common execution attributes
func ExecutionAttributes(id string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(ExecutionIDKey, id),
	}
}
