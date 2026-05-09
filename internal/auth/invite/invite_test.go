package invite_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/dbtest"
)

func TestTenantInviteIssueRotateRedeem(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	service := invite.Service{DB: harness.SQL}
	ctx := context.Background()

	token1, inviteID1, err := service.Issue(ctx, invite.IssueRequest{TenantID: &tenantID, Email: "user@example.com", Role: "admin", CreatedByKind: "tenant_admin"})
	if err != nil {
		t.Fatalf("issue invite: %v", err)
	}
	token2, inviteID2, err := service.Issue(ctx, invite.IssueRequest{TenantID: &tenantID, Email: "user@example.com", Role: "admin", CreatedByKind: "tenant_admin"})
	if err != nil {
		t.Fatalf("reissue invite: %v", err)
	}
	if token1 == token2 || inviteID1 != inviteID2 {
		t.Fatalf("invite rotation token/id mismatch token1=%q token2=%q ids=%s/%s", token1, token2, inviteID1, inviteID2)
	}
	if _, err := service.Redeem(ctx, token1, "User"); !errors.Is(err, invite.ErrInvalidToken) {
		t.Fatalf("old token error = %v", err)
	}
	redeemed, err := service.Redeem(ctx, token2, "User")
	if err != nil {
		t.Fatalf("redeem invite: %v", err)
	}
	if redeemed.TenantID == nil || *redeemed.TenantID != tenantID || redeemed.UserID == uuid.Nil {
		t.Fatalf("redeemed = %#v", redeemed)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM tenant_memberships WHERE tenant_id = $1 AND role = 'admin'`, 1, tenantID)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM email_outbox WHERE template = 'admin-invite'`, 2)
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM audit_entries WHERE action = 'invite.redeem'`, 1)
}

func TestInstanceInviteRedeemsAdmin(t *testing.T) {
	harness := dbtest.New(t)
	service := invite.Service{DB: harness.SQL}
	token, _, err := service.Issue(context.Background(), invite.IssueRequest{Email: "admin@example.com", Role: "instance_admin", CreatedByKind: "instance_admin"})
	if err != nil {
		t.Fatalf("issue instance invite: %v", err)
	}
	redeemed, err := service.Redeem(context.Background(), token, "Admin")
	if err != nil {
		t.Fatalf("redeem instance invite: %v", err)
	}
	if redeemed.AdminID == uuid.Nil || redeemed.TenantID != nil {
		t.Fatalf("redeemed = %#v", redeemed)
	}
	dbtest.RequireCount(t, harness.SQL, `SELECT count(*) FROM instance_admins WHERE email = 'admin@example.com'`, 1)
}
