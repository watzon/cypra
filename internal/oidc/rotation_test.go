package oidc_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/oidc"
)

func TestForceRotateMovesActiveToOverlapAndCreatesActive(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	kek := bytes.Repeat([]byte{1}, crypto.MasterKeyBytes)

	service := oidc.NewRotationService(harness.SQL, kek)
	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC() })
	if err := service.ForceRotate(context.Background(), tenantID); err != nil {
		t.Fatalf("force rotate: %v", err)
	}

	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'active'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'overlap'`, 1, tenantID)
}

func TestSunsetKeysAndPrune(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	kek := bytes.Repeat([]byte{1}, crypto.MasterKeyBytes)

	service := oidc.NewRotationService(harness.SQL, kek)
	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC() })
	if err := service.SunsetKeys(context.Background(), tenantID); err != nil {
		t.Fatalf("sunset keys: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'sunsetting'`, 1, tenantID)

	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC().Add(oidc.SunsetWindow + time.Second) })
	if err := service.PruneSunsetKeys(context.Background()); err != nil {
		t.Fatalf("prune keys: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1`, 0, tenantID)
}
