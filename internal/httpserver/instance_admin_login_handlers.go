package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/auth/webauthn"
)

type instanceAdminLoginPayload struct {
	CeremonyID string          `json:"ceremony_id"`
	Response   json.RawMessage `json:"response"`
}

type instanceAdminBackupCodePayload struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// authInstanceAdminLogin drives the install-host instance-admin sign-in
// ceremony. POST with no body (or just `{}`) starts a discoverable webauthn
// challenge; POST with `{ceremony_id, response}` verifies the assertion and
// establishes the session. The browser picks the credential, so an email is
// not required.
func (s *Server) authInstanceAdminLogin(w http.ResponseWriter, r *http.Request) {
	var payload instanceAdminLoginPayload
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "request.invalid")
			return
		}
	}
	service := webauthn.Service{DB: s.DB}
	if strings.TrimSpace(payload.CeremonyID) == "" {
		options, ceremonyID, err := service.BeginInstanceAdminAssertion(r.Context(), webauthn.BeginInstanceAdminAssertionRequest{RPID: s.installRPID()})
		if err != nil {
			// Don't leak whether there are no admins / no passkeys vs server error.
			writeError(w, http.StatusUnauthorized, "auth.passkey_invalid")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ceremony_id": ceremonyID, "options": options})
		return
	}
	ceremonyID, err := uuid.Parse(payload.CeremonyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "webauthn.ceremony_invalid")
		return
	}
	adminID, _, err := service.FinishInstanceAdminAssertion(r.Context(), webauthn.FinishInstanceAdminAssertionRequest{RPID: s.installRPID(), Origins: []string{s.requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)})
	if err != nil || adminID == uuid.Nil {
		recordAuthAttempt("instance_admin_passkey", "fail")
		writeError(w, http.StatusUnauthorized, "auth.passkey_invalid")
		return
	}
	if _, err := s.establishInstanceAdminSession(w, r, adminID); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("instance_admin_passkey", "success")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// authInstanceAdminBackupCode lets an admin who has lost access to their
// passkey sign in by consuming a backup code. Each code is single-use.
func (s *Server) authInstanceAdminBackupCode(w http.ResponseWriter, r *http.Request) {
	var payload instanceAdminBackupCodePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	code := strings.TrimSpace(payload.Code)
	if email == "" || code == "" {
		writeError(w, http.StatusBadRequest, "auth.backup_code_required")
		return
	}
	adminID, ok := s.lookupInstanceAdminID(r, w, email)
	if !ok {
		return
	}
	consumed, err := (backupcodes.Service{DB: s.DB}).ConsumeForInstanceAdmin(r.Context(), adminID, code)
	if err != nil || !consumed {
		recordAuthAttempt("instance_admin_backup_code", "fail")
		writeError(w, http.StatusUnauthorized, "auth.backup_code_invalid")
		return
	}
	if _, err := s.establishInstanceAdminSession(w, r, adminID); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("instance_admin_backup_code", "success")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) lookupInstanceAdminID(r *http.Request, w http.ResponseWriter, email string) (uuid.UUID, bool) {
	var adminID uuid.UUID
	err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM instance_admins WHERE lower(email) = $1 AND disabled_at IS NULL`, email).Scan(&adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "auth.passkey_invalid")
			return uuid.Nil, false
		}
		writeError(w, http.StatusInternalServerError, "auth.lookup_failed")
		return uuid.Nil, false
	}
	return adminID, true
}
