package oidc_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/oidc"
)

func TestRedirectURIMatchingIsStrict(t *testing.T) {
	registered := []string{"https://app.example.com:8443/callback?x=1"}
	if !oidc.RedirectURIMatches(registered, "https://APP.example.com:8443/callback?x=1") {
		t.Fatal("host case folding should match")
	}
	for _, uri := range []string{
		"https://app.example.com:8443/Callback?x=1",
		"https://app.example.com:8443/callback/?x=1",
		"https://app.example.com/callback?x=1",
		"http://app.example.com:8443/callback?x=1",
		"https://app.example.com:8443/callback?x=1#fragment",
	} {
		if oidc.RedirectURIMatches(registered, uri) {
			t.Fatalf("uri unexpectedly matched: %s", uri)
		}
	}
}

func TestAuthorizationCodeTokenRefreshAndReuse(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	projectID := seedProject(t, harness, tenantID)
	userID := seedOIDCUser(t, harness, tenantID)
	kek := dbtest.TestMasterKey()
	clientSecret, err := oidc.EncryptClientSecret("secret", kek)
	if err != nil {
		t.Fatalf("encrypt client secret: %v", err)
	}
	clientUUID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, $6, 'client_secret_post')`, clientUUID, tenantID, projectID, clientSecret, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid", "email", "profile"})); err != nil {
		t.Fatalf("seed oidc client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: kek, InstallDomain: "cypra.localhost"}
	verifier := "correct-horse-battery-staple"
	code, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid", "email"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256", Nonce: "nonce"})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	tokens, err := provider.Token(context.Background(), oidc.TokenRequest{TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier})
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	if tokens.AccessToken == "" || tokens.IDToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("tokens = %+v", tokens)
	}
	refreshed, err := provider.Token(context.Background(), oidc.TokenRequest{TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "refresh_token", RefreshToken: tokens.RefreshToken})
	if err != nil {
		t.Fatalf("refresh exchange: %v", err)
	}
	if refreshed.RefreshToken == "" || refreshed.RefreshToken == tokens.RefreshToken {
		t.Fatalf("refreshed tokens = %+v", refreshed)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "refresh_token", RefreshToken: tokens.RefreshToken}); !errors.Is(err, oidc.ErrInvalidGrant) {
		t.Fatalf("reuse error = %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_refresh_tokens WHERE family_id = (SELECT family_id FROM oidc_refresh_tokens WHERE token_hash IS NOT NULL LIMIT 1) AND revoked_at IS NOT NULL`, 2)
	if err := provider.Revoke(context.Background(), refreshed.AccessToken); err != nil {
		t.Fatalf("revoke access no-op: %v", err)
	}
	if err := provider.Revoke(context.Background(), "unknown"); err != oidc.ErrUnsupportedToken {
		t.Fatalf("unknown revoke error = %v", err)
	}
	userinfo, err := provider.UserInfo(context.Background(), "acme", tenantID, tokens.AccessToken)
	if err != nil {
		t.Fatalf("userinfo: %v", err)
	}
	if !strings.Contains(userinfo["sub"].(string), userID.String()) || userinfo["email"] != "user@example.com" {
		t.Fatalf("userinfo = %+v", userinfo)
	}
}

func TestCleanupExpiredAuthorizationCodes(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	projectID := seedProject(t, harness, tenantID)
	userID := seedOIDCUser(t, harness, tenantID)
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, 'none')`, clientID, tenantID, projectID, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_authorization_codes (code_hash, tenant_id, oidc_client_uuid, user_id, redirect_uri, scope, pkce_challenge, pkce_method, expires_at, consumed_at) VALUES ('hash'::bytea, $1, $2, $3, 'https://app.example.com/callback', $4, 'challenge', 'S256', $5, $6)`, tenantID, clientID, userID, pq.Array([]string{"openid"}), time.Now().Add(-8*24*time.Hour), time.Now().Add(-8*24*time.Hour)); err != nil {
		t.Fatalf("seed auth code: %v", err)
	}
	if err := oidc.CleanupExpired(context.Background(), harness.SQL, time.Now()); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_authorization_codes`, 0)
}

func TestSigningKeysIncludeSunsettingWindow(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	kek := dbtest.TestMasterKey()
	service := oidc.NewRotationService(harness.SQL, kek)
	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC() })
	if err := service.SunsetKeys(context.Background(), tenantID); err != nil {
		t.Fatalf("sunset: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: kek, Now: func() time.Time { return time.Unix(100, 0).UTC() }}
	keys, err := provider.SigningKeys(context.Background(), tenantID)
	if err != nil || len(keys) != 1 {
		t.Fatalf("sunsetting keys len=%d err=%v", len(keys), err)
	}
	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC().Add(oidc.SunsetWindow + time.Second) })
	if err := service.PruneSunsetKeys(context.Background()); err != nil {
		t.Fatalf("prune: %v", err)
	}
	keys, err = provider.SigningKeys(context.Background(), tenantID)
	if err != nil || len(keys) != 0 {
		t.Fatalf("pruned keys len=%d err=%v", len(keys), err)
	}
}

func challenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func seedProject(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	projectID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return projectID
}

func seedOIDCUser(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, email_verified_at, metadata) VALUES ($1, $2, 'user@example.com', now(), '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return userID
}

func TestRedirectURLCaseFoldKeepsPathCase(t *testing.T) {
	registered := []string{"https://Example.com/callback"}
	requested, _ := url.Parse("https://example.com/callback")
	if requested.Hostname() != "example.com" || !oidc.RedirectURIMatches(registered, requested.String()) {
		t.Fatal("expected host case-folded match")
	}
	if oidc.RedirectURIMatches(registered, "https://example.com/Callback") {
		t.Fatal("path case should be significant")
	}
}
