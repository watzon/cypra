package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
)

func TestRegistrationSettingsRoundtrip(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)

	body := `{"mode":"restricted","allowlist":["*@Example.com","alice@Example.com","alice@example.com","bogus"],"invites_enabled":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/registration", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status = %d body = %s", rec.Code, rec.Body.String())
	}

	var stored []string
	var mode string
	var invitesEnabled bool
	if err := harness.SQL.QueryRow(`SELECT signup_mode::text, signup_allowlist, invites_enabled FROM tenants WHERE id = $1`, tenantID).Scan(&mode, pqArray(&stored), &invitesEnabled); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if mode != "restricted" || invitesEnabled {
		t.Fatalf("got mode=%s invites_enabled=%v", mode, invitesEnabled)
	}
	// Expect normalized + dedupd: bogus dropped, two alice entries collapse.
	want := []string{"*@example.com", "alice@example.com"}
	if len(stored) != len(want) {
		t.Fatalf("allowlist = %v, want %v", stored, want)
	}
	for i, v := range want {
		if stored[i] != v {
			t.Fatalf("allowlist[%d] = %q, want %q", i, stored[i], v)
		}
	}
}

func TestRegistrationSettingsRejectsInvalidMode(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)

	body := `{"mode":"limbo","allowlist":[],"invites_enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/registration", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "registration_mode_invalid") {
		t.Fatalf("expected 400 mode_invalid, got %d %s", rec.Code, rec.Body.String())
	}
}

func TestSignupClosedBlocksHostedSignup(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`UPDATE tenants SET signup_mode = 'closed' WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set closed: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)

	body := strings.NewReader("email=newbie@example.com&password=correct horse battery staple")
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/signup", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 signup_closed, got %d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "auth.signup_closed") {
		t.Fatalf("expected signup_closed in body, got %s", rec.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE tenant_id = $1`, 0, tenantID)
}

func TestRestrictedModeAllowsMatchingEmail(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`UPDATE tenants SET signup_mode = 'restricted', signup_allowlist = ARRAY['*@example.com'] WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set restricted: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)

	rejected := strings.NewReader("email=alice@gmail.com&password=correct horse battery staple")
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/signup", rejected)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "auth.signup_restricted") {
		t.Fatalf("expected restricted rejection, got %d %s", rec.Code, rec.Body.String())
	}

	accepted := strings.NewReader("email=alice@example.com&password=correct horse battery staple")
	req = httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/signup", accepted)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected accepted signup, got %d %s", rec.Code, rec.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE tenant_id = $1 AND email = 'alice@example.com'`, 1, tenantID)
}

func TestSignupPageRendersAllEnabledMethods(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'password', true), ($1, 'magic_link', true), ($1, 'passkey', true)`, tenantID); err != nil {
		t.Fatalf("seed methods: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE tenants SET default_auth_method = 'magic_link'::tenant_auth_method WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set default: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/signup", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	// magic-link is the primary signup method per default_auth_method.
	if !strings.Contains(body, `>Send a sign-up link</button>`) {
		t.Fatalf("expected magic-link primary 'Send a sign-up link', got: %s", body)
	}
	// password + passkey should appear as secondary alt-rows.
	if !strings.Contains(body, "Use a password") {
		t.Fatalf("expected password alt-row, got: %s", body)
	}
	if !strings.Contains(body, "Continue with passkey") {
		t.Fatalf("expected passkey alt-row, got: %s", body)
	}
	// Hidden method markers must be present so the backend dispatch can see them.
	if !strings.Contains(body, `name="method" value="magic_link"`) || !strings.Contains(body, `name="method" value="password"`) {
		t.Fatalf("expected hidden method inputs, got: %s", body)
	}
}

func TestSignupPageRendersOnlyEnabledMethods(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'magic_link', true), ($1, 'passkey', true)`, tenantID); err != nil {
		t.Fatalf("seed methods: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/signup", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	body := rec.Body.String()
	if strings.Contains(body, "Use a password") || strings.Contains(body, `name="method" value="password"`) {
		t.Fatalf("password should not render when disabled: %s", body)
	}
	for _, want := range []string{"Send a sign-up link", "Continue with passkey"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
}

func TestSignupViaMagicLinkCreatesUserAndIssuesToken(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'magic_link', true)`, tenantID); err != nil {
		t.Fatalf("seed methods: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO email_provider_configs (tenant_id, kind, from_address, from_name, config_encrypted, enabled) VALUES ($1, 'terminal', 'noreply@example.com', 'Cypra', '{}'::bytea, true)`, tenantID); err != nil {
		t.Fatalf("seed email provider: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	body := strings.NewReader("method=magic_link&email=newbie@example.com")
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/signup", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE tenant_id = $1 AND email = 'newbie@example.com'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM magic_link_tokens WHERE tenant_id = $1 AND email = 'newbie@example.com'`, 1, tenantID)
}

func TestSignupViaMagicLinkRequiresEmailProvider(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'magic_link', true)`, tenantID); err != nil {
		t.Fatalf("seed methods: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	body := strings.NewReader("method=magic_link&email=newbie@example.com")
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/signup", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE tenant_id = $1 AND email = 'newbie@example.com'`, 0, tenantID)
}

func TestSignupPasskeyBeginCreatesUserAndReturnsCeremony(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'passkey', true)`, tenantID); err != nil {
		t.Fatalf("seed methods: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/signup/passkey", strings.NewReader(`{"email":"pkuser@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var payload struct {
		CeremonyID string `json:"ceremony_id"`
		UserID     string `json:"user_id"`
		Options    any    `json:"options"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v body=%s", err, rec.Body.String())
	}
	if payload.CeremonyID == "" || payload.UserID == "" || payload.Options == nil {
		t.Fatalf("expected ceremony_id+user_id+options, got %+v", payload)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE tenant_id = $1 AND email = 'pkuser@example.com'`, 1, tenantID)
}

// pqArray adapts a *[]string to satisfy sql.Scanner via lib/pq's StringArray.
func pqArray(dest *[]string) interface{ Scan(any) error } {
	return (*scanArray)(dest)
}

type scanArray []string

func (s *scanArray) Scan(src any) error {
	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	}
	// pq encodes string arrays as "{a,b,c}". Tiny parser is sufficient for tests.
	body := strings.TrimSpace(string(raw))
	body = strings.TrimPrefix(body, "{")
	body = strings.TrimSuffix(body, "}")
	if body == "" {
		*s = nil
		return nil
	}
	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.Trim(part, `"`))
	}
	*s = out
	return nil
}

// Ensure encoding/json import isn't dead-pruned when we extend tests later.
var _ = json.Marshal
