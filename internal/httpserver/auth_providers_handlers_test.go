package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
)

func newAuthProvidersTestServer(t *testing.T, harness *dbtest.Harness) *Server {
	t.Helper()
	return newHostedLoginTestServer(t, harness)
}

func TestSaveAuthProviderRoundtripsConfig(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newAuthProvidersTestServer(t, harness)

	body := `{"enabled":true,"config":{"min_length":18,"require_upper":true}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/password", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status = %d body = %s", rec.Code, rec.Body.String())
	}

	var stored []byte
	if err := harness.SQL.QueryRow(`SELECT config FROM tenant_auth_methods WHERE tenant_id = $1 AND method = 'password'`, tenantID).Scan(&stored); err != nil {
		t.Fatalf("read config: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(stored, &got); err != nil {
		t.Fatalf("unmarshal stored config: %v", err)
	}
	if got["min_length"].(float64) != 18 || got["require_upper"] != true {
		t.Fatalf("stored config = %+v, want min_length=18 require_upper=true", got)
	}
}

func TestSaveAuthProviderRejectsUnknownConfigField(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newAuthProvidersTestServer(t, harness)
	body := `{"enabled":true,"config":{"made_up_field":"oops"}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/password", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "auth_providers.config_invalid") {
		t.Fatalf("expected config_invalid error, got %s", rec.Body.String())
	}
}

func TestSetDefaultAuthMethodRejectsDisabledMethod(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newAuthProvidersTestServer(t, harness)

	body := `{"method":"password"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/default", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for disabled-method default, got %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "auth_providers.default_not_enabled") {
		t.Fatalf("expected default_not_enabled error, got %s", rec.Body.String())
	}
}

func TestSetDefaultAuthMethodAcceptsEnabledMethod(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'password', true)`, tenantID); err != nil {
		t.Fatalf("seed: %v", err)
	}
	server := newAuthProvidersTestServer(t, harness)
	body := `{"method":"password"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/default", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}

	var stored *string
	if err := harness.SQL.QueryRow(`SELECT default_auth_method::text FROM tenants WHERE id = $1`, tenantID).Scan(&stored); err != nil {
		t.Fatalf("read default: %v", err)
	}
	if stored == nil || *stored != "password" {
		t.Fatalf("default_auth_method = %v, want password", stored)
	}
}

func TestDisableProviderThatIsCurrentDefaultRejected(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'password', true)`, tenantID); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE tenants SET default_auth_method = 'password'::tenant_auth_method WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set default: %v", err)
	}
	server := newAuthProvidersTestServer(t, harness)
	body := `{"enabled":false,"config":{}}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth-providers/password", strings.NewReader(body))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "auth_providers.default_disabled") {
		t.Fatalf("expected default_disabled error, got %s", rec.Body.String())
	}
}
