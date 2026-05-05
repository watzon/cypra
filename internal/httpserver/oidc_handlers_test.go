package httpserver_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/oidc"
)

func TestOIDCDiscoveryJWKSAndAuthCodeFlow(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	projectID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, email_verified_at, metadata) VALUES ($1, $2, 'user@example.com', now(), '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	secret, err := oidc.EncryptClientSecret("secret", dbtest.TestMasterKey())
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, 'client', $3, $4, $5, 'client_secret_post')`, tenantID, projectID, secret, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid", "email"})); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, MasterKey: dbtest.TestMasterKey()})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	router := server.Router()

	req := httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil)
	req.Host = "acme.cypra.localhost"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Header().Get("Cache-Control") != "public, max-age=600, must-revalidate" {
		t.Fatalf("discovery = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/.well-known/jwks.json", nil)
	req.Host = "acme.cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Header().Get("Cache-Control") != "public, max-age=300, must-revalidate" || !bytes.Contains(resp.Body.Bytes(), []byte(tenantID.String()+":1")) {
		t.Fatalf("jwks = %d %s", resp.Code, resp.Body.String())
	}

	verifier := "correct-horse-battery-staple"
	values := url.Values{}
	values.Set("client_id", "client")
	values.Set("redirect_uri", "https://app.example.com/callback")
	values.Set("scope", "openid email")
	values.Set("state", "state")
	values.Set("nonce", "nonce")
	values.Set("code_challenge", oidcChallenge(verifier))
	values.Set("code_challenge_method", "S256")
	values.Set("user_id", userID.String())
	req = httptest.NewRequest(http.MethodGet, "/oidc/authorize?"+values.Encode(), nil)
	req.Host = "acme.cypra.localhost"
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusFound {
		t.Fatalf("authorize = %d %s", resp.Code, resp.Body.String())
	}
	location, err := url.Parse(resp.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	code := location.Query().Get("code")
	if code == "" || location.Query().Get("state") != "state" {
		t.Fatalf("redirect location = %s", location.String())
	}

	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {"client"}, "client_secret": {"secret"}, "code": {code}, "redirect_uri": {"https://app.example.com/callback"}, "code_verifier": {verifier}}
	req = httptest.NewRequest(http.MethodPost, "/oidc/token", strings.NewReader(form.Encode()))
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("token = %d %s", resp.Code, resp.Body.String())
	}
	var tokens map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	accessToken, _ := tokens["access_token"].(string)
	if accessToken == "" || tokens["id_token"] == "" || tokens["refresh_token"] == "" {
		t.Fatalf("tokens = %+v", tokens)
	}

	req = httptest.NewRequest(http.MethodGet, "/oidc/userinfo", nil)
	req.Host = "acme.cypra.localhost"
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("userinfo = %d %s", resp.Code, resp.Body.String())
	}
}

func oidcChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
