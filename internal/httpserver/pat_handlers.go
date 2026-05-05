package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/pat"
)

func (s *Server) listPATs(w http.ResponseWriter, r *http.Request) {
	tenant, userID, ok := s.patSubject(w, r)
	if !ok {
		return
	}
	tokens, err := (pat.Service{DB: s.DB}).List(r.Context(), tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pat.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (s *Server) createPAT(w http.ResponseWriter, r *http.Request) {
	tenant, userID, ok := s.patSubject(w, r)
	if !ok {
		return
	}
	var payload struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	created, err := (pat.Service{DB: s.DB}).Create(r.Context(), tenant.ID, userID, payload.Name, payload.Scopes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pat.create_failed")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) revokePAT(w http.ResponseWriter, r *http.Request) {
	tenant, userID, ok := s.patSubject(w, r)
	if !ok {
		return
	}
	tokenID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "pat.id_invalid")
		return
	}
	if err := (pat.Service{DB: s.DB}).Revoke(r.Context(), tenant.ID, userID, tokenID); err != nil {
		writeError(w, http.StatusInternalServerError, "pat.revoke_failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) patSubject(w http.ResponseWriter, r *http.Request) (Tenant, uuid.UUID, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return Tenant{}, uuid.Nil, false
	}
	userID, err := uuid.Parse(r.Header.Get("X-Cypra-User-Id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return Tenant{}, uuid.Nil, false
	}
	return tenant, userID, true
}
