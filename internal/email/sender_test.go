package email_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/email"
)

func TestRenderKnownTemplate(t *testing.T) {
	message, err := email.Render("magic-link", "user@example.com", json.RawMessage(`{"magic_link_url":"https://example.test/magic"}`))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if message.Subject == "magic-link" || !strings.Contains(message.Text, "https://example.test/magic") {
		t.Fatalf("message = %+v", message)
	}
}

func TestResendSenderPostsMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	sender := email.ResendSender{APIKey: "test-key", Endpoint: server.URL, FromAddress: "noreply@example.com"}
	if err := sender.Send(context.Background(), email.Message{To: "user@example.com", Subject: "Hello", Text: "Hi"}); err != nil {
		t.Fatalf("send resend: %v", err)
	}
}

func TestResolverRequiresTenantProvider(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	resolver := email.Resolver{DB: harness.SQL}
	if _, err := resolver.Resolve(context.Background(), tenantID); err != email.ErrProviderRequired {
		t.Fatalf("resolve error = %v", err)
	}
}

func TestResolverDecryptsTerminalProvider(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	kek := bytes.Repeat([]byte{7}, crypto.MasterKeyBytes)
	config, err := email.EncryptConfig([]byte(`{}`), kek)
	if err != nil {
		t.Fatalf("encrypt config: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO email_provider_configs (tenant_id, kind, config_encrypted, from_address, from_name) VALUES ($1, 'terminal', $2, 'noreply@example.com', 'Cypra')`, tenantID, config); err != nil {
		t.Fatalf("seed provider: %v", err)
	}
	var out bytes.Buffer
	resolver := email.Resolver{DB: harness.SQL, KEK: kek, TerminalWriter: &out}
	sender, err := resolver.Resolve(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := sender.Send(context.Background(), email.Message{To: "user@example.com", Subject: "Hi", Text: "Body"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(out.String(), "user@example.com") {
		t.Fatalf("terminal output = %q", out.String())
	}
}
