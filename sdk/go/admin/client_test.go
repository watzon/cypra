package admin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/watzon/cypra/sdk/go/admin"
)

func TestCreateTenantRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tenants/" || r.Header.Get("X-Cypra-Instance-Admin") != "true" {
			t.Fatalf("unexpected request path/header: %s %s", r.URL.Path, r.Header.Get("X-Cypra-Instance-Admin"))
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
