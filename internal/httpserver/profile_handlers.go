package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth"
	"gorm.io/gorm"
)

type metadataPatchPayload struct {
	Metadata json.RawMessage `json:"metadata"`
}

func (s *Server) getUserMe(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		s.writeInstanceAdminMe(w, r, actor.InstanceAdminID)
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
		userID      uuid.UUID
		email       string
		displayName string
		metadata    []byte
		createdAt   time.Time
	)
	err := s.DB.QueryRowContext(
		r.Context(),
		`SELECT id, email, display_name, metadata, created_at FROM users WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenant.ID, actor.UserID,
	).Scan(&userID, &email, &displayName, &metadata, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "users.me.not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "users.me.fetch_failed")
		return
	}
	if displayName == "" {
		if at := strings.IndexByte(email, '@'); at > 0 {
			displayName = email[:at]
		} else {
			displayName = email
		}
	}
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":         "user",
		"id":           userID.String(),
		"sub":          userID.String(),
		"email":        email,
		"display_name": displayName,
		"tenant_id":    tenant.ID.String(),
		"created_at":   createdAt,
		"metadata":     json.RawMessage(metadata),
	})
}

func (s *Server) writeInstanceAdminMe(w http.ResponseWriter, r *http.Request, adminID uuid.UUID) {
	var (
		id          uuid.UUID
		email       string
		displayName string
		metadata    []byte
		createdAt   time.Time
	)
	err := s.DB.QueryRowContext(
		r.Context(),
		`SELECT id, email, display_name, metadata, created_at FROM instance_admins WHERE id = $1 AND disabled_at IS NULL`,
		adminID,
	).Scan(&id, &email, &displayName, &metadata, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "instance_admin.me.not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "instance_admin.me.fetch_failed")
		return
	}
	if displayName == "" {
		if at := strings.IndexByte(email, '@'); at > 0 {
			displayName = email[:at]
		} else {
			displayName = email
		}
	}
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"kind":         "instance_admin",
		"id":           id.String(),
		"sub":          id.String(),
		"email":        email,
		"display_name": displayName,
		"tenant_id":    nil,
		"created_at":   createdAt,
		"metadata":     json.RawMessage(metadata),
	})
}

func (s *Server) patchUserMe(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.UserID == uuid.Nil {
		writeJSON(w, http.StatusOK, map[string]any{"metadata": json.RawMessage(`{}`)})
		return
	}
	payload, ok := decodeMetadataPatch(w, r)
	if !ok {
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE users SET metadata = metadata || ?::jsonb, updated_at = now() WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL`, payload.Metadata, tenant.ID, actor.UserID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "users.me.update_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"metadata": payload.Metadata})
}

func (s *Server) patchInstanceAdminMe(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if actor.InstanceAdminID == uuid.Nil {
		writeJSON(w, http.StatusOK, map[string]any{"metadata": json.RawMessage(`{}`)})
		return
	}
	payload, ok := decodeMetadataPatch(w, r)
	if !ok {
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE instance_admins SET metadata = metadata || $1::jsonb WHERE id = $2 AND disabled_at IS NULL`, payload.Metadata, actor.InstanceAdminID); err != nil {
		writeError(w, http.StatusInternalServerError, "instance_admins.me.update_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"metadata": payload.Metadata})
}

func decodeMetadataPatch(w http.ResponseWriter, r *http.Request) (metadataPatchPayload, bool) {
	var payload metadataPatchPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || !json.Valid(payload.Metadata) || len(payload.Metadata) == 0 {
		writeError(w, http.StatusBadRequest, "metadata.invalid")
		return payload, false
	}
	return payload, true
}
