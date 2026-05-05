package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
)

func TestHostedLoginRendersTenantBrandingAndFocusRing(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`UPDATE tenants SET name = 'Acme', branding = '{"display_name":"Acme Login","accent":"#767676","logo_url":"https://cdn.example/acme.svg","powered_by":true}'::jsonb WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("brand tenant: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/login", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Acme Login", "https://cdn.example/acme.svg", "--accent-primary: #767676", "--border-focus: #0D9488", "Powered by Cypra"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestTenantBrandingAccentValidationRejectsFailingAccent(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/"+tenantID.String()+"/branding", strings.NewReader(`{"accent":"#FFFF00"}`))
	req.Host = "cypra.localhost"
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "tenant.accent_contrast_failed") || !strings.Contains(rec.Body.String(), "ratio") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHostedLoginHTMXPasswordFailureUsesUniformError(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/login/password", strings.NewReader("email=user@example.com&password=bad"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), uniformAuthError) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func newHostedLoginTestServer(t *testing.T, harness *dbtest.Harness) *Server {
	t.Helper()
	server, err := New(Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", DevOpenAPI: true, KEKLoaded: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return server
}
