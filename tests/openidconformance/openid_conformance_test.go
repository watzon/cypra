package openidconformance_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/oidc"
)

func TestSeededTenantOIDCConformanceSmoke(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "conformance")
	projectID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, email_verified_at, metadata) VALUES ($1, $2, 'user@example.com', now(), '{"name":"Conformance User"}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	secret, err := oidc.EncryptClientSecret("secret", dbtest.TestMasterKey())
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	clientUUID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'conformance-client', $4, $5, $6, 'client_secret_post')`, clientUUID, tenantID, projectID, secret, pq.Array([]string{"https://client.example/callback"}), pq.Array([]string{"openid", "email", "profile"})); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if err := (oidc.Provider{DB: harness.SQL}).RecordConsent(context.Background(), tenantID, userID, clientUUID, []string{"openid", "email", "profile"}); err != nil {
		t.Fatalf("seed consent: %v", err)
	}
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, $4)`, sessionID, userID, tenantID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, MasterKey: dbtest.TestMasterKey()})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	router := server.Router()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	req.Host = "conformance.cypra.localhost"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "authorization_endpoint") {
		t.Fatalf("discovery = %d %s", resp.Code, resp.Body.String())
	}

	verifier := "conformance-verifier"
	values := url.Values{}
	values.Set("client_id", "conformance-client")
	values.Set("redirect_uri", "https://client.example/callback")
	values.Set("scope", "openid email profile")
	values.Set("state", "state")
	values.Set("nonce", "nonce")
	values.Set("code_challenge", challenge(verifier))
	values.Set("code_challenge_method", "S256")
	req = httptest.NewRequest(http.MethodGet, "/oidc/authorize?"+values.Encode(), nil)
	req.Host = "conformance.cypra.localhost"
	req.AddCookie(&http.Cookie{Name: "cypra_session", Value: sessionID.String()})
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusFound {
		t.Fatalf("authorize = %d %s", resp.Code, resp.Body.String())
	}
	location, err := url.Parse(resp.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse authorize redirect: %v", err)
	}
	code := location.Query().Get("code")
	if code == "" || location.Query().Get("state") != "state" {
		t.Fatalf("authorize redirect = %s", location.String())
	}

	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {"conformance-client"}, "client_secret": {"secret"}, "code": {code}, "redirect_uri": {"https://client.example/callback"}, "code_verifier": {verifier}}
	req = httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader(form.Encode()))
	req.Host = "conformance.cypra.localhost"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("token = %d %s", resp.Code, resp.Body.String())
	}
	var tokens map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" || tokens["id_token"] == "" || tokens["refresh_token"] == "" {
		t.Fatalf("token response = %+v", tokens)
	}

	req = httptest.NewRequest(http.MethodGet, "/oidc/userinfo", nil)
	req.Host = "conformance.cypra.localhost"
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "Conformance User") {
		t.Fatalf("userinfo = %d %s", resp.Code, resp.Body.String())
	}
}

func challenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
