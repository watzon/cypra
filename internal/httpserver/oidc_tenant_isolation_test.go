package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
)

func TestPhase5OIDCRoutesUseResolvedTenant(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	dbtest.SeedSecondTenant(t, harness.SQL, "bravo")

	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/.well-known/openid-configuration", nil)
	resp := httptest.NewRecorder()
	newTestServer(t, harness).Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("discovery status = %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "https://acme.cypra.localhost") || strings.Contains(body, "https://bravo.cypra.localhost") {
		t.Fatalf("discovery route tenant isolation failed: %s", body)
	}
}
