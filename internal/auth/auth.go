// Package auth contains Phase 3 authorization scaffolding.
package auth

//revive:disable:exported

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/logging"
	"github.com/watzon/cypra/internal/pat"
)

type Actor struct {
	Kind            string
	InstanceAdmin   bool
	InstanceAdminID uuid.UUID
	TenantRole      string
	PersonalTokenID string
	TenantID        uuid.UUID
	UserID          uuid.UUID
	Scopes          []string
}

type actorKey struct{}

// Middleware resolves an Actor from cookie session and (when permitted) test
// headers. It honors X-Cypra-Instance-Admin / X-Cypra-Tenant-Role; only use in
// tests or behind a trusted proxy.
func Middleware(next http.Handler) http.Handler {
	return MiddlewareWithPAT(nil, true)(next)
}

// MiddlewareWithPAT resolves the request actor.
//
// trustDevHeaders gates whether X-Cypra-Instance-Admin / X-Cypra-Tenant-Role /
// X-Cypra-User-Id can elevate the actor. These headers exist for test fixtures
// and trusted reverse proxies; in normal production deployments they must be
// rejected so an attacker can't escalate by setting a header.
func MiddlewareWithPAT(db *sql.DB, trustDevHeaders bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := Actor{}
			// Cookie session is the production source of truth. Resolve it first so
			// test-mode headers below cannot mask a real instance-admin or user
			// session that's already authenticated via the cypra_session cookie.
			if db != nil {
				actor = actorFromSessionCookie(r, db)
			}
			// Headers may elevate the actor (test fixtures, upstream proxies) but
			// never downgrade: a cookie-resolved instance admin stays an instance
			// admin even if a stray X-Cypra-Tenant-Role lands on the request. Only
			// honored when trustDevHeaders is set.
			if trustDevHeaders && !actor.InstanceAdmin && r.Header.Get("X-Cypra-Instance-Admin") == "true" {
				actor.Kind = "instance_admin"
				actor.InstanceAdmin = true
				if id, err := uuid.Parse(r.Header.Get("X-Cypra-Instance-Admin-Id")); err == nil {
					actor.InstanceAdminID = id
					logging.SetActorID(r.Context(), id.String())
				}
			}
			if trustDevHeaders {
				if role := r.Header.Get("X-Cypra-Tenant-Role"); role != "" && !actor.InstanceAdmin {
					if actor.TenantRole == "" {
						actor.Kind = "tenant_admin"
						actor.TenantRole = role
					}
					if actor.UserID == uuid.Nil {
						if id, err := uuid.Parse(r.Header.Get("X-Cypra-User-Id")); err == nil {
							actor.UserID = id
							logging.SetActorID(r.Context(), id.String())
						}
					}
				}
			}
			if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
				plaintext := strings.TrimPrefix(authz, "Bearer ")
				if db != nil {
					token, err := (pat.Service{DB: db}).Authenticate(r.Context(), plaintext)
					if err != nil {
						writeError(w, http.StatusUnauthorized, "auth.pat_invalid")
						return
					}
					actor = Actor{Kind: "pat", PersonalTokenID: plaintext, TenantID: token.TenantID, UserID: token.UserID, Scopes: token.Scopes}
					logging.SetActorID(r.Context(), token.UserID.String())
					upsertPATSession(db, token, r.UserAgent())
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, actor)))
		})
	}
}

func actorFromSessionCookie(r *http.Request, db *sql.DB) Actor {
	cookie, err := r.Cookie("cypra_session")
	if err != nil {
		return Actor{}
	}
	sessionID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return Actor{}
	}
	var adminID uuid.UUID
	err = db.QueryRowContext(r.Context(), `SELECT instance_admin_id FROM instance_admin_sessions WHERE id = $1 AND revoked_at IS NULL AND expires_at > $2`, sessionID, time.Now().UTC()).Scan(&adminID)
	if err == nil {
		logging.SetActorID(r.Context(), adminID.String())
		bumpInstanceAdminSessionLastSeen(db, sessionID)
		return Actor{Kind: "instance_admin", InstanceAdmin: true, InstanceAdminID: adminID}
	}
	var userID uuid.UUID
	var tenantID uuid.UUID
	var role sql.NullString
	err = db.QueryRowContext(r.Context(), `SELECT s.subject_id, s.tenant_id, tm.role::text FROM sessions s LEFT JOIN tenant_memberships tm ON tm.tenant_id = s.tenant_id AND tm.user_id = s.subject_id WHERE s.id = $1 AND s.subject_kind = 'user' AND s.revoked_at IS NULL AND s.expires_at > $2`, sessionID, time.Now().UTC()).Scan(&userID, &tenantID, &role)
	if err != nil {
		return Actor{}
	}
	logging.SetActorID(r.Context(), userID.String())
	bumpUserSessionLastSeen(db, sessionID)
	actor := Actor{Kind: "tenant_admin", TenantID: tenantID, UserID: userID}
	if role.Valid {
		actor.TenantRole = role.String
	}
	return actor
}

// bumpUserSessionLastSeen fires off an async UPDATE so a busy request path
// isn't blocked on a write. Errors are logged-and-forgotten.
func bumpUserSessionLastSeen(db *sql.DB, sessionID uuid.UUID) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = now() WHERE id = $1`, sessionID)
	}()
}

func bumpInstanceAdminSessionLastSeen(db *sql.DB, sessionID uuid.UUID) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `UPDATE instance_admin_sessions SET last_seen_at = now() WHERE id = $1`, sessionID)
	}()
}

// upsertPATSession surfaces PAT-authenticated traffic in the user's session
// list. The session id mirrors the PAT id so each PAT collapses to a single
// session row regardless of request count; expires_at tracks the token's own
// expiry (with a far-future fallback when the PAT has none).
func upsertPATSession(db *sql.DB, token pat.Token, userAgent string) {
	expires := time.Now().UTC().Add(time.Hour * 24 * 365 * 10)
	if token.ExpiresAt != nil {
		expires = *token.ExpiresAt
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `INSERT INTO sessions (id, subject_id, subject_kind, tenant_id, expires_at, user_agent, auth_kind) VALUES ($1, $2, 'user', $3, $4, $5, 'pat') ON CONFLICT (id) DO UPDATE SET last_seen_at = now(), expires_at = EXCLUDED.expires_at, user_agent = EXCLUDED.user_agent`,
			token.ID, token.UserID, token.TenantID, expires, userAgent)
	}()
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
		actor := ActorFromContext(r.Context())
		if actor.InstanceAdmin {
			next.ServeHTTP(w, r)
			return
		}
		role := actor.TenantRole
		if role != "owner" && role != "admin" {
			writeError(w, http.StatusForbidden, "auth.forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireTenantRoleOrPATScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := ActorFromContext(r.Context())
			if actor.InstanceAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if actor.Kind == "pat" {
				if !hasScope(actor.Scopes, scope) {
					writeError(w, http.StatusForbidden, "auth.scope_forbidden")
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			role := actor.TenantRole
			if role != "owner" && role != "admin" {
				writeError(w, http.StatusForbidden, "auth.forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required || scope == "*" {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `"}`))
}
