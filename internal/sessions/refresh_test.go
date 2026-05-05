package sessions_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/sessions"
)

func TestRefreshTokenReuseRevokesFamilyAndAudits(t *testing.T) {
	harness := dbtest.New(t)
	tenantID, clientID, userID := seedRefreshGraph(t, harness)
	service := sessions.NewRefreshService(harness.SQL)
	metricCount := 0
	service.OnReuseDetected(func() { metricCount++ })
	root, err := service.Mint(context.Background(), tenantID, clientID, userID, []string{"openid"}, nil, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("mint root refresh: %v", err)
	}
	if _, err := service.Consume(context.Background(), root.Plaintext, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("consume root refresh: %v", err)
	}
	if _, err := service.Consume(context.Background(), root.Plaintext, time.Now().Add(time.Hour)); !errors.Is(err, sessions.ErrRefreshReuseDetected) {
		t.Fatalf("reuse error = %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_refresh_tokens WHERE family_id = $1 AND revoked_at IS NOT NULL`, 2, root.FamilyID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE action = 'cypra_oidc_refresh_reuse_detected'`, 1)
	if metricCount != 1 {
		t.Fatalf("metric count = %d, want 1", metricCount)
	}
}

func TestRefreshTokenRaceAllowsOneConsumer(t *testing.T) {
	harness := dbtest.New(t)
	tenantID, clientID, userID := seedRefreshGraph(t, harness)
	service := sessions.NewRefreshService(harness.SQL)
	root, err := service.Mint(context.Background(), tenantID, clientID, userID, []string{"openid"}, nil, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("mint root refresh: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Consume(context.Background(), root.Plaintext, time.Now().Add(time.Hour))
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	successes := 0
	reuses := 0
	for err := range errs {
		if err == nil {
			successes++
		}
		if errors.Is(err, sessions.ErrRefreshReuseDetected) {
			reuses++
		}
	}
	if successes != 1 || reuses != 1 {
		t.Fatalf("successes=%d reuses=%d, want 1/1", successes, reuses)
	}
}

func TestRefreshTokenFamilyProperty(t *testing.T) {
	harness := dbtest.New(t)
	tenantID, clientID, userID := seedRefreshGraph(t, harness)
	service := sessions.NewRefreshService(harness.SQL)
	current, err := service.Mint(context.Background(), tenantID, clientID, userID, []string{"openid", "email"}, nil, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("mint root refresh: %v", err)
	}
	reused := current.Plaintext
	for range 5 {
		current, err = service.Consume(context.Background(), current.Plaintext, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatalf("consume chain refresh: %v", err)
		}
	}
	if _, err := service.Consume(context.Background(), reused, time.Now().Add(time.Hour)); !errors.Is(err, sessions.ErrRefreshReuseDetected) {
		t.Fatalf("reuse error = %v", err)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM oidc_refresh_tokens WHERE family_id = $1 AND revoked_at IS NULL`, 0, current.FamilyID)
}

func TestRefreshMintWithParentUsesExistingFamily(t *testing.T) {
	harness := dbtest.New(t)
	tenantID, clientID, userID := seedRefreshGraph(t, harness)
	service := sessions.NewRefreshService(harness.SQL)
	now := time.Unix(100, 0).UTC()
	service.SetNow(func() time.Time { return now })

	root, err := service.Mint(context.Background(), tenantID, clientID, userID, []string{"openid"}, nil, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("mint root refresh: %v", err)
	}
	child, err := service.Mint(context.Background(), tenantID, clientID, userID, []string{"openid"}, &root.ID, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("mint child refresh: %v", err)
	}
	if child.FamilyID != root.FamilyID || child.ParentID == nil || *child.ParentID != root.ID {
		t.Fatalf("child = %#v root = %#v", child, root)
	}
}

func TestRefreshConsumeUnknownTokenReportsReuse(t *testing.T) {
	harness := dbtest.New(t)
	service := sessions.NewRefreshService(harness.SQL)
	service.SetNow(func() time.Time { return time.Unix(100, 0).UTC() })

	if _, err := service.Consume(context.Background(), "missing", time.Unix(200, 0).UTC()); !errors.Is(err, sessions.ErrRefreshReuseDetected) {
		t.Fatalf("unknown token error = %v, want reuse detected", err)
	}
}

func seedRefreshGraph(t *testing.T, harness *dbtest.Harness) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	projectID := uuid.New()
	clientID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'user@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES ($1, $2, 'app', 'App')`, projectID, tenantID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO oidc_clients (id, tenant_id, project_id, client_id, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES ($1, $2, $3, 'client', ARRAY['https://app.example/callback'], ARRAY['openid'], 'none')`, clientID, tenantID, projectID); err != nil {
		t.Fatalf("seed oidc client: %v", err)
	}
	return tenantID, clientID, userID
}
