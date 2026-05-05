package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
)

func TestHealthReadyAndTenantCRUD(t *testing.T) {
	harness := dbtest.New(t)
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("healthz status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("readyz status = %d body=%s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"acme","name":"Acme"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create tenant status = %d body=%s", resp.Code, resp.Body.String())
	}
	var tenant map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &tenant); err != nil {
		t.Fatalf("decode tenant: %v", err)
	}
	if tenant["slug"] != "acme" {
		t.Fatalf("tenant slug = %#v", tenant["slug"])
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"admin","name":"Admin"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest || !bytes.Contains(resp.Body.Bytes(), []byte("tenant.slug_reserved")) {
		t.Fatalf("reserved slug response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"1bad","name":"Bad"}`)))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest || !bytes.Contains(resp.Body.Bytes(), []byte("tenant.slug_invalid")) {
		t.Fatalf("invalid slug response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", bytes.NewReader([]byte(`{"slug":"bravo","name":"Bravo"}`)))
	req.Host = "cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("non-admin tenant create status = %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	req.Host = "cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !bytes.Contains(resp.Body.Bytes(), []byte("/api/v1/tenants")) {
		t.Fatalf("openapi response = %d %s", resp.Code, resp.Body.String())
	}
}

func TestBotVerifierFailureBlocksAuthHandler(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, BotVerifier: failingVerifier{}})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password/signup", bytes.NewReader([]byte(`{"email":"user@example.com","password":"correct horse battery staple","bot_token":"bad"}`)))
	req.Host = "acme.cypra.localhost"
	resp := httptest.NewRecorder()
	server.Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden || !bytes.Contains(resp.Body.Bytes(), []byte("bot.verify_failed")) {
		t.Fatalf("bot failure response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE email = 'user@example.com'`, 0)
}

func TestTenantResolverProjectCRUDAndAudit(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/", bytes.NewReader([]byte(`{"slug":"app","name":"App"}`)))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create project status = %d body=%s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM projects WHERE slug = 'app'`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE action = 'project.create'`, 1)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/projects/", nil)
	req.Host = "unknown.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("unknown tenant status = %d", resp.Code)
	}
}

func newTestServer(t *testing.T, harness *dbtest.Harness) *httpserver.Server {
	t.Helper()
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", DevOpenAPI: true, KEKLoaded: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return server
}

type failingVerifier struct{}

func (failingVerifier) Verify(context.Context, string) error { return errors.New("bot rejected") }
