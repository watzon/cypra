package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
)

func TestSigningKeyHandlersListAndForceRotate(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	var activeKID string
	if err := harness.SQL.QueryRow(`SELECT kid FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'active'`, tenantID).Scan(&activeKID); err != nil {
		t.Fatalf("load active kid: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/signing-keys/", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), activeKID) || !strings.Contains(resp.Body.String(), `"state":"active"`) {
		t.Fatalf("list signing keys = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/signing-keys/rotate", strings.NewReader(`{"kid":"wrong"}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("rotate wrong kid = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/signing-keys/rotate", strings.NewReader(`{"kid":"`+activeKID+`"}`))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"state":"overlap"`) {
		t.Fatalf("rotate signing key = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'active'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'overlap'`, 1, tenantID)
}
