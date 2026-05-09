package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/db"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestTenantScopedHandlerOverridesTamperedDBTenantContext(t *testing.T) {
	harness := dbtest.New(t)
	acmeID := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravoID := dbtest.SeedTenant(t, harness.SQL, "bravo")
	seedProject(t, harness, acmeID, "acme-app")
	seedProject(t, harness, bravoID, "bravo-app")

	server, err := New(Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	handler := server.tenantResolver(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.listProjects(w, r.WithContext(db.ContextWithTenant(r.Context(), bravoID)))
	}))

	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/api/v1/projects/", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "acme-app") {
		t.Fatalf("response missing acme project: %s", body)
	}
	if strings.Contains(body, "bravo-app") {
		t.Fatalf("response leaked bravo project under tampered tenant context: %s", body)
	}
}

func TestInstallDashboardTenantRouteResolvesTenantScopedAPIFromReferer(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	seedProject(t, harness, tenantID, "acme-app")

	server, err := New(Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://cypra.localhost/api/v1/projects/", nil)
	req.Header.Set("Referer", "https://cypra.localhost/dashboard/tenants/acme/projects")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	server.Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "acme-app") {
		t.Fatalf("response missing tenant project: %s", resp.Body.String())
	}
}

func TestInstallDashboardTenantHeaderResolvesTenantScopedAPI(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	seedProject(t, harness, tenantID, "acme-app")

	server, err := New(Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://cypra.localhost/api/v1/projects/", nil)
	req.Header.Set("X-Cypra-Tenant-Slug", "acme")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	server.Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "acme-app") {
		t.Fatalf("response missing tenant project: %s", resp.Body.String())
	}
}

func seedProject(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID, slug string) {
	t.Helper()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, $3, $4)`, uuid.New(), tenantID, slug, slug); err != nil {
		t.Fatalf("seed project %s: %v", slug, err)
	}
}
