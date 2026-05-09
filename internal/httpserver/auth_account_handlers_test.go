package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestGetUserMeReturnsProfile(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, display_name, metadata) VALUES ($1, $2, 'chris@example.com', 'Chris Watson', '{"theme":"dark"}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("get me response = %d %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, want := range []string{
		`"email":"chris@example.com"`,
		`"display_name":"Chris Watson"`,
		`"theme":"dark"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("get me missing %q in %s", want, body)
		}
	}
}

func TestGetUserMeFallsBackToEmailLocalPart(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	// display_name defaults to '' so the handler should derive from the email local part.
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'priya@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"display_name":"priya"`) {
		t.Fatalf("expected priya fallback, got %d %s", resp.Code, resp.Body.String())
	}
}

func TestAuthListPasskeysReturnsCallerOwnedRows(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	otherUserID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb), ($3, $2, 'b@example.com', '{}'::jsonb)`, userID, tenantID, otherUserID); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id, nickname) VALUES ($1, $2, '\xaa', 'pk'::bytea, 'acme.cypra.localhost', 'MacBook')`, tenantID, userID); err != nil {
		t.Fatalf("seed passkey caller: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id, nickname) VALUES ($1, $2, '\xbb', 'pk'::bytea, 'acme.cypra.localhost', 'OtherUser')`, tenantID, otherUserID); err != nil {
		t.Fatalf("seed passkey other: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/passkeys", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list passkeys = %d %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"label":"MacBook"`) {
		t.Fatalf("list passkeys missing caller label: %s", body)
	}
	if strings.Contains(body, `"label":"OtherUser"`) {
		t.Fatalf("list passkeys leaked another user's row: %s", body)
	}
}

func TestAuthDeletePasskeyLastRemainingGuard(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	pkID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (id, tenant_id, user_id, credential_id, public_key, rp_id) VALUES ($1, $2, $3, '\xaa', 'pk'::bytea, 'acme.cypra.localhost')`, pkID, tenantID, userID); err != nil {
		t.Fatalf("seed passkey: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/passkeys/"+pkID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict || !strings.Contains(resp.Body.String(), "passkey.last_remaining") {
		t.Fatalf("expected 409 last-remaining guard, got %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM passkey_credentials WHERE id = $1`, 1, pkID)
}

func TestAuthDeletePasskeySucceedsWithMultiple(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	pkA := uuid.New()
	pkB := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (id, tenant_id, user_id, credential_id, public_key, rp_id) VALUES ($1, $3, $4, '\xaa', 'pk'::bytea, 'acme.cypra.localhost'), ($2, $3, $4, '\xbb', 'pk'::bytea, 'acme.cypra.localhost')`, pkA, pkB, tenantID, userID); err != nil {
		t.Fatalf("seed passkeys: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/passkeys/"+pkA.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM passkey_credentials WHERE user_id = $1`, 1, userID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE action = 'passkey.delete' AND resource_id = $1`, 1, pkA)
}

func TestAuthListSessionsFlagsCurrent(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	currentID := uuid.New()
	otherID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at, user_agent) VALUES ($1, $3, 'user', $4, now() + interval '1 hour', 'Chrome'), ($2, $3, 'user', $4, now() + interval '1 hour', 'Firefox')`, currentID, otherID, userID, tenantID); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	req.Header.Set("X-Cypra-Session-Id", currentID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list sessions = %d %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, currentID.String()) || !strings.Contains(body, otherID.String()) {
		t.Fatalf("list sessions missing rows: %s", body)
	}
	currentSegment := body
	if idx := strings.Index(currentSegment, currentID.String()); idx >= 0 {
		// Look for "current":true near the current id; the sessions are returned ordered by last_seen_at DESC.
		surrounding := currentSegment[idx:]
		if !strings.Contains(surrounding[:min(400, len(surrounding))], `"current":true`) {
			t.Fatalf("expected current=true on current session, got %s", body)
		}
	}
}

func TestAuthRevokeSessionRefusesCurrent(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, now() + interval '1 hour')`, sessionID, userID, tenantID); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+sessionID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	req.Header.Set("X-Cypra-Session-Id", sessionID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict || !strings.Contains(resp.Body.String(), "session.cannot_revoke_current") {
		t.Fatalf("expected 409 cannot-revoke-current, got %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND revoked_at IS NULL`, 1, sessionID)
}

func TestAuthRevokeSessionRevokesOther(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	currentID := uuid.New()
	otherID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $3, 'user', $4, now() + interval '1 hour'), ($2, $3, 'user', $4, now() + interval '1 hour')`, currentID, otherID, userID, tenantID); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/"+otherID.String(), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	req.Header.Set("X-Cypra-Session-Id", currentID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND revoked_at IS NOT NULL`, 1, otherID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND revoked_at IS NULL`, 1, currentID)
}

func TestAuthListMFAFactorsBasics(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'a@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO totp_credentials (tenant_id, user_id, secret_encrypted, confirmed_at) VALUES ($1, $2, '\xaa'::bytea, now())`, tenantID, userID); err != nil {
		t.Fatalf("seed totp: %v", err)
	}
	for range 6 {
		if _, err := harness.SQL.Exec(`INSERT INTO user_backup_codes (tenant_id, user_id, code_hash) VALUES ($1, $2, '\xaa'::bytea)`, tenantID, userID); err != nil {
			t.Fatalf("seed backup code: %v", err)
		}
	}
	if _, err := harness.SQL.Exec(`INSERT INTO user_backup_codes (tenant_id, user_id, code_hash, used_at) VALUES ($1, $2, '\xbb'::bytea, now()), ($1, $2, '\xcc'::bytea, now())`, tenantID, userID); err != nil {
		t.Fatalf("seed used backup codes: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/mfa/factors", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("mfa factors = %d %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, want := range []string{
		`"enrolled":true`,
		`"remaining":6`,
		`"total":8`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("mfa missing %q in %s", want, body)
		}
	}
}

func TestListTenantsEnrichedFields(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userA := uuid.New()
	userB := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $3, 'a@example.com', '{}'::jsonb), ($2, $3, 'b@example.com', '{}'::jsonb)`, userA, userB, tenantID); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_memberships (tenant_id, user_id, role) VALUES ($1, $2, 'admin')`, tenantID, userA); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'console', 'Console')`, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/", nil)
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list tenants = %d %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, want := range []string{
		`"member_count":1`,
		`"user_count":2`,
		`"project_count":1`,
		`"email_provider_required":true`,
		`"created_at"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("list tenants missing %q in %s", want, body)
		}
	}
}
