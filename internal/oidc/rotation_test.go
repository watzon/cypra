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

func TestForceRotateDuringOverlapPreservesSingleOverlap(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	kek := bytes.Repeat([]byte{1}, crypto.MasterKeyBytes)
	service := oidc.NewRotationService(harness.SQL, kek)
	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC() })
	if err := service.ForceRotate(context.Background(), tenantID); err != nil {
		t.Fatalf("first force rotate: %v", err)
	}
	service.SetNow(func() time.Time { return time.Unix(200, 0).UTC() })
	if err := service.ForceRotate(context.Background(), tenantID); err != nil {
		t.Fatalf("second force rotate: %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'active'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'overlap'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'retired'`, 1, tenantID)
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

func TestRotateDueRotatesActiveAndRetiresExpiredOverlap(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	retireTenantID := dbtest.SeedTenant(t, harness.SQL, "bravo")
	kek := bytes.Repeat([]byte{1}, crypto.MasterKeyBytes)
	now := time.Unix(100, 0).UTC()

	if _, err := harness.SQL.Exec(`UPDATE oidc_signing_keys SET activated_at = $1 WHERE tenant_id = $2 AND state = 'active'`, now.Add(-oidc.RotationInterval-time.Second), tenantID); err != nil {
		t.Fatalf("backdate active key: %v", err)
	}
	expired, err := crypto.GenerateSigningKey(retireTenantID, 99, crypto.SigningAlgRS256, kek, now.Add(-oidc.OverlapWindow))
	if err != nil {
		t.Fatalf("generate expired overlap key: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_signing_keys (tenant_id, kid, algorithm, public_key_jwk, private_key_encrypted, state, activated_at, retires_at) VALUES ($1, $2, $3, $4, $5, 'overlap', $6, $7)`, retireTenantID, expired.KID, expired.Algorithm, []byte(expired.PublicJWK), expired.PrivateKeyEncrypted, now.Add(-oidc.OverlapWindow), now.Add(-time.Second)); err != nil {
		t.Fatalf("insert expired overlap key: %v", err)
	}

	service := oidc.NewRotationService(harness.SQL, kek)
	service.SetNow(func() time.Time { return now })
	if err := service.RotateDue(context.Background()); err != nil {
		t.Fatalf("rotate due: %v", err)
	}

	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'active'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'overlap'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'retired'`, 1, retireTenantID)
}

func TestRotationRunStopsOnContextCancel(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	kek := bytes.Repeat([]byte{1}, crypto.MasterKeyBytes)
	service := oidc.NewRotationService(harness.SQL, kek)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- service.Run(ctx, 10*time.Millisecond)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("rotation run did not stop after cancellation")
	}
}
