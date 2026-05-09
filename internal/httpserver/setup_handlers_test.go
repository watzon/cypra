package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/watzon/cypra/internal/bootstrap"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestSetupVerifyAndPasskeyBegin(t *testing.T) {
	harness := dbtest.New(t)
	token, err := bootstrap.NewService(harness.SQL, nil).MintSetupToken(context.Background())
	if err != nil {
		t.Fatalf("mint setup token: %v", err)
	}
	router := newTestServer(t, harness).Router()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/setup/verify", strings.NewReader(`{"token":"`+token+`"}`))
	req.Host = "cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("verify response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/setup/passkey/begin", strings.NewReader(`{"token":"`+token+`","email":"root@example.com","display_name":"Root"}`))
	req.Host = "cypra.localhost"
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "ceremony_id") {
		t.Fatalf("passkey begin response = %d %s", resp.Code, resp.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM instance_admin_webauthn_challenges`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM instance_admins`, 0)
}
