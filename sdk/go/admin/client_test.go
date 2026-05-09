package admin_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	cypra "github.com/watzon/cypra/sdk/go"
	"github.com/watzon/cypra/sdk/go/admin"
)

func TestCreateTenantRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenants/" || r.Header.Get("X-Cypra-Instance-Admin") != "true" || r.Header.Get("Authorization") != "" {
			t.Fatalf("unexpected request path/header: %s %s %s", r.URL.Path, r.Header.Get("X-Cypra-Instance-Admin"), r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"tenant-id","slug":"acme","name":"Acme"}`))
	}))
	defer server.Close()
	client, err := admin.New(server.URL, "pat")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	tenant, err := client.CreateTenant(context.Background(), "acme", "Acme")
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if tenant.Slug != "acme" {
		t.Fatalf("tenant = %#v", tenant)
	}
}

func TestInstanceAdminHelpersDoNotSendTenantPAT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-Cypra-Instance-Admin") != "true" {
			t.Fatalf("unexpected instance admin headers: auth=%q instance=%q", r.Header.Get("Authorization"), r.Header.Get("X-Cypra-Instance-Admin"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	client, err := admin.New(server.URL, "tenant-pat")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if _, err := client.ListInstanceAdmins(context.Background()); err != nil {
		t.Fatalf("list instance admins: %v", err)
	}
}

func TestTypedErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"tenant.slug_conflict"}`))
	}))
	defer server.Close()
	client, err := admin.New(server.URL, "pat")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = client.CreateTenant(context.Background(), "acme", "Acme")
	if !errors.Is(err, cypra.ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
	var apiErr *cypra.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "tenant.slug_conflict" || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("expected api error metadata, got %#v", err)
	}
}

func TestTenantScopedHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/projects/" || r.Header.Get("X-Cypra-Tenant-Role") != "admin" {
			t.Fatalf("unexpected tenant request: %s %s", r.URL.Path, r.Header.Get("X-Cypra-Tenant-Role"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"project-id","slug":"console","name":"Console"}`))
	}))
	defer server.Close()
	client, err := admin.New(server.URL, "pat")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	project, err := client.CreateProject(context.Background(), "console", "Console")
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.Slug != "console" {
		t.Fatalf("project = %#v", project)
	}
}

func TestSupportedCRUDRoutes(t *testing.T) {
	seen := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		seen[key]++
		w.Header().Set("Content-Type", "application/json")
		switch key {
		case "GET /api/v1/tenants/":
			_, _ = w.Write([]byte(`[{"id":"tenant-id","slug":"acme","name":"Acme"}]`))
		case "GET /api/v1/tenants/tenant-id":
			_, _ = w.Write([]byte(`{"id":"tenant-id","slug":"acme","name":"Acme"}`))
		case "PUT /api/v1/tenants/tenant-id":
			_, _ = w.Write([]byte(`{"id":"tenant-id","slug":"acme","name":"Renamed"}`))
		case "DELETE /api/v1/tenants/tenant-id":
			w.WriteHeader(http.StatusNoContent)
		case "GET /api/v1/projects/":
			_, _ = w.Write([]byte(`[{"id":"project-id","slug":"console","name":"Console"}]`))
		case "GET /api/v1/users/":
			_, _ = w.Write([]byte(`[{"id":"user-id","email":"user@example.com"}]`))
		case "POST /api/v1/users/":
			_, _ = w.Write([]byte(`{"id":"user-id","email":"new@example.com"}`))
		case "GET /api/v1/users/user-id/export":
			_, _ = w.Write([]byte(`{"id":"user-id"}`))
		case "DELETE /api/v1/users/user-id":
			w.WriteHeader(http.StatusNoContent)
		case "GET /api/v1/audit/export":
			_, _ = w.Write([]byte(`{"id":"audit-id","actor_kind":"system","action":"test","resource_kind":"test"}` + "\n"))
		case "GET /api/v1/instance/admins":
			_, _ = w.Write([]byte(`[{"id":"admin-id","email":"admin@example.com"}]`))
		case "POST /api/v1/instance/invite":
			_, _ = w.Write([]byte(`{"id":"invite-id","token":"invite-token"}`))
		case "DELETE /api/v1/instance/admins/admin-id":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected SDK route %s", key)
		}
	}))
	defer server.Close()
	client, err := admin.New(server.URL, "pat")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()
	if _, err := client.ListTenants(ctx); err != nil {
		t.Fatalf("list tenants: %v", err)
	}
	if _, err := client.GetTenant(ctx, "tenant-id"); err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if _, err := client.UpdateTenant(ctx, "tenant-id", "Renamed"); err != nil {
		t.Fatalf("update tenant: %v", err)
	}
	if err := client.DeleteTenant(ctx, "tenant-id"); err != nil {
		t.Fatalf("delete tenant: %v", err)
	}
	if _, err := client.ListProjects(ctx); err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if _, err := client.ListUsers(ctx); err != nil {
		t.Fatalf("list users: %v", err)
	}
	if _, err := client.CreateUser(ctx, "new@example.com"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := client.ExportUser(ctx, "user-id"); err != nil {
		t.Fatalf("export user: %v", err)
	}
	if err := client.DeleteUser(ctx, "user-id"); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	if _, err := client.ListAuditEntries(ctx); err != nil {
		t.Fatalf("list audit entries: %v", err)
	}
	if _, err := client.ListInstanceAdmins(ctx); err != nil {
		t.Fatalf("list instance admins: %v", err)
	}
	if _, err := client.InviteInstanceAdmin(ctx, "admin2@example.com", ""); err != nil {
		t.Fatalf("invite instance admin: %v", err)
	}
	if err := client.DemoteInstanceAdmin(ctx, "admin-id"); err != nil {
		t.Fatalf("demote instance admin: %v", err)
	}
}
