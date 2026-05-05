package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth"
)

type metadataPatchPayload struct {
	Metadata json.RawMessage `json:"metadata"`
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
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE users SET metadata = metadata || $1::jsonb, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`, payload.Metadata, actor.UserID); err != nil {
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
