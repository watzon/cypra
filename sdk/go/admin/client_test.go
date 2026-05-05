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
		if r.URL.Path != "/api/v1/tenants/" || r.Header.Get("X-Cypra-Instance-Admin") != "true" || r.Header.Get("Authorization") != "Bearer pat" {
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
