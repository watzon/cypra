package httpserver_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPhase3TenantScopedRoutesUseResolvedTenant(t *testing.T) {
	harness := dbtest.New(t)
	acmeID := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravoID := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	if _, err := harness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'acme-app', 'Acme App')`, acmeID); err != nil {
		t.Fatalf("seed acme project: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'bravo-app', 'Bravo App')`, bravoID); err != nil {
		t.Fatalf("seed bravo project: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/projects/", nil)
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	newTestServer(t, harness).Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("project list status = %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "acme-app") || strings.Contains(body, "bravo-app") {
		t.Fatalf("project route tenant isolation failed: %s", body)
	}
}

func TestPhase4AuthRoutesUseResolvedTenant(t *testing.T) {
	harness := dbtest.New(t)
	acmeID := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravoID := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	userID := uuid.New()
	credentialID := []byte("credential")
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, acmeID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id) VALUES ($1, $2, $3, 'public'::bytea, 'acme.cypra.localhost')`, acmeID, userID, credentialID); err != nil {
		t.Fatalf("seed passkey: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'other@example.com', '{}'::jsonb)`, uuid.New(), bravoID); err != nil {
		t.Fatalf("seed bravo user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "https://bravo.cypra.localhost/api/v1/auth/passkey/assert", bytes.NewReader([]byte(`{"user_id":"`+userID.String()+`","credential_id":"Y3JlZGVudGlhbA","public_key":""}`)))
	resp := httptest.NewRecorder()
	newTestServer(t, harness).Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("cross-tenant passkey status = %d body=%s", resp.Code, resp.Body.String())
	}
}
