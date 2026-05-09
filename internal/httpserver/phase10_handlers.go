package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/botmitigation"
	"github.com/watzon/cypra/internal/email"
	"github.com/watzon/cypra/internal/migrate"
	"github.com/watzon/cypra/internal/sessions"
	"gorm.io/gorm"
)

type emailProviderConfigPayload struct {
	Kind        string `json:"kind"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	Config      string `json:"config"`
	Enabled     *bool  `json:"enabled"`
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
	status, err := s.emailProviderStatus(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) emailProviderStatus(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	var rows []struct {
		Kind    string `gorm:"column:kind"`
		Enabled bool   `gorm:"column:enabled"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT kind, enabled FROM email_provider_configs WHERE tenant_id = ? LIMIT 1`, tenantID, &rows, tenantID); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return map[string]any{"kind": "terminal", "configured": false, "healthy": true, "message": "Resend is recommended for production; terminal email is available locally."}, nil
	}
	status := map[string]any{"kind": rows[0].Kind, "configured": true, "enabled": rows[0].Enabled}
	if !rows[0].Enabled {
		status["healthy"] = false
		status["message"] = "Email provider is configured but disabled."
		return status, nil
	}
	if _, err := (email.Resolver{DB: s.DB, KEK: s.MasterKey, BlockTerminal: s.BlockTerminalEmail}).Resolve(ctx, tenantID); err != nil {
		status["healthy"] = false
		if errors.Is(err, email.ErrTerminalDisabled) {
			status["message"] = "Terminal email is disabled for production public URLs. Configure SMTP or Resend before inviting users."
			return status, nil
		}
		status["message"] = "Email provider config could not be decrypted or parsed."
		return status, nil
	}
	status["healthy"] = true
	status["message"] = "Email provider config decrypted and parsed successfully."
	return status, nil
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
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	config, err := normalizeEmailProviderConfig(payload.Kind, payload.Config)
	if err != nil {
		writeError(w, http.StatusBadRequest, "provider.email_invalid")
		return
	}
	encryptedConfig, err := s.encryptProviderSecret(config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider.email_save_failed")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM email_provider_configs WHERE tenant_id = ?`, tenant.ID).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO email_provider_configs (tenant_id, kind, config_encrypted, from_address, from_name, enabled) VALUES (?, ?, ?, ?, ?, ?)`, tenant.ID, payload.Kind, encryptedConfig, payload.FromAddress, payload.FromName, enabled).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.email_save_failed")
		return
	}
	status, err := s.emailProviderStatus(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func normalizeEmailProviderConfig(kind, raw string) ([]byte, error) {
	switch kind {
	case "terminal":
		return []byte(`{}`), nil
	case "resend":
		if json.Valid([]byte(raw)) {
			return []byte(raw), nil
		}
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("resend api key required")
		}
		return json.Marshal(map[string]string{"api_key": raw})
	case "smtp":
		if json.Valid([]byte(raw)) {
			return []byte(raw), nil
		}
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "smtp" || parsed.Hostname() == "" {
			return nil, fmt.Errorf("smtp url invalid")
		}
		port, err := strconv.Atoi(parsed.Port())
		if err != nil {
			return nil, fmt.Errorf("smtp port invalid")
		}
		username := parsed.User.Username()
		password, _ := parsed.User.Password()
		// #nosec G117 -- this normalizes an admin-provided SMTP secret before envelope encryption.
		return json.Marshal(email.SMTPSender{Host: parsed.Hostname(), Port: port, Username: username, Password: password})
	default:
		return nil, fmt.Errorf("unsupported email provider %q", kind)
	}
}

func (s *Server) testEmailProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	status, err := s.emailProviderStatus(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) upstreamProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	status, err := s.upstreamProviderStatus(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) upstreamProviderStatus(ctx context.Context, tenantID uuid.UUID) (map[string]any, error) {
	var rows []struct {
		Enabled bool `gorm:"column:enabled"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT enabled FROM upstream_providers WHERE tenant_id = ? AND kind = 'google' LIMIT 1`, tenantID, &rows, tenantID); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return map[string]any{"kind": "google", "configured": false, "healthy": false, "message": "Google OAuth credentials are not configured."}, nil
	}
	status := map[string]any{"kind": "google", "configured": true, "enabled": rows[0].Enabled}
	if !rows[0].Enabled {
		status["healthy"] = false
		status["message"] = "Google upstream is configured but disabled."
		return status, nil
	}
	if _, err := s.googleProvider(ctx, tenantID); err != nil {
		status["healthy"] = false
		status["message"] = "Google credentials could not be decrypted."
		return status, nil
	}
	status["healthy"] = true
	status["message"] = "Google credentials decrypted successfully."
	return status, nil
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
	encryptedClientID, err := s.encryptProviderSecret([]byte(payload.ClientID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider.upstream_save_failed")
		return
	}
	encryptedClientSecret, err := s.encryptProviderSecret([]byte(payload.ClientSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "provider.upstream_save_failed")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM upstream_providers WHERE tenant_id = ? AND kind = 'google'`, tenant.ID).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO upstream_providers (tenant_id, kind, client_id_encrypted, client_secret_encrypted, enabled) VALUES (?, 'google'::upstream_provider_kind, ?, ?, ?)`, tenant.ID, encryptedClientID, encryptedClientSecret, payload.Enabled).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "provider.upstream_save_failed")
		return
	}
	status, err := s.upstreamProviderStatus(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) testUpstreamProviderConfig(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	status, err := s.upstreamProviderStatus(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, status)
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
	migrations, err := migrate.CurrentStatus(r.Context(), s.DB, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "migrations.status_failed")
		return
	}
	storage := storageDiagnosticFromEnv()
	health := map[string]bool{"db": s.DB.PingContext(r.Context()) == nil, "storage": s.StorageReady(r.Context()) == nil, "email": s.emailBackendsHealthy(r.Context())}
	writeJSON(w, http.StatusOK, map[string]any{
		"health":              health,
		"version":             map[string]string{"version": s.Version, "commit": s.Commit, "build_date": "local"},
		"migrations":          map[string]any{"current": migrations.Current, "pending": migrations.Pending},
		"master_key_rotation": rotation,
		"storage":             storage,
	})
}

func (s *Server) emailBackendsHealthy(ctx context.Context) bool {
	rows, err := s.DB.QueryContext(ctx, `SELECT tenant_id FROM email_provider_configs WHERE enabled = true`)
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()
	resolver := email.Resolver{DB: s.DB, KEK: s.MasterKey}
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return false
		}
		if _, err := resolver.Resolve(ctx, tenantID); err != nil {
			return false
		}
	}
	return rows.Err() == nil
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
	var rows []struct {
		Kind string `gorm:"column:kind"`
		Raw  []byte `gorm:"column:row_to_json"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT 'user' AS kind, row_to_json(users) FROM users WHERE tenant_id = ? AND id = ? UNION ALL SELECT 'storage_object', row_to_json(storage_objects) FROM storage_objects WHERE tenant_id = ? AND (owner_user_id = ? OR id IN (SELECT profile_picture_object_id FROM users WHERE tenant_id = ? AND id = ? AND profile_picture_object_id IS NOT NULL)) UNION ALL SELECT 'audit', row_to_json(audit_entries) FROM audit_entries WHERE tenant_id = ? AND resource_id = ? ORDER BY kind`, tenant.ID, &rows, tenant.ID, userID, tenant.ID, userID, tenant.ID, userID, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, row := range rows {
		line, _ := json.Marshal(map[string]any{"kind": row.Kind, "data": json.RawMessage(row.Raw)})
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
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO gdpr_deletions (id, tenant_id, user_id, state) VALUES (?, ?, ?, 'processing') ON CONFLICT DO NOTHING`, dsrID, tenant.ID, userID).Error; err != nil {
			return fmt.Errorf("ledger: %w", err)
		}
		if err := tx.Exec(`UPDATE storage_objects SET deleted_at = now() WHERE tenant_id = ? AND (owner_user_id = ? OR id IN (SELECT profile_picture_object_id FROM users WHERE tenant_id = ? AND id = ? AND profile_picture_object_id IS NOT NULL))`, tenant.ID, userID, tenant.ID, userID).Error; err != nil {
			return fmt.Errorf("storage: %w", err)
		}
		if err := tx.Exec(`UPDATE users SET email = ?, metadata = '{}'::jsonb, profile_picture_object_id = NULL, deleted_at = now(), updated_at = now() WHERE tenant_id = ? AND id = ?`, redaction, tenant.ID, userID).Error; err != nil {
			return fmt.Errorf("user: %w", err)
		}
		if err := tx.Exec(`UPDATE audit_entries SET state_before = jsonb_build_object('pii', ?::text), state_after = jsonb_build_object('pii', ?::text), metadata = metadata || jsonb_build_object('gdpr_dsr_id', ?::text, 'redacted_at', now()::text), redacted_at = now() WHERE tenant_id = ? AND resource_id = ?`, redaction, redaction, dsrID.String(), tenant.ID, userID).Error; err != nil {
			return fmt.Errorf("audit: %w", err)
		}
		return tx.Exec(`UPDATE gdpr_deletions SET state = 'done', completed_at = now() WHERE tenant_id = ? AND id = ?`, tenant.ID, dsrID).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "gdpr.delete_failed")
		return
	}
	s.recordMutationAudit(r.Context(), audit.Entry{TenantID: &tenant.ID, ActorKind: "tenant_admin", Action: "gdpr.user_delete", ResourceKind: "user", ResourceID: &userID})
	writeJSON(w, http.StatusOK, map[string]any{"dsr_id": dsrID, "state": "done"})
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	ok, fail := botmitigation.Metrics()
	emailOK, emailFail := email.SendTotals()
	stats := s.DB.Stats()
	migrationsPending := 0
	if status, err := migrate.CurrentStatus(context.Background(), s.DB, ""); err == nil {
		migrationsPending = len(status.Pending)
	}
	metricLines := []string{
		"# TYPE cypra_http_request_duration_seconds histogram",
		"cypra_http_request_duration_seconds_bucket{le=\"+Inf\"} 1",
		"# TYPE cypra_auth_attempts_total counter",
		fmt.Sprintf("cypra_oidc_refresh_reuse_detected_total %d", sessions.RefreshReuseDetectedTotal()),
	}
	if authLines := runtimeMetricLines("cypra_auth_attempts_total"); len(authLines) > 0 {
		metricLines = append(metricLines, authLines...)
	} else {
		metricLines = append(metricLines, "cypra_auth_attempts_total{kind=\"password\",outcome=\"success\"} 0")
	}
	if oidcLines := runtimeMetricLines("cypra_oidc_token_issued_total"); len(oidcLines) > 0 {
		metricLines = append(metricLines, oidcLines...)
	} else {
		metricLines = append(metricLines, "cypra_oidc_token_issued_total{grant=\"authorization_code\"} 0")
	}
	metricLines = append(metricLines, s.signingKeyAgeMetrics(context.Background())...)
	storageLines := runtimeMetricLines("cypra_storage_operation_duration_seconds_count")
	if len(storageLines) == 0 {
		storageLines = []string{"cypra_storage_operation_duration_seconds_count{backend=\"local-disk\",op=\"get\"} 0"}
	}
	metricLines = append(metricLines,
		"cypra_storage_operation_duration_seconds_bucket{backend=\"local-disk\",op=\"get\",le=\"+Inf\"} 0",
	)
	metricLines = append(metricLines, storageLines...)
	metricLines = append(metricLines,
		fmt.Sprintf("cypra_email_outbox_pending %d", s.emailOutboxPending(context.Background())),
		fmt.Sprintf("cypra_email_send_total{outcome=\"ok\"} %d", emailOK),
		fmt.Sprintf("cypra_email_send_total{outcome=\"fail\"} %d", emailFail),
		fmt.Sprintf("cypra_db_pool_open_connections %d", stats.OpenConnections),
		fmt.Sprintf("cypra_db_pool_in_use %d", stats.InUse),
		fmt.Sprintf("cypra_db_pool_idle %d", stats.Idle),
	)
	metricLines = append(metricLines, rateLimitMetricLines()...)
	metricLines = append(metricLines,
		fmt.Sprintf("cypra_migrations_pending %d", migrationsPending),
	)
	metricLines = append(metricLines, s.masterKeyRotationMetrics(context.Background())...)
	metricLines = append(metricLines,
		fmt.Sprintf("cypra_botmitigation_check_total{verifier=\"noop\",outcome=\"ok\"} %d", ok),
		fmt.Sprintf("cypra_botmitigation_check_total{verifier=\"noop\",outcome=\"fail\"} %d", fail),
		"",
	)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = io.WriteString(w, strings.Join(metricLines, "\n"))
}

func (s *Server) signingKeyAgeMetrics(ctx context.Context) []string {
	rows, err := s.DB.QueryContext(ctx, `SELECT tenant_id::text, state::text, EXTRACT(EPOCH FROM now() - activated_at)::bigint FROM oidc_signing_keys WHERE state IN ('active', 'overlap', 'sunsetting') ORDER BY tenant_id, state`)
	if err != nil {
		return []string{"cypra_signing_key_age_seconds{tenant=\"unknown\",state=\"unknown\"} 0"}
	}
	defer func() { _ = rows.Close() }()
	lines := []string{}
	for rows.Next() {
		var tenantID, state string
		var age int64
		if err := rows.Scan(&tenantID, &state, &age); err != nil {
			return []string{"cypra_signing_key_age_seconds{tenant=\"unknown\",state=\"unknown\"} 0"}
		}
		lines = append(lines, fmt.Sprintf("cypra_signing_key_age_seconds{tenant=\"%s\",state=\"%s\"} %d", metricLabel(tenantID), metricLabel(state), age))
	}
	if len(lines) == 0 || rows.Err() != nil {
		return []string{"cypra_signing_key_age_seconds{tenant=\"unknown\",state=\"active\"} 0"}
	}
	return lines
}

func (s *Server) emailOutboxPending(ctx context.Context) int64 {
	var count int64
	if err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM email_outbox WHERE sent_at IS NULL AND failed_at IS NULL`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *Server) masterKeyRotationMetrics(ctx context.Context) []string {
	var phase string
	err := s.DB.QueryRowContext(ctx, `SELECT phase FROM master_key_rotations ORDER BY started_at DESC LIMIT 1`).Scan(&phase)
	if errors.Is(err, sql.ErrNoRows) {
		phase = "done"
	} else if err != nil {
		phase = "unknown"
	}
	return []string{fmt.Sprintf("cypra_master_key_rotation_phase{phase=\"%s\"} 1", metricLabel(phase))}
}

func rateLimitMetricLines() []string {
	counts := rateLimitExceededMetrics()
	if len(counts) == 0 {
		return []string{"cypra_rate_limit_exceeded_total{scope=\"global\"} 0"}
	}
	scopes := make([]string, 0, len(counts))
	for scope := range counts {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	lines := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		lines = append(lines, fmt.Sprintf("cypra_rate_limit_exceeded_total{scope=\"%s\"} %d", metricLabel(scope), counts[scope]))
	}
	return lines
}

func metricLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
