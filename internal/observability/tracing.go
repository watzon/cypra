// Package observability contains small OpenTelemetry helpers shared by runtime paths.
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartSpan starts a Cypra OpenTelemetry span with the shared tracer name.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return otel.Tracer("github.com/watzon/cypra").Start(ctx, name, trace.WithAttributes(attrs...))
}

// RecordError marks span as failed when err is non-nil.
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
