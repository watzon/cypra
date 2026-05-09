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
	"github.com/watzon/cypra/internal/storage/localdisk"
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
	pictureID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO storage_objects (id, tenant_id, backend, key, content_type, byte_size) VALUES ($1, $2, 'local-disk', 'profiles/user.png', 'image/png', 42)`, pictureID, tenantID); err != nil {
		t.Fatalf("seed picture object: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE users SET metadata = '{"name":"Test User"}'::jsonb, profile_picture_object_id = $1 WHERE id = $2`, pictureID, userID); err != nil {
		t.Fatalf("update user profile: %v", err)
	}
	kek := dbtest.TestMasterKey()
	clientSecret, err := oidc.EncryptClientSecret("secret", kek)
	if err != nil {
		t.Fatalf("encrypt client secret: %v", err)
	}
	clientUUID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, $6, 'client_secret_post')`, clientUUID, tenantID, projectID, clientSecret, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid", "email", "profile"})); err != nil {
		t.Fatalf("seed oidc client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: kek, InstallDomain: "cypra.localhost", Storage: localdisk.Store{Root: t.TempDir(), Secret: kek}}
	verifier := "correct-horse-battery-staple"
	code, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid", "email"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256", Nonce: "nonce"})
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	tokens, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier})
	if err != nil {
		t.Fatalf("token exchange: %v", err)
	}
	if tokens.AccessToken == "" || tokens.IDToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("tokens = %+v", tokens)
	}
	refreshed, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "refresh_token", RefreshToken: tokens.RefreshToken})
	if err != nil {
		t.Fatalf("refresh exchange: %v", err)
	}
	if refreshed.RefreshToken == "" || refreshed.RefreshToken == tokens.RefreshToken {
		t.Fatalf("refreshed tokens = %+v", refreshed)
	}
	if err := provider.Revoke(context.Background(), refreshed.RefreshToken); err != nil {
		t.Fatalf("revoke refresh token: %v", err)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "refresh_token", RefreshToken: tokens.RefreshToken}); !errors.Is(err, oidc.ErrInvalidGrant) {
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
	if !strings.Contains(userinfo["sub"].(string), userID.String()) || userinfo["email"] != "user@example.com" || userinfo["name"] != "Test User" {
		t.Fatalf("userinfo = %+v", userinfo)
	}
	picture, _ := userinfo["picture"].(string)
	if !strings.Contains(picture, "https://acme.cypra.localhost/storage/") {
		t.Fatalf("userinfo picture = %q", picture)
	}
	bravoID := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	if _, err := provider.UserInfo(context.Background(), "bravo", bravoID, tokens.AccessToken); err == nil {
		t.Fatal("userinfo accepted token under another tenant")
	}
}

func TestPublicClientTokenExchangeAndInvalidGrantPaths(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	projectID := seedProject(t, harness, tenantID)
	userID := seedOIDCUser(t, harness, tenantID)
	clientUUID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'public-client', $4, $5, 'none')`, clientUUID, tenantID, projectID, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid", "email"})); err != nil {
		t.Fatalf("seed public client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: dbtest.TestMasterKey(), InstallDomain: "cypra.localhost"}
	verifier := "public-client-verifier"

	code, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "public-client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256"})
	if err != nil {
		t.Fatalf("authorize public client: %v", err)
	}
	tokens, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "public-client", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier})
	if err != nil {
		t.Fatalf("public token exchange: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("tokens = %+v", tokens)
	}

	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "public-client", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier}); !errors.Is(err, oidc.ErrInvalidGrant) {
		t.Fatalf("reused authorization code error = %v", err)
	}

	redirectCode, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "public-client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256"})
	if err != nil {
		t.Fatalf("authorize redirect mismatch code: %v", err)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "public-client", GrantType: "authorization_code", Code: redirectCode, RedirectURI: "https://app.example.com/other", CodeVerifier: verifier}); !errors.Is(err, oidc.ErrInvalidGrant) {
		t.Fatalf("redirect mismatch error = %v", err)
	}

	pkceCode, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "public-client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256"})
	if err != nil {
		t.Fatalf("authorize pkce mismatch code: %v", err)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "public-client", GrantType: "authorization_code", Code: pkceCode, RedirectURI: "https://app.example.com/callback", CodeVerifier: "wrong"}); !errors.Is(err, oidc.ErrInvalidGrant) {
		t.Fatalf("pkce mismatch error = %v", err)
	}
}

func TestAuthorizeRejectsInvalidInputsAndUnsupportedGrant(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	projectID := seedProject(t, harness, tenantID)
	userID := seedOIDCUser(t, harness, tenantID)
	clientUUID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, 'none')`, clientUUID, tenantID, projectID, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: dbtest.TestMasterKey(), InstallDomain: "cypra.localhost"}
	base := oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid"}, CodeChallenge: challenge("verifier"), CodeChallengeMethod: "S256"}

	badRedirect := base
	badRedirect.RedirectURI = "https://evil.example.com/callback"
	if _, err := provider.Authorize(context.Background(), badRedirect); !errors.Is(err, oidc.ErrInvalidRedirectURI) {
		t.Fatalf("bad redirect error = %v", err)
	}

	badPKCE := base
	badPKCE.CodeChallengeMethod = "plain"
	if _, err := provider.Authorize(context.Background(), badPKCE); !errors.Is(err, oidc.ErrUnsupportedPKCEMode) {
		t.Fatalf("bad pkce error = %v", err)
	}

	badScope := base
	badScope.Scope = []string{"openid", "profile"}
	if _, err := provider.Authorize(context.Background(), badScope); !errors.Is(err, oidc.ErrInvalidScope) {
		t.Fatalf("bad scope error = %v", err)
	}

	if _, err := provider.Token(context.Background(), oidc.TokenRequest{GrantType: "client_credentials"}); !errors.Is(err, oidc.ErrUnsupportedGrant) {
		t.Fatalf("unsupported grant error = %v", err)
	}
	if err := provider.RecordConsent(context.Background(), tenantID, userID, clientUUID, []string{"openid"}); err != nil {
		t.Fatalf("record consent: %v", err)
	}
	if err := provider.RecordConsent(context.Background(), tenantID, userID, clientUUID, []string{"openid", "email"}); err != nil {
		t.Fatalf("update consent: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_consents WHERE tenant_id = $1 AND user_id = $2 AND oidc_client_uuid = $3`, 1, tenantID, userID, clientUUID)
}

func TestClientSecretValidationRejectsMissingAndWrongSecrets(t *testing.T) {
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
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'confidential', $4, $5, $6, 'client_secret_post')`, clientUUID, tenantID, projectID, clientSecret, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed confidential client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: kek, InstallDomain: "cypra.localhost"}
	verifier := "confidential-verifier"
	code, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: tenantID, UserID: userID, ClientID: "confidential", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256"})
	if err != nil {
		t.Fatalf("authorize confidential client: %v", err)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "confidential", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier}); !errors.Is(err, oidc.ErrInvalidClient) {
		t.Fatalf("missing secret error = %v", err)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "confidential", ClientSecret: "wrong", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier}); !errors.Is(err, oidc.ErrInvalidClient) {
		t.Fatalf("wrong secret error = %v", err)
	}
}

func TestTokenExchangeRejectsClientFromAnotherTenant(t *testing.T) {
	harness := dbtest.New(t)
	acmeID := dbtest.SeedTenant(t, harness.SQL, "acme")
	bravoID := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	bravoProjectID := seedProject(t, harness, bravoID)
	bravoUserID := seedOIDCUser(t, harness, bravoID)
	bravoClientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'bravo-client', $4, $5, 'none')`, bravoClientID, bravoID, bravoProjectID, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed bravo client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: dbtest.TestMasterKey(), InstallDomain: "cypra.localhost"}
	verifier := "bravo-verifier"
	code, err := provider.Authorize(context.Background(), oidc.AuthorizeRequest{TenantID: bravoID, UserID: bravoUserID, ClientID: "bravo-client", RedirectURI: "https://app.example.com/callback", Scope: []string{"openid"}, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256"})
	if err != nil {
		t.Fatalf("authorize bravo client: %v", err)
	}
	if _, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: acmeID, TenantSlug: "acme", ClientID: "bravo-client", GrantType: "authorization_code", Code: code, RedirectURI: "https://app.example.com/callback", CodeVerifier: verifier}); !errors.Is(err, oidc.ErrInvalidClient) {
		t.Fatalf("cross-tenant token exchange error = %v", err)
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
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_refresh_tokens (family_id, token_hash, tenant_id, oidc_client_uuid, user_id, scope, expires_at, consumed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, uuid.New(), []byte("expired-refresh"), tenantID, clientID, userID, pq.Array([]string{"openid"}), time.Now().Add(-time.Hour), nil); err != nil {
		t.Fatalf("seed expired refresh token: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_refresh_tokens (family_id, token_hash, tenant_id, oidc_client_uuid, user_id, scope, expires_at, consumed_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, uuid.New(), []byte("consumed-refresh"), tenantID, clientID, userID, pq.Array([]string{"openid"}), time.Now().Add(time.Hour), time.Now().Add(-8*24*time.Hour)); err != nil {
		t.Fatalf("seed consumed refresh token: %v", err)
	}
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, $4)`, sessionID, userID, tenantID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("seed expired session: %v", err)
	}
	if err := oidc.CleanupExpired(context.Background(), harness.SQL, time.Now()); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_authorization_codes`, 0)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_refresh_tokens`, 0)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND revoked_at IS NOT NULL`, 1, sessionID)
}

func TestRunCleanupTickerReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := oidc.RunCleanupTicker(ctx, nil, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("cleanup ticker error = %v", err)
	}
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

func TestConsentCoversScopes(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	projectID := seedProject(t, harness, tenantID)
	userID := seedOIDCUser(t, harness, tenantID)
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, 'none')`, clientID, tenantID, projectID, pq.Array([]string{"https://app.example.com/callback"}), pq.Array([]string{"openid", "email", "profile"})); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	provider := oidc.Provider{DB: harness.SQL, KEK: dbtest.TestMasterKey()}
	covered, prompted, err := provider.ConsentCovers(context.Background(), tenantID, userID, clientID, []string{"openid"})
	if err != nil || covered || prompted {
		t.Fatalf("pre-consent covered=%v prompted=%v err=%v", covered, prompted, err)
	}
	if err := provider.RecordConsent(context.Background(), tenantID, userID, clientID, []string{"openid", "email"}); err != nil {
		t.Fatalf("record consent: %v", err)
	}
	covered, prompted, err = provider.ConsentCovers(context.Background(), tenantID, userID, clientID, []string{"email", "openid"})
	if err != nil || !covered {
		t.Fatalf("covered=%v prompted=%v err=%v", covered, prompted, err)
	}
	covered, prompted, err = provider.ConsentCovers(context.Background(), tenantID, userID, clientID, []string{"openid", "profile"})
	if err != nil || covered {
		t.Fatalf("scope upgrade covered=%v prompted=%v err=%v", covered, prompted, err)
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
