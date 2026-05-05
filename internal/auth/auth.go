// Package auth contains Phase 3 authorization scaffolding.
package auth

//revive:disable:exported

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/pat"
)

type Actor struct {
	Kind            string
	InstanceAdmin   bool
	TenantRole      string
	PersonalTokenID string
	TenantID        uuid.UUID
	UserID          uuid.UUID
	Scopes          []string
}

type actorKey struct{}

func Middleware(next http.Handler) http.Handler {
	return MiddlewareWithPAT(nil)(next)
}

func MiddlewareWithPAT(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
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
				plaintext := strings.TrimPrefix(authz, "Bearer ")
				actor.PersonalTokenID = plaintext
				if db != nil {
					token, err := (pat.Service{DB: db}).Authenticate(r.Context(), plaintext)
					if err != nil {
						writeError(w, http.StatusUnauthorized, "auth.pat_invalid")
						return
					}
					actor.Kind = "pat"
					actor.TenantRole = "admin"
					actor.TenantID = token.TenantID
					actor.UserID = token.UserID
					actor.Scopes = token.Scopes
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, actor)))
		})
	}
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
