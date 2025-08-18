package otelhelper

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func SetError(span trace.Span, err error, attrs ...attribute.KeyValue) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	span.AddEvent("error_occurred", trace.WithAttributes(
		attrs...,
	))
}
