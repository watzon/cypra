package fuzz_test

import (
	"context"
	"strings"
	"testing"

	fuzzharness "github.com/watzon/cypra/internal/db/fuzz"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestTenantIsolationFuzzerAllowsTenantScopedHandler(t *testing.T) {
	dbHarness := dbtest.New(t)
	protectedTenant := dbtest.SeedTenant(t, dbHarness.SQL, "acme")
	otherTenant := dbtest.SeedSecondTenant(t, dbHarness.SQL, "bravo")
	if _, err := dbHarness.SQL.Exec(`INSERT INTO projects (tenant_id, slug, name) VALUES ($1, 'app', 'App')`, protectedTenant); err != nil {
		t.Fatalf("seed protected project: %v", err)
	}

	fuzzer := fuzzharness.Harness{ProtectedTenant: protectedTenant, OtherTenant: otherTenant}
	err := fuzzer.AssertIsolated(context.Background(), func(ctx context.Context) fuzzharness.Outcome {
		var rows int
		if err := dbHarness.TenantDB.RawScan(ctx, `SELECT count(*) FROM projects`, otherTenant, &rows); err != nil {
			return fuzzharness.Outcome{Err: err}
		}
		return fuzzharness.Outcome{Rows: rows}
	})
	if err != nil {
		t.Fatalf("tenant-scoped handler rejected: %v", err)
	}
}

func TestTenantIsolationFuzzerCatchesCrossTenantRows(t *testing.T) {
	dbHarness := dbtest.New(t)
	protectedTenant := dbtest.SeedTenant(t, dbHarness.SQL, "acme")
	otherTenant := dbtest.SeedSecondTenant(t, dbHarness.SQL, "bravo")
	fuzzer := fuzzharness.Harness{ProtectedTenant: protectedTenant, OtherTenant: otherTenant}

	err := fuzzer.AssertIsolated(context.Background(), func(context.Context) fuzzharness.Outcome {
		return fuzzharness.Outcome{Rows: 1}
	})
	if err == nil || !strings.Contains(err.Error(), "tenant-isolation violation") {
		t.Fatalf("fuzzer error = %v, want tenant-isolation violation", err)
	}
}
