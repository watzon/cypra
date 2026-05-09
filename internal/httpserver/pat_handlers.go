package httpserver

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/auth"
	"github.com/watzon/cypra/internal/pat"
)

func (s *Server) listPATs(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	actor := auth.ActorFromContext(r.Context())
	// Instance admins see every PAT in the tenant; tenant users see their own.
	if actor.InstanceAdmin {
		tokens, err := s.listAllTenantPATs(r, tenant.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "pat.list_failed")
			return
		}
		writeJSON(w, http.StatusOK, tokens)
		return
	}
	_, userID, resolved := s.patSubject(w, r)
	if !resolved {
		return
	}
	tokens, err := (pat.Service{DB: s.DB}).List(r.Context(), tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pat.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) listAllTenantPATs(r *http.Request, tenantID uuid.UUID) ([]pat.Token, error) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, tenant_id, user_id, name, token_suffix, scopes, created_at, expires_at, last_used_at, revoked_at FROM personal_access_tokens WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	tokens := []pat.Token{}
	for rows.Next() {
		var token pat.Token
		var expires, lastUsed, revoked sql.NullTime
		if err := rows.Scan(&token.ID, &token.TenantID, &token.UserID, &token.Name, &token.Last4, pq.Array(&token.Scopes), &token.CreatedAt, &expires, &lastUsed, &revoked); err != nil {
			return nil, err
		}
		if expires.Valid {
			t := expires.Time
			token.ExpiresAt = &t
		}
		if lastUsed.Valid {
			t := lastUsed.Time
			token.LastUsedAt = &t
		}
		if revoked.Valid {
			t := revoked.Time
			token.RevokedAt = &t
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Server) createPAT(w http.ResponseWriter, r *http.Request) {
	tenant, userID, ok := s.patSubject(w, r)
	if !ok {
		return
	}
	var payload struct {
		Name      string     `json:"name"`
		Scopes    []string   `json:"scopes"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	created, err := (pat.Service{DB: s.DB}).CreateWithExpiry(r.Context(), tenant.ID, userID, payload.Name, payload.Scopes, payload.ExpiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pat.create_failed")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) revokePAT(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	tokenID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "pat.id_invalid")
		return
	}
	actor := auth.ActorFromContext(r.Context())
	// Instance admins can revoke any PAT in the tenant; users can only revoke their own.
	if actor.InstanceAdmin {
		res, err := s.DB.ExecContext(r.Context(), `UPDATE personal_access_tokens SET revoked_at = now() WHERE tenant_id = $1 AND id = $2 AND revoked_at IS NULL`, tenant.ID, tokenID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "pat.revoke_failed")
			return
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			writeError(w, http.StatusNotFound, "pat.not_found")
			return
		}
		_, _ = s.DB.ExecContext(r.Context(), `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND auth_kind = 'pat' AND revoked_at IS NULL`, tokenID)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_, userID, resolved := s.patSubject(w, r)
	if !resolved {
		return
	}
	if err := (pat.Service{DB: s.DB}).Revoke(r.Context(), tenant.ID, userID, tokenID); err != nil {
		writeError(w, http.StatusInternalServerError, "pat.revoke_failed")
		return
	}
	// Tear down the synthetic session that PAT auth materialised so the
	// dashboard's session list stops showing it.
	_, _ = s.DB.ExecContext(r.Context(), `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND auth_kind = 'pat' AND revoked_at IS NULL`, tokenID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) patSubject(w http.ResponseWriter, r *http.Request) (Tenant, uuid.UUID, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return Tenant{}, uuid.Nil, false
	}
	actor := auth.ActorFromContext(r.Context())
	if actor.UserID != uuid.Nil && (actor.TenantID == uuid.Nil || actor.TenantID == tenant.ID) {
		return tenant, actor.UserID, true
	}
	if userID, ok := s.currentUserSession(r, tenant.ID); ok {
		return tenant, userID, true
	}
	writeError(w, http.StatusUnauthorized, "auth.unauthorized")
	return Tenant{}, uuid.Nil, false
}
