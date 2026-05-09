package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/pat"
	"gorm.io/gorm"
)

type tenantMemberResponse struct {
	ID         uuid.UUID  `json:"id" gorm:"column:id"`
	UserID     uuid.UUID  `json:"user_id" gorm:"column:user_id"`
	Email      string     `json:"email" gorm:"column:email"`
	Role       string     `json:"role" gorm:"column:role"`
	CreatedAt  time.Time  `json:"created_at" gorm:"column:created_at"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty" gorm:"column:last_seen_at"`
}

type updateTenantMemberPayload struct {
	Role string `json:"role"`
}

func (s *Server) listTenantMembers(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var members []tenantMemberResponse
	if err := s.TenantDB.RawScan(r.Context(), `
		SELECT tm.id, tm.user_id, u.email, tm.role::text AS role, tm.created_at, max(s.last_seen_at) AS last_seen_at
		FROM tenant_memberships tm
		JOIN users u ON u.id = tm.user_id AND u.tenant_id = tm.tenant_id
		LEFT JOIN sessions s ON s.subject_id = tm.user_id AND s.tenant_id = tm.tenant_id AND s.subject_kind = 'user' AND s.revoked_at IS NULL
		WHERE tm.tenant_id = ? AND u.deleted_at IS NULL
		GROUP BY tm.id, tm.user_id, u.email, tm.role, tm.created_at
		ORDER BY tm.created_at ASC`, tenant.ID, &members, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, members)
}

func (s *Server) updateTenantMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	membershipID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "member.id_invalid")
		return
	}
	var payload updateTenantMemberPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if payload.Role != "owner" && payload.Role != "admin" && payload.Role != "member" {
		writeError(w, http.StatusBadRequest, "member.role_invalid")
		return
	}
	member, err := s.loadTenantMember(r, tenant.ID, membershipID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "member.not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if member.Role == "owner" && payload.Role != "owner" && !s.hasAnotherOwner(r, tenant.ID, membershipID) {
		writeError(w, http.StatusConflict, "member.last_owner")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE tenant_memberships SET role = ? WHERE tenant_id = ? AND id = ?`, payload.Role, tenant.ID, membershipID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "member.update_failed")
		return
	}
	if payload.Role == "member" {
		_ = (pat.Service{DB: s.DB}).RevokeForUser(r.Context(), tenant.ID, member.UserID, "membership_downgrade")
	}
	member.Role = payload.Role
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "member.role_update", ResourceKind: "tenant_membership", ResourceID: &membershipID, StateAfter: map[string]string{"role": payload.Role}})
	writeJSON(w, http.StatusOK, member)
}

func (s *Server) removeTenantMember(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	membershipID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "member.id_invalid")
		return
	}
	member, err := s.loadTenantMember(r, tenant.ID, membershipID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "member.not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if member.Role == "owner" && !s.hasAnotherOwner(r, tenant.ID, membershipID) {
		writeError(w, http.StatusConflict, "member.last_owner")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`DELETE FROM tenant_memberships WHERE tenant_id = ? AND id = ?`, tenant.ID, membershipID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "member.remove_failed")
		return
	}
	_ = (pat.Service{DB: s.DB}).RevokeForUser(r.Context(), tenant.ID, member.UserID, "membership_removed")
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "member.remove", ResourceKind: "tenant_membership", ResourceID: &membershipID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadTenantMember(r *http.Request, tenantID, membershipID uuid.UUID) (tenantMemberResponse, error) {
	var rows []tenantMemberResponse
	if err := s.TenantDB.RawScan(r.Context(), `SELECT tm.id, tm.user_id, u.email, tm.role::text, tm.created_at FROM tenant_memberships tm JOIN users u ON u.id = tm.user_id AND u.tenant_id = tm.tenant_id WHERE tm.tenant_id = ? AND tm.id = ? AND u.deleted_at IS NULL`, tenantID, &rows, tenantID, membershipID); err != nil {
		return tenantMemberResponse{}, err
	}
	if len(rows) == 0 {
		return tenantMemberResponse{}, sql.ErrNoRows
	}
	return rows[0], nil
}

func (s *Server) hasAnotherOwner(r *http.Request, tenantID, membershipID uuid.UUID) bool {
	var rows []struct {
		Count int `gorm:"column:count"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT count(*) AS count FROM tenant_memberships WHERE tenant_id = ? AND role = 'owner' AND id <> ?`, tenantID, &rows, tenantID, membershipID); err != nil {
		return false
	}
	if len(rows) == 0 {
		return false
	}
	return rows[0].Count > 0
}
