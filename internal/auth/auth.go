// Package auth contains Phase 3 authorization scaffolding.
package auth

//revive:disable:exported

import (
	"context"
	"net/http"
	"strings"
)

type Actor struct {
	Kind            string
	InstanceAdmin   bool
	TenantRole      string
	PersonalTokenID string
}

type actorKey struct{}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor := Actor{}
		if r.Header.Get("X-Cypra-Instance-Admin") == "true" {
			actor.Kind = "instance_admin"
			actor.InstanceAdmin = true
		}
		if role := r.Header.Get("X-Cypra-Tenant-Role"); role != "" {
			actor.Kind = "tenant_admin"
			actor.TenantRole = role
		}
		if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
			actor.PersonalTokenID = strings.TrimPrefix(authz, "Bearer ")
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, actor)))
	})
}

func ActorFromContext(ctx context.Context) Actor {
	actor, _ := ctx.Value(actorKey{}).(Actor)
	return actor
}

func RequireInstanceAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !ActorFromContext(r.Context()).InstanceAdmin {
			writeError(w, http.StatusForbidden, "auth.forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireTenantRole(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role := ActorFromContext(r.Context()).TenantRole
		if role != "owner" && role != "admin" {
			writeError(w, http.StatusForbidden, "auth.forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `"}`))
}
