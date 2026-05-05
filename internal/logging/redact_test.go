package logging_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/logging"
)

func TestRedactingHandlerRedactsSecretShapedFields(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(logging.RedactingHandler{Handler: slog.NewTextHandler(&out, nil)})
	logger.Info("test", slog.String("password", "secret"), slog.String("safe", "ok"))
	got := out.String()
	if strings.Contains(got, "secret") || !strings.Contains(got, "safe=ok") {
		t.Fatalf("redacted log = %q", got)
	}
}
