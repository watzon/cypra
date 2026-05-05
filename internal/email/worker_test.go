package email_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/email"
)

func TestWorkerDispatchesTerminalEmail(t *testing.T) {
	harness := dbtest.New(t)
	var out bytes.Buffer
	if _, err := harness.SQL.Exec(`INSERT INTO email_outbox (to_address, template, payload) VALUES ('user@example.com', 'welcome', '{"body":"hi"}'::jsonb)`); err != nil {
		t.Fatalf("insert outbox row: %v", err)
	}
	worker := email.Worker{DB: harness.SQL, Sender: email.TerminalSender{Writer: &out}}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("user@example.com")) {
		t.Fatalf("terminal output = %q", out.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM email_outbox WHERE sent_at IS NOT NULL`, 1)
}
