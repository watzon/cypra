// Package logging provides structured logging helpers.
package logging

//revive:disable:exported

import (
	"context"
	"log/slog"
	"strings"
)

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
	return strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "email_body")
}
