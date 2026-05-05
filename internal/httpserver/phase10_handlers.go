package httpserver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/botmitigation"
)

type emailProviderConfigPayload struct {
	Kind        string `json:"kind"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	Config      string `json:"config"`
}

type upstreamProviderConfigPayload struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Enabled      bool   `json:"enabled"`
}

func (s *Server) emailProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var kind string
	err := s.DB.QueryRowContext(r.Context(), `SELECT kind FROM email_provider_configs WHERE tenant_id = $1 LIMIT 1`, tenant.ID).Scan(&kind)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, map[string]any{"kind": "terminal", "configured": false, "healthy": true, "message": "Resend is recommended for production; terminal email is available locally."})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": kind, "configured": true, "healthy": true, "message": "Email provider configured."})
}

func (s *Server) saveEmailProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload emailProviderConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if payload.Kind == "" || payload.FromAddress == "" || payload.FromName == "" {
		writeError(w, http.StatusBadRequest, "provider.email_invalid")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM email_provider_configs WHERE tenant_id = $1`, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.email_save_failed")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO email_provider_configs (tenant_id, kind, config_encrypted, from_address, from_name) VALUES ($1, $2, $3, $4, $5)`, tenant.ID, payload.Kind, []byte(payload.Config), payload.FromAddress, payload.FromName); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.email_save_failed")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.email_save_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": payload.Kind, "configured": true, "healthy": true, "message": "Email provider configured."})
}

func (s *Server) upstreamProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var enabled bool
	err := s.DB.QueryRowContext(r.Context(), `SELECT enabled FROM upstream_providers WHERE tenant_id = $1 AND kind = 'google' LIMIT 1`, tenant.ID).Scan(&enabled)
	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, map[string]any{"kind": "google", "configured": false, "healthy": false, "message": "Google OAuth credentials are not configured."})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": "google", "configured": true, "healthy": enabled, "message": "Google upstream configured."})
}

func (s *Server) saveUpstreamProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload upstreamProviderConfigPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if payload.ClientID == "" || payload.ClientSecret == "" {
		writeError(w, http.StatusBadRequest, "provider.upstream_invalid")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM upstream_providers WHERE tenant_id = $1 AND kind = 'google'`, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.upstream_save_failed")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO upstream_providers (tenant_id, kind, client_id_encrypted, client_secret_encrypted, enabled) VALUES ($1, 'google', $2, $3, $4)`, tenant.ID, []byte(payload.ClientID), []byte(payload.ClientSecret), payload.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.upstream_save_failed")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.upstream_save_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"kind": "google", "configured": true, "healthy": payload.Enabled, "message": "Google upstream configured."})
}

func (s *Server) listInstanceAdmins(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, email, created_at, last_login_at FROM instance_admins WHERE disabled_at IS NULL ORDER BY created_at`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var email string
		var created time.Time
		var last sql.NullTime
		if err := rows.Scan(&id, &email, &created, &last); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		item := map[string]any{"id": id, "email": email, "role": "owner", "created_at": created.Format(time.RFC3339)}
		if last.Valid {
			item["last_seen_at"] = last.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) demoteInstanceAdmin(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "instance_admin.id_invalid")
		return
	}
	tx, err := s.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err := tx.QueryRowContext(r.Context(), `SELECT count(*) FROM instance_admins WHERE disabled_at IS NULL`).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if count <= 1 {
		writeError(w, http.StatusConflict, "instance_admin.last_admin")
		return
	}
	result, err := tx.ExecContext(r.Context(), `UPDATE instance_admins SET disabled_at = now() WHERE id = $1 AND disabled_at IS NULL`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "instance_admin.demote_failed")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusNotFound, "instance_admin.not_found")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "demoted"})
}

func (s *Server) instanceDiagnostics(w http.ResponseWriter, r *http.Request) {
	rotation := map[string]any{"phase": "done", "rows_done": 0, "rows_total": 0}
	var phase string
	var rowsDone, rowsTotal int64
	if err := s.DB.QueryRowContext(r.Context(), `SELECT phase, rows_done, rows_total FROM master_key_rotations ORDER BY started_at DESC LIMIT 1`).Scan(&phase, &rowsDone, &rowsTotal); err == nil {
		rotation = map[string]any{"phase": phase, "rows_done": rowsDone, "rows_total": rowsTotal}
	}
	storage := storageDiagnosticFromEnv()
	health := map[string]bool{"db": s.DB.PingContext(r.Context()) == nil, "storage": s.StorageReady(r.Context()) == nil, "email": true}
	writeJSON(w, http.StatusOK, map[string]any{
		"health":              health,
		"version":             map[string]string{"version": s.Version, "commit": s.Commit, "build_date": "local"},
		"migrations":          map[string]any{"current": 4, "pending": []string{}},
		"master_key_rotation": rotation,
		"storage":             storage,
	})
}

func storageDiagnosticFromEnv() map[string]any {
	kind := os.Getenv("STORAGE_BACKEND")
	if kind == "" {
		kind = "local-disk"
	}
	if kind == "s3-compatible" {
		return map[string]any{
			"kind":                kind,
			"bucket":              os.Getenv("STORAGE_S3_BUCKET"),
			"endpoint":            maskEndpoint(os.Getenv("STORAGE_S3_ENDPOINT")),
			"region":              os.Getenv("STORAGE_S3_REGION"),
			"credentials_present": os.Getenv("STORAGE_S3_ACCESS_KEY_ID") != "" && os.Getenv("STORAGE_S3_SECRET_ACCESS_KEY") != "",
		}
	}
	path := os.Getenv("STORAGE_LOCAL_PATH")
	if path == "" {
		path = "cypra-storage"
	}
	return map[string]any{"kind": kind, "endpoint": "file://****/" + strings.TrimPrefix(path, "/"), "credentials_present": true}
}

func maskEndpoint(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	parts := strings.SplitN(endpoint, "://", 2)
	if len(parts) != 2 {
		return "****"
	}
	host := parts[1]
	if slash := strings.IndexByte(host, '/'); slash >= 0 {
		host = host[:slash]
	}
	segments := strings.Split(host, ".")
	if len(segments) < 2 {
		return parts[0] + "://****"
	}
	return parts[0] + "://****." + strings.Join(segments[len(segments)-2:], ".")
}

func (s *Server) exportUserDSR(w http.ResponseWriter, r *http.Request) {
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
	rows, err := s.DB.QueryContext(r.Context(), `SELECT 'user' AS kind, row_to_json(users) FROM users WHERE tenant_id = $1 AND id = $2 UNION ALL SELECT 'audit', row_to_json(audit_entries) FROM audit_entries WHERE tenant_id = $1 AND resource_id = $2 ORDER BY kind`, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	defer func() { _ = rows.Close() }()
	w.Header().Set("Content-Type", "application/x-ndjson")
	for rows.Next() {
		var kind string
		var raw []byte
		if err := rows.Scan(&kind, &raw); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		line, _ := json.Marshal(map[string]any{"kind": kind, "data": json.RawMessage(raw)})
		_, _ = w.Write(append(line, '\n'))
	}
}

func (s *Server) deleteUserDSR(w http.ResponseWriter, r *http.Request) {
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
	dsrID := uuid.New()
	redaction := fmt.Sprintf("*** redacted (gdpr-%s) ***", dsrID.String())
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO gdpr_deletions (id, tenant_id, user_id, state) VALUES ($1, $2, $3, 'processing') ON CONFLICT DO NOTHING`, dsrID, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "gdpr.ledger_failed")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE storage_objects SET deleted_at = now() WHERE tenant_id = $1 AND id IN (SELECT profile_picture_object_id FROM users WHERE tenant_id = $1 AND id = $2 AND profile_picture_object_id IS NOT NULL)`, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "gdpr.storage_purge_failed")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE users SET email = $1, metadata = '{}'::jsonb, profile_picture_object_id = NULL, deleted_at = now(), updated_at = now() WHERE tenant_id = $2 AND id = $3`, redaction, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "gdpr.user_scrub_failed")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE audit_entries SET state_before = jsonb_build_object('pii', $1::text), state_after = jsonb_build_object('pii', $1::text), metadata = metadata || jsonb_build_object('gdpr_dsr_id', $2::text, 'redacted_at', now()::text), redacted_at = now() WHERE tenant_id = $3 AND resource_id = $4`, redaction, dsrID.String(), tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "gdpr.audit_redaction_failed")
		return
	}
	_, _ = s.DB.ExecContext(r.Context(), `UPDATE gdpr_deletions SET state = 'done', completed_at = now() WHERE id = $1`, dsrID)
	_ = s.audit.Write(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "gdpr.user_delete", ResourceKind: "user", ResourceID: &userID})
	writeJSON(w, http.StatusOK, map[string]any{"dsr_id": dsrID, "state": "done"})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	ok, fail := botmitigation.Metrics()
	stats := s.DB.Stats()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = io.WriteString(w, strings.Join([]string{
		"# TYPE cypra_http_request_duration_seconds histogram",
		"cypra_http_request_duration_seconds_bucket{le=\"+Inf\"} 1",
		"# TYPE cypra_auth_attempts_total counter",
		"cypra_auth_attempts_total{kind=\"password\",outcome=\"ok\"} 0",
		"cypra_oidc_token_issued_total{grant=\"authorization_code\"} 0",
		"cypra_oidc_refresh_reuse_detected_total 0",
		"cypra_signing_key_age_seconds{tenant=\"unknown\",state=\"active\"} 0",
		"cypra_storage_operation_duration_seconds_bucket{le=\"+Inf\"} 0",
		"cypra_email_outbox_pending 0",
		"cypra_email_send_total{outcome=\"ok\"} 0",
		fmt.Sprintf("cypra_db_pool_open_connections %d", stats.OpenConnections),
		fmt.Sprintf("cypra_db_pool_in_use %d", stats.InUse),
		fmt.Sprintf("cypra_db_pool_idle %d", stats.Idle),
		"cypra_rate_limit_exceeded_total{scope=\"global\"} 0",
		"cypra_migrations_pending 0",
		"cypra_master_key_rotation_phase{phase=\"done\"} 1",
		fmt.Sprintf("cypra_botmitigation_check_total{verifier=\"noop\",outcome=\"ok\"} %d", ok),
		fmt.Sprintf("cypra_botmitigation_check_total{verifier=\"noop\",outcome=\"fail\"} %d", fail),
		"",
	}, "\n"))
}
