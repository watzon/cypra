// Package logging provides structured logging helpers.
package logging

//revive:disable:exported

import (
	"context"
	"log/slog"
	"strings"
)

type requestFieldsKey struct{}

type RequestFields struct {
	RequestID string
	TenantID  string
	ActorID   string
}

func ContextWithRequestFields(ctx context.Context, fields *RequestFields) context.Context {
	return context.WithValue(ctx, requestFieldsKey{}, fields)
}

func SetTenantID(ctx context.Context, tenantID string) {
	if fields, ok := ctx.Value(requestFieldsKey{}).(*RequestFields); ok {
		fields.TenantID = tenantID
	}
}

func SetActorID(ctx context.Context, actorID string) {
	if fields, ok := ctx.Value(requestFieldsKey{}).(*RequestFields); ok {
		fields.ActorID = actorID
	}
}

func RequestFieldsFromContext(ctx context.Context) RequestFields {
	if fields, ok := ctx.Value(requestFieldsKey{}).(*RequestFields); ok && fields != nil {
		return *fields
	}
	return RequestFields{}
}

type RedactingHandler struct{ slog.Handler }

func (h RedactingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		if isSensitive(attr.Key) {
			attr.Value = slog.StringValue("[redacted]")
		}
		redacted.AddAttrs(attr)
		return true
	})
	return h.Handler.Handle(ctx, redacted)
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "email")
}
