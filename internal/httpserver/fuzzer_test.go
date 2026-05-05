package httpserver_test

import (
	"context"
	"testing"

	"github.com/watzon/cypra/internal/db"
	fuzzharness "github.com/watzon/cypra/internal/db/fuzz"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestPhase3TenantScopedHandlersEnrollWithFuzzer(t *testing.T) {
	harness := dbtest.New(t)
	protectedTenant := dbtest.SeedTenant(t, harness.SQL, "acme")
	otherTenant := dbtest.SeedSecondTenant(t, harness.SQL, "bravo")
	if _, err := harness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'app', 'App')`, protectedTenant); err != nil {
		t.Fatalf("seed protected project: %v", err)
	}
	fuzzer := fuzzharness.Harness{ProtectedTenant: protectedTenant, OtherTenant: otherTenant}
	if err := fuzzer.AssertIsolated(context.Background(), func(ctx context.Context) fuzzharness.Outcome {
		tenantID, ok := db.TenantFromContext(ctx)
		if !ok {
			return fuzzharness.Outcome{Err: db.ErrTenantContextMissing}
		}
		var rows int
		if err := harness.SQL.QueryRowContext(ctx, `SELECT count(*) FROM projects WHERE tenant_id = $1`, tenantID).Scan(&rows); err != nil {
			return fuzzharness.Outcome{Err: err}
		}
		return fuzzharness.Outcome{Rows: rows}
	}); err != nil {
		t.Fatalf("phase 3 fuzzer enrollment failed: %v", err)
	}
}
