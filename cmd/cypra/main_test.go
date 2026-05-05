package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/pat"
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
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admins (email, display_name, metadata) VALUES ('admin@example.com', 'Admin', '{}'::jsonb)`); err != nil {
		t.Fatalf("seed admin: %v", err)
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
	if err := json.Unmarshal(content, &exported); err != nil || exported["format"] != "cypra-export-v1" {
		t.Fatalf("exported=%+v err=%v", exported, err)
	}
	if err := run([]string{"import", "--passphrase-file", passphrase, backup}); !isExitCode(err, 8) {
		t.Fatalf("import non-empty error = %v", err)
	}
	if err := run([]string{"admin", "invite", "recovery@example.com"}); err != nil {
		t.Fatalf("admin invite: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM pending_invitations WHERE tenant_id IS NULL AND email = 'recovery@example.com'`, 1)
	token, _, err := (invite.Service{DB: harness.SQL}).Issue(context.Background(), invite.IssueRequest{Email: "second@example.com", Role: "instance_admin", CreatedByKind: "system"})
	if err != nil {
		t.Fatalf("issue recovery invite: %v", err)
	}
	if _, err := (invite.Service{DB: harness.SQL}).Redeem(context.Background(), token, "Second Admin"); err != nil {
		t.Fatalf("redeem recovery invite: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM instance_admins`, 2)
	_ = tenantID
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

func isExitCode(err error, code int) bool {
	var exitErr exitError
	return errors.As(err, &exitErr) && exitErr.code == code
}

var _ *sql.DB
