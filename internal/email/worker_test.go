package email_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/email"
	"github.com/watzon/cypra/internal/logging"
)

func TestWorkerDispatchesTerminalEmail(t *testing.T) {
	harness := dbtest.New(t)
	var out bytes.Buffer
	var logs bytes.Buffer
	if _, err := harness.SQL.Exec(`INSERT INTO email_outbox (to_address, template, payload) VALUES ('user@example.com', 'welcome', '{"body":"hi"}'::jsonb)`); err != nil {
		t.Fatalf("insert outbox row: %v", err)
	}
	worker := email.Worker{DB: harness.SQL, Sender: email.TerminalSender{Writer: &out}, Logger: slog.New(logging.RedactingHandler{Handler: slog.NewTextHandler(&logs, nil)})}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("user@example.com")) {
		t.Fatalf("terminal output = %q", out.String())
	}
	logged := logs.String()
	if !strings.Contains(logged, "request_id=") || !strings.Contains(logged, "tenant_id=") || !strings.Contains(logged, "actor_id=system") || !strings.Contains(logged, "to_email=[redacted]") {
		t.Fatalf("email dispatch log missing required redacted fields: %s", logged)
	}
	if strings.Contains(logged, "user@example.com") {
		t.Fatalf("email dispatch log leaked recipient: %s", logged)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM email_outbox WHERE sent_at IS NOT NULL`, 1)
}

func TestWorkerUsesExponentialBackoffForRetryableFailures(t *testing.T) {
	harness := dbtest.New(t)
	if _, err := harness.SQL.Exec(`INSERT INTO email_outbox (to_address, template, payload, attempts) VALUES ('user@example.com', 'welcome', '{"body":"hi"}'::jsonb, 1)`); err != nil {
		t.Fatalf("insert outbox row: %v", err)
	}
	worker := email.Worker{DB: harness.SQL, Sender: failingSender{}}
	started := time.Now()
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run worker: %v", err)
	}
	var attempts int
	var nextAttempt time.Time
	if err := harness.SQL.QueryRow(`SELECT attempts, next_attempt_at FROM email_outbox WHERE to_address = 'user@example.com'`).Scan(&attempts, &nextAttempt); err != nil {
		t.Fatalf("read outbox row: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
	if nextAttempt.Before(started.Add(90 * time.Second)) {
		t.Fatalf("next_attempt_at did not use exponential backoff: %s", nextAttempt)
	}
}

type failingSender struct{}

func (failingSender) Send(context.Context, email.Message) error {
	return errors.New("temporary failure")
}
