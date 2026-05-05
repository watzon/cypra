package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/pat"
)

func TestPATHandlersCreateListAndRevoke(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pats/", bytes.NewReader([]byte(`{"name":"dev","scopes":["projects.read"]}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create pat = %d %s", resp.Code, resp.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created["token"] == "" {
		t.Fatalf("created = %+v", created)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/pats/", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || bytes.Contains(resp.Body.Bytes(), []byte(created["token"].(string))) {
		t.Fatalf("list pat = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/pats/"+created["id"].(string), nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("revoke pat = %d %s", resp.Code, resp.Body.String())
	}
}

func TestPATBearerAuthAllowsAPIAndRevokedTokenIsUnauthorized(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	created, err := (pat.Service{DB: harness.SQL}).Create(context.Background(), tenantID, userID, "api", []string{"projects.read"})
	if err != nil {
		t.Fatalf("create pat: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("pat auth response = %d %s", resp.Code, resp.Body.String())
	}
	if err := (pat.Service{DB: harness.SQL}).Revoke(context.Background(), tenantID, userID, created.ID); err != nil {
		t.Fatalf("revoke pat: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("revoked pat response = %d %s", resp.Code, resp.Body.String())
	}
}
