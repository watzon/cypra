package httpserver

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/auth"
)

// passkeyRow is the wire shape for /api/v1/auth/passkeys.
type passkeyRow struct {
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	AAGUID     *string   `json:"aaguid,omitempty"`
	Transports []string  `json:"transports"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt *string   `json:"last_used_at,omitempty"`
	ThisDevice bool      `json:"this_device"`
}

// authListPasskeys returns the passkeys for the calling user. The caller is
// identified via auth.MiddlewareWithPAT; we never list other users' passkeys.
// Instance admins get their own passkey list (instance_admin_passkey_credentials).
func (s *Server) authListPasskeys(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		s.listInstanceAdminPasskeys(w, r, actor.InstanceAdminID)
		return
	}
	if actor.UserID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	rows, err := s.DB.QueryContext(
		r.Context(),
		`SELECT id, nickname, aaguid, transports, created_at, last_used_at FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2 ORDER BY created_at DESC`,
		tenant.ID, actor.UserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "passkey.list_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	out := []passkeyRow{}
	for rows.Next() {
		var (
			id         uuid.UUID
			nickname   string
			aaguid     uuid.NullUUID
			transports pq.StringArray
			createdAt  time.Time
			lastUsedAt sql.NullTime
		)
		if err := rows.Scan(&id, &nickname, &aaguid, &transports, &createdAt, &lastUsedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "passkey.list_failed")
			return
		}
		label := nickname
		if label == "" {
			label = "Passkey"
		}
		row := passkeyRow{
			ID:         id.String(),
			Label:      label,
			Transports: append([]string{}, transports...),
			CreatedAt:  createdAt,
		}
		if aaguid.Valid {
			s := aaguid.UUID.String()
			row.AAGUID = &s
		}
		if lastUsedAt.Valid {
			s := lastUsedAt.Time.Format(time.RFC3339)
			row.LastUsedAt = &s
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listInstanceAdminPasskeys(w http.ResponseWriter, r *http.Request, adminID uuid.UUID) {
	rows, err := s.DB.QueryContext(
		r.Context(),
		`SELECT id, nickname, aaguid, transports, created_at, last_used_at FROM instance_admin_passkey_credentials WHERE instance_admin_id = $1 ORDER BY created_at DESC`,
		adminID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "passkey.list_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	out := []passkeyRow{}
	for rows.Next() {
		var (
			id         uuid.UUID
			nickname   string
			aaguid     uuid.NullUUID
			transports pq.StringArray
			createdAt  time.Time
			lastUsedAt sql.NullTime
		)
		if err := rows.Scan(&id, &nickname, &aaguid, &transports, &createdAt, &lastUsedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "passkey.list_failed")
			return
		}
		label := nickname
		if label == "" {
			label = "Passkey"
		}
		row := passkeyRow{
			ID:         id.String(),
			Label:      label,
			Transports: append([]string{}, transports...),
			CreatedAt:  createdAt,
		}
		if aaguid.Valid {
			s := aaguid.UUID.String()
			row.AAGUID = &s
		}
		if lastUsedAt.Valid {
			s := lastUsedAt.Time.Format(time.RFC3339)
			row.LastUsedAt = &s
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

// authDeletePasskey removes a passkey. Refuses if the deletion would leave the
// user with zero passkeys (last-passkey self-removal guard).
func (s *Server) authDeletePasskey(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		writeError(w, http.StatusNotImplemented, "passkey.instance_admin_delete_unsupported")
		return
	}
	if actor.UserID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "passkey.id_invalid")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err := tx.QueryRowContext(
		r.Context(),
		`SELECT count(*) FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2`,
		tenant.ID, actor.UserID,
	).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "passkey.list_failed")
		return
	}
	if count <= 1 {
		writeError(w, http.StatusConflict, "passkey.last_remaining")
		return
	}
	res, err := tx.ExecContext(
		r.Context(),
		`DELETE FROM passkey_credentials WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
		id, tenant.ID, actor.UserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "passkey.delete_failed")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "passkey.not_found")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	tenantID := tenant.ID
	resourceID := id
	actorID := actor.UserID
	s.recordMutationAudit(r.Context(), audit.Entry{
		TenantID:     &tenantID,
		ActorKind:    "user",
		ActorID:      &actorID,
		Action:       "passkey.delete",
		ResourceKind: "passkey",
		ResourceID:   &resourceID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// sessionRow is the wire shape for /api/v1/auth/sessions.
type sessionRow struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IP         *string   `json:"ip,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	AuthKind   string    `json:"auth_kind"`
	Current    bool      `json:"current"`
}

// authListSessions returns the active sessions for the calling user. The
// session that issued the request is flagged with current=true. Instance
// admins get their parallel session list (instance_admin_sessions).
func (s *Server) authListSessions(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		s.listInstanceAdminSessions(w, r, actor.InstanceAdminID)
		return
	}
	if actor.UserID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	currentSession := currentSessionID(r)
	rows, err := s.DB.QueryContext(
		r.Context(),
		`SELECT id, created_at, last_seen_at, expires_at, ip, user_agent, auth_kind FROM sessions WHERE tenant_id = $1 AND subject_id = $2 AND subject_kind = 'user' AND revoked_at IS NULL AND expires_at > now() ORDER BY last_seen_at DESC`,
		tenant.ID, actor.UserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session.list_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	out := []sessionRow{}
	for rows.Next() {
		var (
			id         uuid.UUID
			createdAt  time.Time
			lastSeen   time.Time
			expiresAt  time.Time
			ip         sql.NullString
			userAgent  sql.NullString
			authKind   string
		)
		if err := rows.Scan(&id, &createdAt, &lastSeen, &expiresAt, &ip, &userAgent, &authKind); err != nil {
			writeError(w, http.StatusInternalServerError, "session.list_failed")
			return
		}
		row := sessionRow{
			ID:         id.String(),
			CreatedAt:  createdAt,
			LastSeenAt: lastSeen,
			ExpiresAt:  expiresAt,
			AuthKind:   authKind,
			Current:    currentSession != "" && id.String() == currentSession,
		}
		if ip.Valid {
			value := ip.String
			row.IP = &value
		}
		if userAgent.Valid {
			value := userAgent.String
			row.UserAgent = &value
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listInstanceAdminSessions(w http.ResponseWriter, r *http.Request, adminID uuid.UUID) {
	currentSession := currentSessionID(r)
	rows, err := s.DB.QueryContext(
		r.Context(),
		`SELECT id, created_at, last_seen_at, expires_at, ip, user_agent FROM instance_admin_sessions WHERE instance_admin_id = $1 AND revoked_at IS NULL AND expires_at > now() ORDER BY last_seen_at DESC`,
		adminID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session.list_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	out := []sessionRow{}
	for rows.Next() {
		var (
			id        uuid.UUID
			createdAt time.Time
			lastSeen  time.Time
			expiresAt time.Time
			ip        sql.NullString
			userAgent sql.NullString
		)
		if err := rows.Scan(&id, &createdAt, &lastSeen, &expiresAt, &ip, &userAgent); err != nil {
			writeError(w, http.StatusInternalServerError, "session.list_failed")
			return
		}
		row := sessionRow{
			ID:         id.String(),
			CreatedAt:  createdAt,
			LastSeenAt: lastSeen,
			ExpiresAt:  expiresAt,
			AuthKind:   "cookie",
			Current:    currentSession != "" && id.String() == currentSession,
		}
		if ip.Valid {
			value := ip.String
			row.IP = &value
		}
		if userAgent.Valid {
			value := userAgent.String
			row.UserAgent = &value
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

// authRevokeSession revokes a single session belonging to the caller. Refuses
// to revoke the current session — clients should use signOut() for that.
func (s *Server) authRevokeSession(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		s.revokeInstanceAdminSession(w, r, actor.InstanceAdminID)
		return
	}
	if actor.UserID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "session.id_invalid")
		return
	}
	if currentSession := currentSessionID(r); currentSession != "" && id.String() == currentSession {
		writeError(w, http.StatusConflict, "session.cannot_revoke_current")
		return
	}
	res, err := s.DB.ExecContext(
		r.Context(),
		`UPDATE sessions SET revoked_at = now() WHERE id = $1 AND tenant_id = $2 AND subject_id = $3 AND subject_kind = 'user' AND revoked_at IS NULL`,
		id, tenant.ID, actor.UserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session.revoke_failed")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "session.not_found")
		return
	}
	tenantID := tenant.ID
	resourceID := id
	actorID := actor.UserID
	s.recordMutationAudit(r.Context(), audit.Entry{
		TenantID:     &tenantID,
		ActorKind:    "user",
		ActorID:      &actorID,
		Action:       "session.revoke",
		ResourceKind: "session",
		ResourceID:   &resourceID,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) revokeInstanceAdminSession(w http.ResponseWriter, r *http.Request, adminID uuid.UUID) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "session.id_invalid")
		return
	}
	if currentSession := currentSessionID(r); currentSession != "" && id.String() == currentSession {
		writeError(w, http.StatusConflict, "session.cannot_revoke_current")
		return
	}
	res, err := s.DB.ExecContext(
		r.Context(),
		`UPDATE instance_admin_sessions SET revoked_at = now() WHERE id = $1 AND instance_admin_id = $2 AND revoked_at IS NULL`,
		id, adminID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session.revoke_failed")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		writeError(w, http.StatusNotFound, "session.not_found")
		return
	}
	resourceID := id
	actorID := adminID
	s.recordMutationAudit(r.Context(), audit.Entry{
		ActorKind:    "instance_admin",
		ActorID:      &actorID,
		Action:       "session.revoke",
		ResourceKind: "session",
		ResourceID:   &resourceID,
	})
	w.WriteHeader(http.StatusNoContent)
}

// authListMFAFactors returns TOTP enrollment + backup-code state for the
// calling user. Instance admins don't have TOTP; their factor list returns
// the backup-code count from `instance_admin_backup_codes` and a null totp.
func (s *Server) authListMFAFactors(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		s.listInstanceAdminMFAFactors(w, r, actor.InstanceAdminID)
		return
	}
	if actor.UserID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var (
		confirmedAt sql.NullTime
		hasTOTP     bool
	)
	err := s.DB.QueryRowContext(
		r.Context(),
		`SELECT confirmed_at FROM totp_credentials WHERE tenant_id = $1 AND user_id = $2`,
		tenant.ID, actor.UserID,
	).Scan(&confirmedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "mfa.fetch_failed")
		return
	}
	hasTOTP = err == nil

	var remaining, total int
	if err := s.DB.QueryRowContext(
		r.Context(),
		`SELECT COUNT(*) FILTER (WHERE used_at IS NULL), COUNT(*) FROM user_backup_codes WHERE tenant_id = $1 AND user_id = $2`,
		tenant.ID, actor.UserID,
	).Scan(&remaining, &total); err != nil {
		writeError(w, http.StatusInternalServerError, "mfa.fetch_failed")
		return
	}

	response := map[string]any{
		"backup_codes": map[string]int{
			"remaining": remaining,
			"total":     total,
		},
	}
	if hasTOTP {
		totp := map[string]any{"enrolled": confirmedAt.Valid}
		if confirmedAt.Valid {
			totp["confirmed_at"] = confirmedAt.Time
		}
		response["totp"] = totp
	} else {
		response["totp"] = nil
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) listInstanceAdminMFAFactors(w http.ResponseWriter, r *http.Request, adminID uuid.UUID) {
	var remaining, total int
	if err := s.DB.QueryRowContext(
		r.Context(),
		`SELECT COUNT(*) FILTER (WHERE used_at IS NULL), COUNT(*) FROM instance_admin_backup_codes WHERE instance_admin_id = $1`,
		adminID,
	).Scan(&remaining, &total); err != nil {
		writeError(w, http.StatusInternalServerError, "mfa.fetch_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"totp": nil,
		"backup_codes": map[string]int{
			"remaining": remaining,
			"total":     total,
		},
	})
}

// currentSessionID resolves the session id that authenticated this request,
// preferring the explicit X-Cypra-Session-Id header (used in tests) and
// falling back to the cypra_session cookie.
func currentSessionID(r *http.Request) string {
	if v := r.Header.Get("X-Cypra-Session-Id"); v != "" {
		return v
	}
	if cookie, err := r.Cookie("cypra_session"); err == nil {
		return cookie.Value
	}
	return ""
}
