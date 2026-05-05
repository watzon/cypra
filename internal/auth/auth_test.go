package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/pat"
)

func TestMiddlewareSetsInstanceAdminActor(t *testing.T) {
	adminID := uuid.New()
	var actor Actor
	handler := Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	req.Header.Set("X-Cypra-Instance-Admin-Id", adminID.String())

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if actor.Kind != "instance_admin" || !actor.InstanceAdmin || actor.InstanceAdminID != adminID {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestMiddlewareSetsTenantActor(t *testing.T) {
	userID := uuid.New()
	var actor Actor
	handler := Middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Cypra-Tenant-Role", "admin")
	req.Header.Set("X-Cypra-User-Id", userID.String())

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if actor.Kind != "tenant_admin" || actor.TenantRole != "admin" || actor.UserID != userID {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestRequireInstanceAdmin(t *testing.T) {
	called := false
	handler := RequireInstanceAdmin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/", nil))
	if denied.Code != http.StatusForbidden || called {
		t.Fatalf("denied code = %d called = %v", denied.Code, called)
	}

	allowed := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{InstanceAdmin: true}))
	handler.ServeHTTP(allowed, req)
	if allowed.Code != http.StatusNoContent || !called {
		t.Fatalf("allowed code = %d called = %v", allowed.Code, called)
	}
}

func TestRequireTenantRole(t *testing.T) {
	for _, role := range []string{"owner", "admin"} {
		t.Run(role, func(t *testing.T) {
			called := false
			handler := RequireTenantRole(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{TenantRole: role}))
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			if resp.Code != http.StatusNoContent || !called {
				t.Fatalf("code = %d called = %v", resp.Code, called)
			}
		})
	}

	called := false
	handler := RequireTenantRole(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.Code != http.StatusForbidden || called {
		t.Fatalf("code = %d called = %v", resp.Code, called)
	}
}

func TestMiddlewareWithPATAuthenticatesAndRejectsBearer(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'pat@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	created, err := (pat.Service{DB: harness.SQL}).Create(context.Background(), tenantID, userID, "automation", []string{"tenants:read"})
	if err != nil {
		t.Fatalf("create pat: %v", err)
	}

	var actor Actor
	handler := MiddlewareWithPAT(harness.SQL)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if actor.Kind != "pat" || actor.TenantID != tenantID || actor.UserID != userID || len(actor.Scopes) != 1 || actor.Scopes[0] != "tenants:read" {
		t.Fatalf("actor = %#v", actor)
	}

	denied := httptest.NewRecorder()
	bad := httptest.NewRequest(http.MethodGet, "/", nil)
	bad.Header.Set("Authorization", "Bearer invalid")
	handler.ServeHTTP(denied, bad)
	if denied.Code != http.StatusUnauthorized || denied.Body.String() != `{"error":"auth.pat_invalid"}` {
		t.Fatalf("denied code = %d body = %s", denied.Code, denied.Body.String())
	}
}
