package backupcodes_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestRegenerateImmediatelyInvalidatesUserCodes(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	service := backupcodes.Service{DB: harness.SQL}
	oldCodes, err := service.RegenerateForUser(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("generate old codes: %v", err)
	}
	newCodes, err := service.RegenerateForUser(context.Background(), tenantID, userID)
	if err != nil {
		t.Fatalf("generate new codes: %v", err)
	}
	if len(newCodes) != backupcodes.CodeCount {
		t.Fatalf("new code count = %d", len(newCodes))
	}
	ok, err := service.ConsumeForUser(context.Background(), userID, oldCodes[0])
	if err != nil || ok {
		t.Fatalf("old code consume ok=%v err=%v", ok, err)
	}
	ok, err = service.ConsumeForUser(context.Background(), userID, newCodes[0])
	if err != nil || !ok {
		t.Fatalf("new code consume ok=%v err=%v", ok, err)
	}
	ok, err = service.ConsumeForUser(context.Background(), userID, newCodes[0])
	if err != nil || ok {
		t.Fatalf("reused code consume ok=%v err=%v", ok, err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM user_backup_codes WHERE user_id = $1 AND used_at IS NOT NULL`, backupcodes.CodeCount+1, userID)
}

func TestInstanceAdminBackupCodes(t *testing.T) {
	harness := dbtest.New(t)
	adminID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admins (id, email, display_name, metadata) VALUES ($1, 'admin@example.com', 'Admin', '{}'::jsonb)`, adminID); err != nil {
		t.Fatalf("seed instance admin: %v", err)
	}
	service := backupcodes.Service{DB: harness.SQL}
	codes, err := service.RegenerateForInstanceAdmin(context.Background(), adminID)
	if err != nil {
		t.Fatalf("generate admin codes: %v", err)
	}
	ok, err := service.ConsumeForInstanceAdmin(context.Background(), adminID, codes[0])
	if err != nil || !ok {
		t.Fatalf("consume admin code ok=%v err=%v", ok, err)
	}
}
