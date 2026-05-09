package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/auth/invite"
	"gorm.io/gorm"
)

func (s *Server) resetUserPassword(w http.ResponseWriter, r *http.Request) {
	tenant, userID, email, ok := s.resolveUserAction(w, r)
	if !ok {
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE password_credentials SET must_reset = true, updated_at = now() WHERE tenant_id = ? AND user_id = ?`, tenant.ID, userID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "user.password_reset_failed")
		return
	}
	token, inviteID, err := (invite.Service{DB: s.DB}).Issue(r.Context(), invite.IssueRequest{TenantID: &tenant.ID, Email: email, Role: "member", RedirectURL: "/reset", CreatedByKind: "tenant_admin"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.issue_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "user.password_reset", ResourceKind: "user", ResourceID: &userID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "invite_id": inviteID, "token": token})
}

func (s *Server) reinviteUser(w http.ResponseWriter, r *http.Request) {
	tenant, userID, email, ok := s.resolveUserAction(w, r)
	if !ok {
		return
	}
	token, inviteID, err := (invite.Service{DB: s.DB}).Issue(r.Context(), invite.IssueRequest{TenantID: &tenant.ID, Email: email, Role: "member", RedirectURL: "/invite", CreatedByKind: "tenant_admin"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.issue_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "user.reinvite", ResourceKind: "user", ResourceID: &userID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "invite_id": inviteID, "token": token})
}

func (s *Server) disableUserMFA(w http.ResponseWriter, r *http.Request) {
	tenant, userID, _, ok := s.resolveUserAction(w, r)
	if !ok {
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM totp_credentials WHERE tenant_id = ? AND user_id = ?`, tenant.ID, userID).Error; err != nil {
			return err
		}
		return tx.Exec(`DELETE FROM passkey_credentials WHERE tenant_id = ? AND user_id = ? AND purpose = 'second_factor'`, tenant.ID, userID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "user.mfa_disable_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "user.mfa_disable", ResourceKind: "user", ResourceID: &userID})
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func (s *Server) inviteUserFactorEnrollment(w http.ResponseWriter, r *http.Request) {
	tenant, userID, email, ok := s.resolveUserAction(w, r)
	if !ok {
		return
	}
	token, inviteID, err := (invite.Service{DB: s.DB}).Issue(r.Context(), invite.IssueRequest{TenantID: &tenant.ID, Email: email, Role: "member", RedirectURL: "/2fa", CreatedByKind: "tenant_admin"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.issue_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "user.factor_enroll_invite", ResourceKind: "user", ResourceID: &userID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "sent", "invite_id": inviteID, "token": token})
}

func (s *Server) resolveUserAction(w http.ResponseWriter, r *http.Request) (Tenant, uuid.UUID, string, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return Tenant{}, uuid.Nil, "", false
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return Tenant{}, uuid.Nil, "", false
	}
	var rows []struct {
		Email string `gorm:"column:email"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT email FROM users WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL`, tenant.ID, &rows, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return Tenant{}, uuid.Nil, "", false
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "user.not_found")
		return Tenant{}, uuid.Nil, "", false
	}
	return tenant, userID, rows[0].Email, true
}
