package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestPATHandlersUseSessionSubjectWithoutUserHeader(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, $4)`, sessionID, userID, tenantID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pats/", bytes.NewReader([]byte(`{"name":"dev","scopes":["projects.read"]}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.AddCookie(&http.Cookie{Name: "cypra_session", Value: sessionID.String()})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create pat with session subject = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM personal_access_tokens WHERE tenant_id = $1 AND user_id = $2`, 1, tenantID, userID)
}

func TestPATHandlersRejectBareUserHeaderFallback(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pats/", bytes.NewReader([]byte(`{"name":"dev","scopes":["projects.read"]}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-User-Id", userID.String())
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden && resp.Code != http.StatusUnauthorized {
		t.Fatalf("bare user header pat create = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM personal_access_tokens WHERE tenant_id = $1 AND user_id = $2`, 0, tenantID, userID)
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
	req = httptest.NewRequest(http.MethodPost, "/api/v1/projects/", bytes.NewReader([]byte(`{"slug":"blocked","name":"Blocked"}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("pat forbidden response = %d %s", resp.Code, resp.Body.String())
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

func TestPATBearerAuthExpiredTokenIsUnauthorized(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	expiresAt := time.Now().Add(-time.Minute)
	created, err := (pat.Service{DB: harness.SQL}).CreateWithExpiry(context.Background(), tenantID, userID, "api", []string{"projects.read"}, &expiresAt)
	if err != nil {
		t.Fatalf("create pat: %v", err)
	}
	router := newTestServer(t, harness).Router()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expired pat response = %d %s", resp.Code, resp.Body.String())
	}
}
