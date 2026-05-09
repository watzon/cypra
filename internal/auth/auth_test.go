package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

	called = false
	instanceHandler := RequireTenantRole(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	resp = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{Kind: "instance_admin", InstanceAdmin: true}))
	instanceHandler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent || !called {
		t.Fatalf("instance admin code = %d called = %v", resp.Code, called)
	}
}

func TestRequireTenantRoleOrPATScope(t *testing.T) {
	called := false
	handler := RequireTenantRoleOrPATScope("projects.write")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{Kind: "pat", Scopes: []string{"projects.write"}}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent || !called {
		t.Fatalf("allowed pat scope code=%d called=%v", resp.Code, called)
	}

	called = false
	req = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{Kind: "pat", Scopes: []string{"projects.read"}}))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden || called {
		t.Fatalf("forbidden pat scope code=%d called=%v", resp.Code, called)
	}

	called = false
	req = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{TenantRole: "admin"}))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent || !called {
		t.Fatalf("tenant admin code=%d called=%v", resp.Code, called)
	}

	called = false
	req = httptest.NewRequest(http.MethodPost, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{Kind: "instance_admin", InstanceAdmin: true}))
	resp = httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent || !called {
		t.Fatalf("instance admin code=%d called=%v", resp.Code, called)
	}
}

func TestRequireTenantRoleOrPATScopeWildcard(t *testing.T) {
	called := false
	handler := RequireTenantRoleOrPATScope("audit.export")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(context.WithValue(context.Background(), actorKey{}, Actor{Kind: "pat", Scopes: []string{"*"}}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent || !called {
		t.Fatalf("wildcard scope code=%d called=%v", resp.Code, called)
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
	handler := MiddlewareWithPAT(harness.SQL, true)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if actor.Kind != "pat" || actor.TenantID != tenantID || actor.UserID != userID || len(actor.Scopes) != 1 || actor.Scopes[0] != "tenants:read" {
		t.Fatalf("actor = %#v", actor)
	}
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	req.Header.Set("Authorization", "Bearer "+created.Plaintext)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if actor.Kind != "pat" || actor.InstanceAdmin {
		t.Fatalf("pat actor inherited instance-admin mode: %#v", actor)
	}

	denied := httptest.NewRecorder()
	bad := httptest.NewRequest(http.MethodGet, "/", nil)
	bad.Header.Set("Authorization", "Bearer invalid")
	handler.ServeHTTP(denied, bad)
	if denied.Code != http.StatusUnauthorized || denied.Body.String() != `{"error":"auth.pat_invalid"}` {
		t.Fatalf("denied code = %d body = %s", denied.Code, denied.Body.String())
	}
}

func TestMiddlewareWithPATResolvesInstanceAdminSessionCookie(t *testing.T) {
	harness := dbtest.New(t)
	adminID := uuid.New()
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admins (id, email, display_name, metadata) VALUES ($1, 'root@example.com', 'Root', '{}'::jsonb)`, adminID); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO instance_admin_sessions (id, instance_admin_id, expires_at) VALUES ($1, $2, $3)`, sessionID, adminID, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("seed admin session: %v", err)
	}

	var actor Actor
	handler := MiddlewareWithPAT(harness.SQL, true)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "cypra_session", Value: sessionID.String()})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if actor.Kind != "instance_admin" || !actor.InstanceAdmin || actor.InstanceAdminID != adminID {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestMiddlewareWithPATResolvesTenantSessionCookie(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	userID := uuid.New()
	sessionID := uuid.New()
	if _, err := harness.SQL.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, 'tenant@example.com', '{}'::jsonb)`, userID, tenantID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO tenant_memberships (tenant_id, user_id, role) VALUES ($1, $2, 'admin')`, tenantID, userID); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at) VALUES ($1, $2, 'user', $3, $4)`, sessionID, userID, tenantID, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("seed user session: %v", err)
	}

	var actor Actor
	handler := MiddlewareWithPAT(harness.SQL, true)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "cypra_session", Value: sessionID.String()})
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if actor.Kind != "tenant_admin" || actor.TenantID != tenantID || actor.UserID != userID || actor.TenantRole != "admin" {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestMiddlewareWithPATIgnoresInvalidSessionCookies(t *testing.T) {
	harness := dbtest.New(t)
	var actor Actor
	handler := MiddlewareWithPAT(harness.SQL, true)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		actor = ActorFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "cypra_session", Value: "not-a-uuid"})
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if actor.Kind != "" {
		t.Fatalf("actor = %#v, want empty", actor)
	}
}
