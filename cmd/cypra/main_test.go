package main

import (
	"context"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/auth/totp"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/oidc"
	"github.com/watzon/cypra/internal/pat"
	"github.com/watzon/cypra/internal/sessions"
)

func TestExportRequiresPassphrase(t *testing.T) {
	err := run([]string{"export", "--out", "backup.json"})
	var exitErr exitError
	if !errors.As(err, &exitErr) || exitErr.code != 7 {
		t.Fatalf("error = %v", err)
	}
}

func TestAdminRequiresCommand(t *testing.T) {
	err := run([]string{"admin"})
	var exitErr exitError
	if !errors.As(err, &exitErr) || exitErr.code != 2 {
		t.Fatalf("error = %v", err)
	}
}

func TestVersionJSON(t *testing.T) {
	if err := run([]string{"version", "--json"}); err != nil {
		t.Fatalf("version json: %v", err)
	}
}

func TestExportImportAndAdminInviteRecovery(t *testing.T) {
	harness := dbtest.New(t)
	t.Setenv("DATABASE_URL", harness.DSN)
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(dbtest.TestMasterKey()))
	sourceStorage := filepath.Join(t.TempDir(), "source-storage")
	restoreStorage := filepath.Join(t.TempDir(), "restore-storage")
	t.Setenv("STORAGE_LOCAL_PATH", sourceStorage)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admins (email, display_name, metadata) VALUES ('admin@example.com', 'Admin', '{}'::jsonb)`); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	projectID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	clientSecret, err := oidc.EncryptClientSecret("secret", dbtest.TestMasterKey())
	if err != nil {
		t.Fatalf("encrypt client secret: %v", err)
	}
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', $4, $5, $6, 'client_secret_post')`, clientID, tenantID, projectID, clientSecret, pq.Array([]string{"https://app.example/callback"}), pq.Array([]string{"openid"})); err != nil {
		t.Fatalf("seed oidc client: %v", err)
	}
	refresh, err := sessions.NewRefreshService(harness.SQL).Mint(context.Background(), tenantID, clientID, userID, []string{"openid"}, nil, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("mint refresh token: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id, credential, purpose) VALUES ($1, $2, $3, $4, 'acme.cypra.localhost', '{}'::jsonb, 'primary')`, tenantID, userID, []byte("credential-id"), []byte("public-key")); err != nil {
		t.Fatalf("seed passkey credential: %v", err)
	}
	totpSecret, _, err := (totp.Service{DB: harness.SQL, KEK: dbtest.TestMasterKey()}).Enroll(context.Background(), tenantID, userID, "Cypra", "user@example.com")
	if err != nil {
		t.Fatalf("seed totp: %v", err)
	}
	storageObjectID := uuid.New()
	storageKey := "profiles/user.txt"
	if _, err := harness.SQL.Exec(`INSERT INTO storage_objects (id, tenant_id, backend, key, content_type, byte_size) VALUES ($1, $2, 'local-disk', $3, 'text/plain', 13)`, storageObjectID, tenantID, storageKey); err != nil {
		t.Fatalf("seed storage object: %v", err)
	}
	if _, err := harness.SQL.Exec(`UPDATE users SET profile_picture_object_id = $1 WHERE id = $2`, storageObjectID, userID); err != nil {
		t.Fatalf("attach storage object: %v", err)
	}
	sourceStoragePath, err := storagePath(sourceStorage, storageKey)
	if err != nil {
		t.Fatalf("storage path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(sourceStoragePath), 0o700); err != nil {
		t.Fatalf("make storage dir: %v", err)
	}
	if err := os.WriteFile(sourceStoragePath, []byte("profile-bytes"), 0o600); err != nil {
		t.Fatalf("write storage object: %v", err)
	}
	backup := filepath.Join(t.TempDir(), "backup.json")
	passphrase := filepath.Join(t.TempDir(), "pass.txt")
	if err := os.WriteFile(passphrase, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write passphrase: %v", err)
	}
	if err := run([]string{"export", "--out", backup, "--passphrase-file", passphrase}); err != nil {
		t.Fatalf("export: %v", err)
	}
	content, err := os.ReadFile(backup) // #nosec G304 -- test-controlled temp path.
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	var exported map[string]any
	if err := json.Unmarshal(content, &exported); err != nil || exported["format"] != backupFormatVersion || exported["ciphertext"] == "" {
		t.Fatalf("exported=%+v err=%v", exported, err)
	}
	if err := run([]string{"import", "--passphrase-file", passphrase, backup}); !isExitCode(err, 8) {
		t.Fatalf("import non-empty error = %v", err)
	}
	restoreHarness := dbtest.New(t)
	t.Setenv("DATABASE_URL", restoreHarness.DSN)
	t.Setenv("STORAGE_LOCAL_PATH", restoreStorage)
	if err := run([]string{"import", "--passphrase-file", passphrase, backup}); err != nil {
		t.Fatalf("import restore: %v", err)
	}
	dbtest.RequireCount(t, restoreHarness.SQL, `SELECT count(*) FROM tenants WHERE id = $1`, 1, tenantID)
	dbtest.RequireCount(t, restoreHarness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1`, 1, tenantID)
	dbtest.RequireCount(t, restoreHarness.SQL, `SELECT count(*) FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2`, 1, tenantID, userID)
	dbtest.RequireCount(t, restoreHarness.SQL, `SELECT count(*) FROM totp_credentials WHERE tenant_id = $1 AND user_id = $2`, 1, tenantID, userID)
	dbtest.RequireCount(t, restoreHarness.SQL, `SELECT count(*) FROM storage_objects WHERE id = $1 AND tenant_id = $2`, 1, storageObjectID, tenantID)
	restoredStoragePath, err := storagePath(restoreStorage, storageKey)
	if err != nil {
		t.Fatalf("restored storage path: %v", err)
	}
	restoredBytes, err := os.ReadFile(restoredStoragePath) // #nosec G304 -- test-controlled temp path.
	if err != nil || string(restoredBytes) != "profile-bytes" {
		t.Fatalf("restored storage bytes = %q err=%v", restoredBytes, err)
	}
	rawTOTPSecret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(totpSecret)
	if err != nil {
		t.Fatalf("decode totp secret: %v", err)
	}
	now := time.Now().UTC()
	if ok, err := (totp.Service{DB: restoreHarness.SQL, KEK: dbtest.TestMasterKey()}).Verify(context.Background(), userID, totp.GenerateCode(rawTOTPSecret, now), now); err != nil || !ok {
		t.Fatalf("totp verify after import ok=%v err=%v", ok, err)
	}
	provider := oidc.Provider{DB: restoreHarness.SQL, KEK: dbtest.TestMasterKey(), InstallDomain: "cypra.localhost"}
	refreshed, err := provider.Token(context.Background(), oidc.TokenRequest{TenantID: tenantID, TenantSlug: "acme", ClientID: "client", ClientSecret: "secret", GrantType: "refresh_token", RefreshToken: refresh.Plaintext})
	if err != nil {
		t.Fatalf("refresh grant after import: %v", err)
	}
	if refreshed.RefreshToken == "" || refreshed.RefreshToken == refresh.Plaintext {
		t.Fatalf("refresh grant did not rotate token: %+v", refreshed)
	}
	dbtest.RequireCount(t, restoreHarness.SQL, `SELECT count(*) FROM oidc_refresh_tokens WHERE family_id = $1`, 2, refresh.FamilyID)
	t.Setenv("DATABASE_URL", harness.DSN)
	output, err := runCaptureStdout([]string{"admin", "invite", "recovery@example.com"})
	if err != nil {
		t.Fatalf("admin invite: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM pending_invitations WHERE tenant_id IS NULL AND email = 'recovery@example.com'`, 1)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE actor_kind = 'instance_admin' AND action = 'instance_admin.recovery' AND metadata->>'cross_tenant' = 'true'`, 1)
	token := tokenFromCLIOutput(t, output)
	if _, err := (invite.Service{DB: harness.SQL}).Redeem(context.Background(), token, "Recovery Admin"); err != nil {
		t.Fatalf("redeem recovery invite: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM instance_admins`, 2)
	_ = tenantID
}

func runCaptureStdout(args []string) (string, error) {
	original := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = write
	runErr := run(args)
	_ = write.Close()
	os.Stdout = original
	content, readErr := io.ReadAll(read)
	_ = read.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(content), runErr
}

func tokenFromCLIOutput(t *testing.T, output string) string {
	t.Helper()
	for _, field := range strings.Fields(output) {
		if token, ok := strings.CutPrefix(field, "token="); ok {
			return token
		}
	}
	t.Fatalf("no token in CLI output %q", output)
	return ""
}

func TestPATRoleDowngradeRevokesTokens(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	service := pat.Service{DB: harness.SQL}
	created, err := service.Create(context.Background(), tenantID, userID, "dev", []string{"admin"})
	if err != nil {
		t.Fatalf("create pat: %v", err)
	}
	if err := service.RevokeForUser(context.Background(), tenantID, userID, "role_downgrade"); err != nil {
		t.Fatalf("revoke for user: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), created.Plaintext); !errors.Is(err, pat.ErrInvalidToken) {
		t.Fatalf("auth revoked token = %v", err)
	}
}

func TestAdminSubcommandsExerciseHappyPathsAndErrors(t *testing.T) {
	harness := dbtest.New(t)
	t.Setenv("DATABASE_URL", harness.DSN)
	t.Setenv("MASTER_KEY", base64.StdEncoding.EncodeToString(dbtest.TestMasterKey()))
	tenantID := dbtest.SeedTenant(t, harness.SQL, "cli")
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admins (email, display_name, metadata) VALUES ('admin@example.com', 'Admin', '{}'::jsonb)`); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO passkey_credentials (tenant_id, user_id, credential_id, public_key, rp_id, credential, purpose) VALUES ($1, $2, 'cli-passkey', 'public-key', 'cli.cypra.localhost', '{}'::jsonb, 'primary')`, tenantID, userID); err != nil {
		t.Fatalf("seed passkey: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO password_credentials (tenant_id, user_id, argon2id_hash) VALUES ($1, $2, 'hash')`, tenantID, userID); err != nil {
		t.Fatalf("seed password: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, 'user', $2, now() + interval '1 hour')`, userID, tenantID); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	if err := run([]string{"admin", "promote", "--tenant=cli", "--email=invitee@example.com"}); err != nil {
		t.Fatalf("admin promote: %v", err)
	}
	if _, err := runCaptureStdout([]string{"admin", "list-instance-admins", "--json"}); err != nil {
		t.Fatalf("list instance admins: %v", err)
	}
	if err := run([]string{"admin", "reset-passkey", "--tenant=cli", "--email=user@example.com"}); err != nil {
		t.Fatalf("reset passkey: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM passkey_credentials WHERE tenant_id = $1`, 0, tenantID)
	if err := run([]string{"admin", "reset-passwords", "--tenant=cli"}); err != nil {
		t.Fatalf("reset passwords: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM password_credentials WHERE tenant_id = $1 AND must_reset = true`, 1, tenantID)
	if err := run([]string{"admin", "rotate-key", "--tenant=cli"}); err != nil {
		t.Fatalf("rotate key: %v", err)
	}
	if err := run([]string{"admin", "revoke-tenant-tokens", "--tenant=cli"}); err != nil {
		t.Fatalf("revoke tenant tokens: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM sessions WHERE tenant_id = $1 AND revoked_at IS NOT NULL`, 1, tenantID)
	if _, err := runCaptureStdout([]string{"admin", "list-tenants", "--json"}); err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if err := run([]string{"admin", "promote", "--email=missing-tenant@example.com"}); !isExitCode(err, 2) {
		t.Fatalf("promote missing tenant error = %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE actor_kind = 'instance_admin' AND metadata->>'cross_tenant' = 'true'`, 7)
}

func TestAdminResetBootstrapHappyPath(t *testing.T) {
	harness := dbtest.New(t)
	t.Setenv("DATABASE_URL", harness.DSN)
	if _, err := runCaptureStdout([]string{"admin", "reset-bootstrap"}); err != nil {
		t.Fatalf("reset bootstrap: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM bootstrap_tokens WHERE consumed_at IS NULL AND revoked_at IS NULL`, 1)
}

func isExitCode(err error, code int) bool {
	var exitErr exitError
	return errors.As(err, &exitErr) && exitErr.code == code
}

var _ *sql.DB
