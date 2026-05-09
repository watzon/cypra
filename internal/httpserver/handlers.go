package httpserver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/audit"
	cypra "github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/observability"
	"github.com/watzon/cypra/internal/oidc"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,63}$`)

func (s *Server) listTenants(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT t.id, t.slug, t.name, t.branding, t.settings, t.created_at,
			(SELECT count(*) FROM tenant_memberships m WHERE m.tenant_id = t.id) AS member_count,
			(SELECT count(*) FROM users u WHERE u.tenant_id = t.id AND u.deleted_at IS NULL) AS user_count,
			(SELECT count(*) FROM projects p WHERE p.tenant_id = t.id AND p.deleted_at IS NULL) AS project_count,
			NOT EXISTS (
				SELECT 1 FROM email_provider_configs e
				WHERE e.tenant_id = t.id AND e.enabled = true
			) AS email_provider_required
		FROM tenants t
		WHERE t.deleted_at IS NULL
		ORDER BY t.slug
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var (
			id                    uuid.UUID
			payload               tenantPayload
			createdAt             time.Time
			memberCount           int64
			userCount             int64
			projectCount          int64
			emailProviderRequired bool
		)
		if err := rows.Scan(
			&id, &payload.Slug, &payload.Name, &payload.Branding, &payload.Settings,
			&createdAt, &memberCount, &userCount, &projectCount, &emailProviderRequired,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		items = append(items, map[string]any{
			"id":                      id,
			"slug":                    payload.Slug,
			"name":                    payload.Name,
			"branding":                payload.Branding,
			"settings":                payload.Settings,
			"created_at":              createdAt,
			"member_count":            memberCount,
			"user_count":              userCount,
			"project_count":           projectCount,
			"email_provider_required": emailProviderRequired,
		})
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
	key, err := cypra.GenerateSigningKey(id, 1, cypra.SigningAlgRS256, s.MasterKey, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.signing_key_failed")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(r.Context(), `INSERT INTO tenants (id, slug, name, branding, settings) VALUES ($1, $2, $3, $4, $5)`, id, payload.Slug, payload.Name, payload.Branding, payload.Settings)
	if err != nil {
		writeError(w, http.StatusConflict, "tenant.slug_conflict")
		return
	}
	_, err = tx.ExecContext(r.Context(), `INSERT INTO oidc_signing_keys (tenant_id, kid, algorithm, public_key_jwk, private_key_encrypted, state, activated_at, retires_at) VALUES ($1, $2, $3, $4, $5, 'active', $6, $7)`, id, key.KID, key.Algorithm, []byte(key.PublicJWK), key.PrivateKeyEncrypted, key.ActivatedAt, key.RetiresAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.signing_key_failed")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.create", ResourceKind: "tenant", ResourceID: &id, StateAfter: payload})
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
	s.recordMutationAudit(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.update", ResourceKind: "tenant", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": payload.Name})
}

func (s *Server) deleteTenant(w http.ResponseWriter, r *http.Request) {
	s.scheduleTenantDeletion(w, r)
}

func (s *Server) suspendTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	if err := s.applyTenantSuspension(r.Context(), id, true); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.suspend_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.suspend", ResourceKind: "tenant", ResourceID: &id})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "suspended": true})
}

func (s *Server) resumeTenant(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	if err := s.applyTenantSuspension(r.Context(), id, false); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.resume_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.resume", ResourceKind: "tenant", ResourceID: &id})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "suspended": false})
}

func (s *Server) applyTenantSuspension(ctx context.Context, id uuid.UUID, suspended bool) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if suspended {
		if _, err = tx.ExecContext(ctx, `UPDATE sessions SET revoked_at = now() WHERE tenant_id = $1 AND revoked_at IS NULL`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE oidc_refresh_tokens SET revoked_at = now(), revoke_reason = 'tenant_suspend' WHERE tenant_id = $1 AND revoked_at IS NULL`, id); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE tenants SET settings = settings || jsonb_build_object('suspended_at', now()), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE tenants SET settings = settings - 'suspended_at', updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Server) scheduleTenantDeletion(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	var payload struct {
		Slug string `json:"slug"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}
	var slug string
	if err = s.DB.QueryRowContext(r.Context(), `SELECT slug FROM tenants WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&slug); err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if payload.Slug != "" && payload.Slug != slug {
		writeError(w, http.StatusBadRequest, "tenant.slug_confirmation_invalid")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(r.Context(), `UPDATE sessions SET revoked_at = now() WHERE tenant_id = $1 AND revoked_at IS NULL`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.delete_failed")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE oidc_refresh_tokens SET revoked_at = now(), revoke_reason = 'tenant_delete' WHERE tenant_id = $1 AND revoked_at IS NULL`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.delete_failed")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE oidc_signing_keys SET state = 'sunsetting', sunset_until = COALESCE(sunset_until, now() + interval '30 days') WHERE tenant_id = $1 AND state <> 'sunsetting'`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.delete_failed")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `UPDATE tenants SET settings = settings || jsonb_build_object('suspended_at', COALESCE(settings->>'suspended_at', now()::text), 'deletion_scheduled_at', now(), 'deletion_cancellable_until', now() + interval '7 days', 'hard_delete_after', now() + interval '30 days'), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.delete", ResourceKind: "tenant", ResourceID: &id})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "slug": slug, "deletion_scheduled": true})
}

func (s *Server) cancelTenantDeletion(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(r.Context(), `UPDATE tenants SET settings = settings - 'deletion_scheduled_at' - 'deletion_cancellable_until' - 'hard_delete_after' - 'suspended_at', updated_at = now() WHERE id = $1 AND deleted_at IS NULL AND settings ? 'deletion_scheduled_at' AND (settings->>'deletion_cancellable_until')::timestamptz > now()`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.delete_cancel_failed")
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		writeError(w, http.StatusConflict, "tenant.deletion_not_cancellable")
		return
	}
	if _, err = tx.ExecContext(r.Context(), `WITH newest AS (SELECT id FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'sunsetting' ORDER BY activated_at DESC LIMIT 1) UPDATE oidc_signing_keys SET state = CASE WHEN id IN (SELECT id FROM newest) THEN 'active'::signing_key_state ELSE 'retired'::signing_key_state END, sunset_until = NULL WHERE tenant_id = $1 AND state = 'sunsetting'`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.delete_cancel_failed")
		return
	}
	if err = tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{ActorKind: "instance_admin", Action: "tenant.delete_cancel", ResourceKind: "tenant", ResourceID: &id})
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "deletion_scheduled": false})
}

// RunTenantDeletionCleanup permanently removes tenants whose deletion window has elapsed.
func RunTenantDeletionCleanup(ctx context.Context, db *sql.DB, cutoff time.Time) error {
	ctx, span := observability.StartSpan(ctx, "tenant_delete.cleanup", attribute.String("job", "tenant_delete_cleanup"))
	defer span.End()
	rows, err := db.QueryContext(ctx, `SELECT id FROM tenants WHERE deleted_at IS NULL AND settings ? 'hard_delete_after' AND (settings->>'hard_delete_after')::timestamptz <= $1`, cutoff)
	if err != nil {
		observability.RecordError(span, err)
		return err
	}
	defer func() { _ = rows.Close() }()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM oidc_signing_keys WHERE tenant_id = $1 AND state = 'sunsetting' AND sunset_until <= $2`, id, cutoff); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM oidc_signing_keys WHERE tenant_id = $1)`, id); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// RunTenantDeletionCleanupTicker runs tenant deletion cleanup immediately and then on interval.
func RunTenantDeletionCleanupTicker(ctx context.Context, db *sql.DB, interval time.Duration) error {
	if err := RunTenantDeletionCleanup(ctx, db, time.Now().UTC()); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if err := RunTenantDeletionCleanup(ctx, db, now.UTC()); err != nil {
				return err
			}
		}
	}
}

func (s *Server) tenantAuthBlocked(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	var blocked bool
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(settings ? 'suspended_at', false) OR COALESCE(settings ? 'deletion_scheduled_at', false) FROM tenants WHERE id = $1 AND deleted_at IS NULL`, tenantID).Scan(&blocked)
	return blocked, err
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var rows []struct {
		ID                      uuid.UUID      `gorm:"column:id"`
		Slug                    string         `gorm:"column:slug"`
		Name                    string         `gorm:"column:name"`
		ClientID                sql.NullString `gorm:"column:client_id"`
		RedirectURIs            pq.StringArray `gorm:"column:redirect_uris"`
		AllowedScopes           pq.StringArray `gorm:"column:allowed_scopes"`
		TokenEndpointAuthMethod sql.NullString `gorm:"column:token_endpoint_auth_method"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT projects.id, projects.slug, projects.name, oidc_clients.client_id, oidc_clients.redirect_uris, oidc_clients.allowed_scopes, oidc_clients.token_endpoint_auth_method FROM projects LEFT JOIN oidc_clients ON oidc_clients.project_id = projects.id AND oidc_clients.deleted_at IS NULL WHERE projects.tenant_id = ? AND projects.deleted_at IS NULL ORDER BY projects.slug`, tenant.ID, &rows, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	items := []map[string]any{}
	for _, row := range rows {
		item := map[string]any{"id": row.ID, "slug": row.Slug, "name": row.Name, "issuer_url": "https://" + tenant.Slug + "." + s.installHost}
		if row.ClientID.Valid {
			item["client_id"] = row.ClientID.String
			item["redirect_uris"] = []string(row.RedirectURIs)
			item["allowed_scopes"] = []string(row.AllowedScopes)
			item["token_endpoint_auth_method"] = row.TokenEndpointAuthMethod.String
		}
		items = append(items, item)
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
	clientID := "client_" + tenant.Slug + "_" + payload.Slug
	secret, err := randomClientSecret()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "project.create_failed")
		return
	}
	encryptedSecret, err := oidc.EncryptClientSecret(secret, s.MasterKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "project.create_failed")
		return
	}
	err = s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO projects (id, tenant_id, slug, name) VALUES (?, ?, ?, ?)`, id, tenant.ID, payload.Slug, payload.Name).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO oidc_clients (tenant_id, project_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method) VALUES (?, ?, ?, ?, ?, ?, ?)`, tenant.ID, id, clientID, encryptedSecret, pq.Array([]string{}), pq.Array([]string{"openid", "email", "profile"}), "client_secret_basic").Error
	})
	if err != nil {
		writeError(w, http.StatusConflict, "project.conflict")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "project.create", ResourceKind: "project", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "slug": payload.Slug, "name": payload.Name, "client_id": clientID, "client_secret": secret, "redirect_uris": []string{}, "allowed_scopes": []string{"openid", "email", "profile"}, "token_endpoint_auth_method": "client_secret_basic"})
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	tenant, projectID, ok := s.projectRouteSubject(w, r)
	if !ok {
		return
	}
	var payload projectPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE projects SET name = COALESCE(NULLIF(?, ''), name), updated_at = now() WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL`, payload.Name, tenant.ID, projectID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE oidc_clients SET redirect_uris = ?, allowed_scopes = ?, token_endpoint_auth_method = ? WHERE tenant_id = ? AND project_id = ? AND deleted_at IS NULL`, pq.Array(payload.RedirectURIs), pq.Array(payload.AllowedScopes), payload.TokenEndpointAuthMethod, tenant.ID, projectID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "project.update_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "project.update", ResourceKind: "project", ResourceID: &projectID, StateAfter: payload})
	writeJSON(w, http.StatusOK, map[string]any{"id": projectID, "name": payload.Name, "redirect_uris": payload.RedirectURIs, "allowed_scopes": payload.AllowedScopes, "token_endpoint_auth_method": payload.TokenEndpointAuthMethod})
}

func (s *Server) rotateProjectSecret(w http.ResponseWriter, r *http.Request) {
	tenant, projectID, ok := s.projectRouteSubject(w, r)
	if !ok {
		return
	}
	secret, err := randomClientSecret()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "project.secret_rotate_failed")
		return
	}
	encryptedSecret, err := oidc.EncryptClientSecret(secret, s.MasterKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "project.secret_rotate_failed")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`UPDATE oidc_clients SET client_secret_encrypted = ? WHERE tenant_id = ? AND project_id = ? AND deleted_at IS NULL`, encryptedSecret, tenant.ID, projectID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "project.secret_rotate_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "project.secret_rotate", ResourceKind: "project", ResourceID: &projectID})
	writeJSON(w, http.StatusOK, map[string]any{"id": projectID, "client_secret": secret, "rotation_in_progress": true})
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	tenant, projectID, ok := s.projectRouteSubject(w, r)
	if !ok {
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE oidc_clients SET deleted_at = now() WHERE tenant_id = ? AND project_id = ? AND deleted_at IS NULL`, tenant.ID, projectID).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE projects SET deleted_at = now() WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL`, tenant.ID, projectID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "project.delete_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "project.delete", ResourceKind: "project", ResourceID: &projectID})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) projectRouteSubject(w http.ResponseWriter, r *http.Request) (Tenant, uuid.UUID, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return Tenant{}, uuid.Nil, false
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "project.id_invalid")
		return Tenant{}, uuid.Nil, false
	}
	return tenant, projectID, true
}

func randomClientSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "cypra_secret_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	limit, offset, ok := parsePagination(w, r, 50, 200)
	if !ok {
		return
	}
	var rows []struct {
		ID       uuid.UUID       `gorm:"column:id"`
		Email    string          `gorm:"column:email"`
		Metadata json.RawMessage `gorm:"column:metadata"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT id, email, metadata FROM users WHERE tenant_id = ? AND deleted_at IS NULL ORDER BY email LIMIT ? OFFSET ?`, tenant.ID, &rows, tenant.ID, limit, offset); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	items := []map[string]any{}
	for _, row := range rows {
		items = append(items, map[string]any{"id": row.ID, "email": row.Email, "metadata": row.Metadata})
	}
	writeJSON(w, http.StatusOK, items)
}

func parsePagination(w http.ResponseWriter, r *http.Request, defaultLimit, maxLimit int) (int, int, bool) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxLimit {
			writeError(w, http.StatusBadRequest, "pagination.limit_invalid")
			return 0, 0, false
		}
		limit = parsed
	}
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "pagination.page_invalid")
			return 0, 0, false
		}
		page = parsed
	}
	return limit, (page - 1) * limit, true
}

func (s *Server) getUserDetail(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	var userRows []struct {
		ID       uuid.UUID       `gorm:"column:id"`
		Email    string          `gorm:"column:email"`
		Metadata json.RawMessage `gorm:"column:metadata"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT id, email, metadata FROM users WHERE tenant_id = ? AND id = ? AND deleted_at IS NULL`, tenant.ID, &userRows, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if len(userRows) == 0 {
		writeError(w, http.StatusNotFound, "user.not_found")
		return
	}
	methods, err := s.userEnrolledMethods(r.Context(), tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	sessions, err := s.userSessionRows(r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	consents, err := s.userConsentRows(r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	auditRows, err := s.userAuditRows(r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":               userRows[0].ID,
			"email":            userRows[0].Email,
			"metadata":         userRows[0].Metadata,
			"sub":              userRows[0].ID,
			"state":            "active",
			"enrolled_methods": methods,
		},
		"auth_methods": methods,
		"sessions":     sessions,
		"consents":     consents,
		"audit":        auditRows,
		"metadata":     userRows[0].Metadata,
	})
}

func (s *Server) userEnrolledMethods(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	checks := []struct {
		name string
		stmt string
	}{
		{"password", `SELECT EXISTS (SELECT 1 FROM password_credentials WHERE tenant_id = $1 AND user_id = $2)`},
		{"passkey", `SELECT EXISTS (SELECT 1 FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2 AND purpose = 'primary')`},
		{"totp", `SELECT EXISTS (SELECT 1 FROM totp_credentials WHERE tenant_id = $1 AND user_id = $2 AND confirmed_at IS NOT NULL)`},
		{"webauthn2fa", `SELECT EXISTS (SELECT 1 FROM passkey_credentials WHERE tenant_id = $1 AND user_id = $2 AND purpose = 'second_factor')`},
		{"google", `SELECT EXISTS (SELECT 1 FROM users WHERE tenant_id = $1 AND id = $2 AND metadata ? 'google_sub')`},
	}
	methods := []string{}
	for _, check := range checks {
		var exists bool
		if err := s.DB.QueryRowContext(ctx, check.stmt, tenantID, userID).Scan(&exists); err != nil {
			return nil, err
		}
		if exists {
			methods = append(methods, check.name)
		}
	}
	return methods, nil
}

func (s *Server) userSessionRows(r *http.Request, tenantID, userID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id::text, created_at, last_seen_at, expires_at, revoked_at, COALESCE(ip::text, ''), COALESCE(user_agent, '') FROM sessions WHERE tenant_id = $1 AND subject_id = $2 AND subject_kind = 'user' ORDER BY last_seen_at DESC LIMIT 20`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id, ip, ua string
		var created, lastSeen, expires time.Time
		var revoked sql.NullTime
		if err := rows.Scan(&id, &created, &lastSeen, &expires, &revoked, &ip, &ua); err != nil {
			return nil, err
		}
		item := map[string]any{"id": id, "created_at": created.Format(time.RFC3339), "last_seen_at": lastSeen.Format(time.RFC3339), "expires_at": expires.Format(time.RFC3339), "ip": ip, "user_agent": ua, "revoked": revoked.Valid}
		if revoked.Valid {
			item["revoked_at"] = revoked.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) userConsentRows(r *http.Request, tenantID, userID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT c.id::text, oc.client_id, c.scopes, c.granted_at, c.revoked_at FROM oidc_consents c JOIN oidc_clients oc ON oc.id = c.oidc_client_uuid WHERE c.tenant_id = $1 AND c.user_id = $2 ORDER BY c.granted_at DESC LIMIT 50`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id, clientID string
		var scopes pq.StringArray
		var granted time.Time
		var revoked sql.NullTime
		if err := rows.Scan(&id, &clientID, &scopes, &granted, &revoked); err != nil {
			return nil, err
		}
		item := map[string]any{"id": id, "client_id": clientID, "scopes": []string(scopes), "granted_at": granted.Format(time.RFC3339), "revoked": revoked.Valid}
		if revoked.Valid {
			item["revoked_at"] = revoked.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) userAuditRows(r *http.Request, tenantID, userID uuid.UUID) ([]map[string]any, error) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id::text, occurred_at, action, resource_kind, COALESCE(state_after, '{}'::jsonb), redacted_at FROM audit_entries WHERE tenant_id = $1 AND resource_id = $2 ORDER BY occurred_at DESC LIMIT 25`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id, action, resourceKind string
		var occurred time.Time
		var stateAfter json.RawMessage
		var redacted sql.NullTime
		if err := rows.Scan(&id, &occurred, &action, &resourceKind, &stateAfter, &redacted); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{"id": id, "occurred_at": occurred.Format(time.RFC3339), "action": action, "resource_kind": resourceKind, "state_after": stateAfter, "redacted": redacted.Valid})
	}
	return items, rows.Err()
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
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "user.create", ResourceKind: "user", ResourceID: &id, StateAfter: payload})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "email": payload.Email, "metadata": payload.Metadata})
}
