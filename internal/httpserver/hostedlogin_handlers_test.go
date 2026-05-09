package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/auth/magiclink"
	passwordauth "github.com/watzon/cypra/internal/auth/password"
	"github.com/watzon/cypra/internal/auth/totp"
	"github.com/watzon/cypra/internal/auth/upstream/google"
	"github.com/watzon/cypra/internal/authpolicy"
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

func TestHostedLoginShowsEmptyStateWhenNoAuthMethodsEnabled(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/login", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "no sign-in methods enabled") {
		t.Fatalf("expected empty-state copy in %s", body)
	}
	for _, banned := range []string{"Continue with passkey", "Use a password", "Send a sign-in link", "Create account"} {
		if strings.Contains(body, banned) {
			t.Fatalf("did not expect %q to render when no auth methods are enabled: %s", banned, body)
		}
	}
}

func TestHostedLoginPasswordPrimaryInlinesPasswordField(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'password', true), ($1, 'magic_link', true)`, tenantID); err != nil {
		t.Fatalf("seed methods: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE tenants SET default_auth_method = 'password'::tenant_auth_method WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set default: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/login", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `name="password"`) {
		t.Fatalf("expected password input rendered inline, got %s", body)
	}
	if !strings.Contains(body, `>Sign in</button>`) {
		t.Fatalf("expected primary 'Sign in' button, got %s", body)
	}
	if !strings.Contains(body, "Send a sign-in link") {
		t.Fatalf("expected magic-link as secondary, got %s", body)
	}
}

func TestHostedLoginPrimaryFallsBackWhenConfiguredDefaultDisabled(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	// magic_link disabled, passkey enabled. Default points at the disabled method.
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'passkey', true), ($1, 'magic_link', false)`, tenantID); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE tenants SET default_auth_method = 'magic_link'::tenant_auth_method WHERE id = $1`, tenantID); err != nil {
		t.Fatalf("set default: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/login", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, "Continue with passkey") {
		t.Fatalf("expected passkey to lead when configured default is disabled, got %s", body)
	}
	if strings.Contains(body, "Send a sign-in link") {
		t.Fatalf("disabled magic_link should not render any button, got %s", body)
	}
}

func TestHostedLoginGatesButtonsByEnabledAuthMethods(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'passkey', true), ($1, 'magic_link', true)`, tenantID); err != nil {
		t.Fatalf("seed auth methods: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/login", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Continue with passkey", "Send a sign-in link", "Create account"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in %s", want, body)
		}
	}
	if strings.Contains(body, "Use a password") {
		t.Fatalf("did not expect password block when password is disabled: %s", body)
	}
}

func TestHostedLoginRendersGoogleWhenConfigured(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	clientID, err := server.encryptProviderSecret([]byte("google-client"))
	if err != nil {
		t.Fatalf("encrypt client id: %v", err)
	}
	clientSecret, err := server.encryptProviderSecret([]byte("google-secret"))
	if err != nil {
		t.Fatalf("encrypt client secret: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO upstream_providers (tenant_id, kind, client_id_encrypted, client_secret_encrypted, enabled) VALUES ($1, 'google', $2, $3, true)`, tenantID, clientID, clientSecret); err != nil {
		t.Fatalf("seed google provider: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled) VALUES ($1, 'google', true)`, tenantID); err != nil {
		t.Fatalf("seed google auth method: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/login?continue=abc", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Sign in with Google") || !strings.Contains(body, "accounts.google.com") || !strings.Contains(body, "client_id=google-client") {
		t.Fatalf("missing google link in %s", body)
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
	if !strings.Contains(rec.Body.String(), html.EscapeString(uniformAuthError)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHostedMagicLinkRequiresEmailProvider(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/login/magic-link", strings.NewReader("email=user@example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusPreconditionRequired {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "tenant.email_provider_required") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHostedPasswordResetIssuesAndConsumesResetToken(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO email_provider_configs (tenant_id, kind, from_address, from_name, config_encrypted, enabled) VALUES ($1, 'terminal', 'noreply@example.com', 'Cypra', '{}'::bytea, true)`, tenantID); err != nil {
		t.Fatalf("seed email provider: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/reset", strings.NewReader("email=user@example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "check your email") {
		t.Fatalf("issue reset = %d %s", rec.Code, rec.Body.String())
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM email_outbox WHERE template = 'password-reset'`, 1)
	resetToken, err := (passwordauth.Service{DB: harness.SQL}).IssueResetToken(contextBackground(), tenantID, userID, time.Minute)
	if err != nil {
		t.Fatalf("issue direct reset token: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/reset", strings.NewReader("token="+url.QueryEscape(resetToken)+"&password=new-correct-password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Password reset") {
		t.Fatalf("consume reset = %d %s", rec.Code, rec.Body.String())
	}
	if err := (passwordauth.Service{DB: harness.SQL}).Verify(contextBackground(), userID, "new-correct-password"); err != nil {
		t.Fatalf("verify new password: %v", err)
	}
}

func TestHostedLoginRateLimitRendersCountdown(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/login/password", strings.NewReader("email=limited@example.com&password=bad"))
		req.RemoteAddr = "203.0.113.10:1234"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		server.Router().ServeHTTP(rec, req)
		if i < 5 && rec.Code == http.StatusTooManyRequests {
			t.Fatalf("rate limited too early on attempt %d: %s", i, rec.Body.String())
		}
		if i == 5 {
			if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), "Try again in 1 minute") {
				t.Fatalf("rate limit response = %d %s", rec.Code, rec.Body.String())
			}
		}
	}
}

func TestHostedFactorVerifiesTOTPAndBackupCode(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	totpService := totp.Service{DB: harness.SQL, KEK: dbtest.TestMasterKey()}
	secret, _, err := totpService.Enroll(contextBackground(), tenantID, userID, "Cypra", "user@example.com")
	if err != nil {
		t.Fatalf("enroll totp: %v", err)
	}
	codes, err := (backupcodes.Service{DB: harness.SQL}).RegenerateForUser(contextBackground(), tenantID, userID)
	if err != nil {
		t.Fatalf("regenerate backup codes: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	login := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/2fa", nil)
	if _, err := server.establishUserSession(login, req, tenantID, userID); err != nil {
		t.Fatalf("establish session: %v", err)
	}
	cookie := sessionCookie(t, login)
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("decode totp secret: %v", err)
	}
	code := totp.GenerateCode(secretBytes, time.Now().UTC())
	req = httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/2fa", strings.NewReader("factor=totp&code="+url.QueryEscape(code)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Factor accepted") {
		t.Fatalf("totp factor = %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/2fa", strings.NewReader("factor=backup&code="+url.QueryEscape(codes[0])))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Factor accepted") {
		t.Fatalf("backup factor = %d %s", rec.Code, rec.Body.String())
	}
}

func TestHostedPasswordSigninCreatesSessionCookieAndRow(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := (passwordauth.Service{DB: harness.SQL}).SetPassword(contextBackground(), tenantID, userID, "correct-password"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/login/password", strings.NewReader("email=user@example.com&password=correct-password"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookie(t, rec)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND tenant_id = $2 AND subject_id = $3 AND revoked_at IS NULL`, 1, cookie.Value, tenantID, userID)
}

func TestAuthMagicLinkVerifyCreatesSession(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	token, err := (magiclink.Service{DB: harness.SQL}).Issue(contextBackground(), tenantID, &userID, "user@example.com", "", authpolicy.MagicLinkPolicy{TTLMinutes: 1})
	if err != nil {
		t.Fatalf("issue magic link: %v", err)
	}
	body, _ := json.Marshal(map[string]string{"token": token})
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/magic-link/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	newHostedLoginTestServer(t, harness).Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookie(t, rec)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND tenant_id = $2 AND subject_id = $3`, 1, cookie.Value, tenantID, userID)
}

func TestInviteRedeemCreatesTenantSession(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	token, _, err := (invite.Service{DB: harness.SQL}).Issue(contextBackground(), invite.IssueRequest{TenantID: &tenantID, Email: "invitee@example.com", Role: "member", CreatedByKind: "tenant_admin"})
	if err != nil {
		t.Fatalf("issue invite: %v", err)
	}
	body, _ := json.Marshal(map[string]string{"token": token, "display_name": "Invitee"})
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/invite/redeem", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	newHostedLoginTestServer(t, harness).Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookie(t, rec)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND tenant_id = $2`, 1, cookie.Value, tenantID)
}

func TestHostedInviteContinuationRedeemsWithBoundCookie(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	token, inviteID, err := (invite.Service{DB: harness.SQL}).Issue(context.Background(), invite.IssueRequest{TenantID: &tenantID, Email: "invitee@example.com", Role: "member", CreatedByKind: "tenant_admin"})
	if err != nil {
		t.Fatalf("issue invite: %v", err)
	}
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://acme.cypra.localhost/invite?token="+token, nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Accept invite") {
		t.Fatalf("invite page status = %d body = %s", rec.Code, rec.Body.String())
	}
	continuationCookie := inviteContinuationCookie(t, rec)

	body := strings.NewReader("continuation_id=" + continuationCookie.Value + "&display_name=Invitee&password=optional-password")
	req = httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/invite/redeem", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(continuationCookie)
	rec = httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("redeem status = %d body = %s", rec.Code, rec.Body.String())
	}
	session := sessionCookie(t, rec)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM pending_invitations WHERE id = $1 AND redeemed_at IS NOT NULL`, 1, inviteID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM invite_continuations WHERE id = $1 AND consumed_at IS NOT NULL`, 1, continuationCookie.Value)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND tenant_id = $2`, 1, session.Value, tenantID)
}

func TestGoogleCallbackCreatesUserSession(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	seedGoogleProvider(t, harness, tenantID, true)
	state, err := (google.Client{Secret: []byte("dev-google-oauth-state-secret")}).SignState(google.State{TenantID: tenantID, ReturnURL: "/app", Nonce: "nonce", ExpiresAt: time.Now().Add(time.Minute).Unix()})
	if err != nil {
		t.Fatalf("sign state: %v", err)
	}
	idToken := "header." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"google-user","email":"google@example.com","name":"Google User","nonce":"nonce"}`)) + ".signature"
	body, _ := json.Marshal(map[string]string{"state": state, "id_token": idToken})
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/google/callback", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	newHostedLoginTestServer(t, harness).Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookie(t, rec)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM users WHERE tenant_id = $1 AND email = 'google@example.com'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE id = $1 AND tenant_id = $2`, 1, cookie.Value, tenantID)
}

func TestGoogleStartRequiresEnabledTenantProvider(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	seedGoogleProvider(t, harness, tenantID, false)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/google/start", bytes.NewReader([]byte(`{"return_url":"/app"}`)))
	rec := httptest.NewRecorder()
	newHostedLoginTestServer(t, harness).Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusPreconditionRequired || !strings.Contains(rec.Body.String(), "tenant.google_provider_required") {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestHostedLoginRendersInstanceAdminFormOnInstallHost(t *testing.T) {
	harness := dbtest.New(t)
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://cypra.localhost/login", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "tenant.not_found") {
		t.Fatalf("response leaked JSON tenant.not_found body: %s", body)
	}
	if !strings.Contains(body, "instance-admin-passkey-button") {
		t.Fatalf("expected instance-admin login form, got: %s", body)
	}
}

func TestHostedLoginRedirectsOnBareInstallHostForNonLogin(t *testing.T) {
	harness := dbtest.New(t)
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodGet, "https://cypra.localhost/signup", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("Location = %q want /", loc)
	}
}

func TestAuthLogoutClearsCookies(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/logout", nil)
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected at least one Set-Cookie header to clear session")
	}
	for _, c := range cookies {
		if c.MaxAge >= 0 {
			t.Fatalf("cookie %q has MaxAge=%d, want negative for clearing", c.Name, c.MaxAge)
		}
	}
}

func TestAuthRevokeOtherSessionsKeepsCallerSession(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	server := newHostedLoginTestServer(t, harness)
	subject := uuid.New()
	current := uuid.New()
	other := uuid.New()
	for _, id := range []uuid.UUID{current, other} {
		if _, err := harness.SQL.Exec(
			`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, now() + interval '1 hour')`,
			id, subject, tenantID,
		); err != nil {
			t.Fatalf("seed session: %v", err)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "https://acme.cypra.localhost/api/v1/auth/sessions/revoke-others", nil)
	req.AddCookie(&http.Cookie{Name: "cypra_session", Value: current.String()})
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var revokedCurrent, revokedOther sql.NullTime
	if err := harness.SQL.QueryRow(`SELECT revoked_at FROM sessions WHERE id = $1`, current).Scan(&revokedCurrent); err != nil {
		t.Fatalf("read current: %v", err)
	}
	if err := harness.SQL.QueryRow(`SELECT revoked_at FROM sessions WHERE id = $1`, other).Scan(&revokedOther); err != nil {
		t.Fatalf("read other: %v", err)
	}
	if revokedCurrent.Valid {
		t.Fatalf("caller session was revoked: %v", revokedCurrent.Time)
	}
	if !revokedOther.Valid {
		t.Fatalf("other session was not revoked")
	}
}

func newHostedLoginTestServer(t *testing.T, harness *dbtest.Harness) *Server {
	t.Helper()
	server, err := New(Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", DevOpenAPI: true, KEKLoaded: true, MasterKey: dbtest.TestMasterKey(), TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return server
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "cypra_session" && cookie.Value != "" && cookie.MaxAge > 0 {
			return cookie
		}
	}
	t.Fatalf("response did not set cypra_session: %v", rec.Result().Cookies())
	return nil
}

func inviteContinuationCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "cypra_invite_continuation" && cookie.Value != "" && cookie.MaxAge > 0 {
			return cookie
		}
	}
	t.Fatalf("response did not set cypra_invite_continuation: %v", rec.Result().Cookies())
	return nil
}

func seedGoogleProvider(t *testing.T, harness *dbtest.Harness, tenantID uuid.UUID, enabled bool) {
	t.Helper()
	if _, err := harness.SQL.Exec(`INSERT INTO upstream_providers (tenant_id, kind, client_id_encrypted, client_secret_encrypted, enabled) VALUES ($1, 'google', 'client'::bytea, 'secret'::bytea, $2)`, tenantID, enabled); err != nil {
		t.Fatalf("seed google provider: %v", err)
	}
}

func contextBackground() context.Context { return context.Background() }
