package httpserver

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"gorm.io/gorm"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,63}$`)

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, slug, name, branding, settings FROM tenants WHERE deleted_at IS NULL ORDER BY slug`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var payload tenantPayload
		if err := rows.Scan(&id, &payload.Slug, &payload.Name, &payload.Branding, &payload.Settings); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		items = append(items, map[string]any{"id": id, "slug": payload.Slug, "name": payload.Name, "branding": payload.Branding, "settings": payload.Settings})
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createTenant(w http.ResponseWriter, r *http.Request) {
	var payload tenantPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if !slugPattern.MatchString(payload.Slug) {
		writeError(w, http.StatusBadRequest, "tenant.slug_invalid")
		return
	}
	if reservedSlugs[payload.Slug] {
		writeError(w, http.StatusBadRequest, "tenant.slug_reserved")
		return
	}
	id := uuid.New()
	if len(payload.Branding) == 0 {
		payload.Branding = []byte(`{}`)
	}
	if len(payload.Settings) == 0 {
		payload.Settings = []byte(`{}`)
	}
	_, err := s.DB.ExecContext(r.Context(), `INSERT INTO tenants (id, slug, name, branding, settings) VALUES ($1, $2, $3, $4, $5)`, id, payload.Slug, payload.Name, payload.Branding, payload.Settings)
	if err != nil {
		writeError(w, http.StatusConflict, "tenant.slug_conflict")
		return
	}
	_ = s.audit.Write(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.create", ResourceKind: "tenant", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "slug": payload.Slug, "name": payload.Name})
}

func (s *Server) getTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	var payload tenantPayload
	err = s.DB.QueryRowContext(r.Context(), `SELECT slug, name, branding, settings FROM tenants WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&payload.Slug, &payload.Name, &payload.Branding, &payload.Settings)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "slug": payload.Slug, "name": payload.Name, "branding": payload.Branding, "settings": payload.Settings})
}

func (s *Server) updateTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	var payload tenantPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `UPDATE tenants SET name = $1, updated_at = now() WHERE id = $2`, payload.Name, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	_ = s.audit.Write(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.update", ResourceKind: "tenant", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": payload.Name})
}

func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	_, err = s.DB.ExecContext(r.Context(), `UPDATE tenants SET deleted_at = now() WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	_ = s.audit.Write(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.delete", ResourceKind: "tenant", ResourceID: &id})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, slug, name FROM projects WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY slug`, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var slug, name string
		if err := rows.Scan(&id, &slug, &name); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		items = append(items, map[string]any{"id": id, "slug": slug, "name": name})
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload projectPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	id := uuid.New()
	err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES (?, ?, ?, ?)`, id, tenant.ID, payload.Slug, payload.Name).Error
	})
	if err != nil {
		writeError(w, http.StatusConflict, "project.conflict")
		return
	}
	_ = s.audit.Write(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "project.create", ResourceKind: "project", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "slug": payload.Slug, "name": payload.Name})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, email, metadata FROM users WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY email`, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var email string
		var metadata json.RawMessage
		if err := rows.Scan(&id, &email, &metadata); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		items = append(items, map[string]any{"id": id, "email": email, "metadata": metadata})
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload userPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if len(payload.Metadata) == 0 {
		payload.Metadata = []byte(`{}`)
	}
	id := uuid.New()
	err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES (?, ?, ?, ?)`, id, tenant.ID, payload.Email, payload.Metadata).Error
	})
	if err != nil {
		writeError(w, http.StatusConflict, "user.conflict")
		return
	}
	_ = s.audit.Write(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "user.create", ResourceKind: "user", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "email": payload.Email, "metadata": payload.Metadata})
}
