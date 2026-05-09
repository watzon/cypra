package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/authpolicy"
	"gorm.io/gorm"
)

var tenantAuthMethods = []string{
	authpolicy.MethodPassword,
	authpolicy.MethodMagicLink,
	authpolicy.MethodPasskey,
	authpolicy.MethodTOTP,
	authpolicy.MethodGoogle,
	authpolicy.MethodOIDCUpstream,
}

type authProviderResponse struct {
	Method        string          `json:"method"`
	Enabled       bool            `json:"enabled"`
	EnrolledCount int64           `json:"enrolled_count"`
	Config        json.RawMessage `json:"config"`
}

type authProviderRow struct {
	Method        string `gorm:"column:method"`
	Enabled       bool   `gorm:"column:enabled"`
	EnrolledCount int64  `gorm:"column:enrolled_count"`
	Config        []byte `gorm:"column:config"`
}

type authProviderPayload struct {
	Enabled bool            `json:"enabled"`
	Config  json.RawMessage `json:"config"`
}

func (s *Server) listAuthProviders(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var rows []authProviderRow
	if err := s.TenantDB.RawScan(r.Context(), `SELECT method::text AS method, enabled, enrolled_count_cache AS enrolled_count, config FROM tenant_auth_methods WHERE tenant_id = ? ORDER BY method`, tenant.ID, &rows, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.list_failed")
		return
	}
	byMethod := map[string]authProviderRow{}
	for _, row := range rows {
		byMethod[row.Method] = row
	}
	items := make([]authProviderResponse, 0, len(tenantAuthMethods))
	for _, method := range tenantAuthMethods {
		row, exists := byMethod[method]
		cfg := row.Config
		if len(cfg) == 0 {
			cfg = []byte("{}")
		}
		item := authProviderResponse{
			Method:        method,
			Enabled:       row.Enabled,
			EnrolledCount: row.EnrolledCount,
			Config:        cfg,
		}
		_ = exists // method existence already encoded by row.Enabled defaulting to false
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getAuthProvider(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	method := chi.URLParam(r, "method")
	if !knownAuthMethod(method) {
		writeError(w, http.StatusBadRequest, "auth_providers.method_invalid")
		return
	}
	var rows []authProviderRow
	if err := s.TenantDB.RawScan(r.Context(), `SELECT method::text AS method, enabled, enrolled_count_cache AS enrolled_count, config FROM tenant_auth_methods WHERE tenant_id = ? AND method = ?::tenant_auth_method`, tenant.ID, &rows, tenant.ID, method); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.load_failed")
		return
	}
	if len(rows) == 0 {
		writeJSON(w, http.StatusOK, authProviderResponse{Method: method, Config: json.RawMessage("{}")})
		return
	}
	row := rows[0]
	cfg := row.Config
	if len(cfg) == 0 {
		cfg = []byte("{}")
	}
	writeJSON(w, http.StatusOK, authProviderResponse{Method: row.Method, Enabled: row.Enabled, EnrolledCount: row.EnrolledCount, Config: cfg})
}

func (s *Server) saveAuthProvider(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	method := chi.URLParam(r, "method")
	if !knownAuthMethod(method) {
		writeError(w, http.StatusBadRequest, "auth_providers.method_invalid")
		return
	}
	var payload authProviderPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.payload_invalid")
		return
	}
	if len(payload.Config) == 0 {
		payload.Config = json.RawMessage("{}")
	}
	if err := validateProviderConfig(method, payload.Config); err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.config_invalid")
		return
	}

	if !payload.Enabled {
		var defaultMethod sql.NullString
		if err := s.DB.QueryRowContext(r.Context(), `SELECT default_auth_method::text FROM tenants WHERE id = $1`, tenant.ID).Scan(&defaultMethod); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		if defaultMethod.Valid && defaultMethod.String == method {
			writeError(w, http.StatusBadRequest, "auth_providers.default_disabled")
			return
		}
	}

	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled, config) VALUES (?, ?::tenant_auth_method, ?, ?::jsonb) ON CONFLICT (tenant_id, method) DO UPDATE SET enabled = EXCLUDED.enabled, config = EXCLUDED.config, updated_at = now()`, tenant.ID, method, payload.Enabled, []byte(payload.Config)).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.update_failed")
		return
	}
	writeJSON(w, http.StatusOK, authProviderResponse{Method: method, Enabled: payload.Enabled, Config: payload.Config})
}

type defaultAuthMethodPayload struct {
	Method *string `json:"method"`
}

func (s *Server) setDefaultAuthMethod(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload defaultAuthMethodPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.payload_invalid")
		return
	}
	if payload.Method == nil {
		if _, err := s.DB.ExecContext(r.Context(), `UPDATE tenants SET default_auth_method = NULL WHERE id = $1`, tenant.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return
		}
		writeJSON(w, http.StatusOK, defaultAuthMethodPayload{})
		return
	}
	method := *payload.Method
	enabled, err := s.defaultMethodEnabled(r.Context(), tenant.ID, method)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if !enabled {
		writeError(w, http.StatusBadRequest, "auth_providers.default_not_enabled")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE tenants SET default_auth_method = $1 WHERE id = $2`, method, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, defaultAuthMethodPayload{Method: &method})
}

// defaultMethodEnabled validates a default-method identifier (built-in,
// social:<kind>, or oidc:<slug>) against the tenant's enabled providers.
func (s *Server) defaultMethodEnabled(ctx context.Context, tenantID uuid.UUID, method string) (bool, error) {
	family, sub := authpolicy.SplitMethod(method)
	switch family {
	case "social":
		if sub == "" || !isSocialKind(sub) {
			return false, nil
		}
		var enabled bool
		err := s.DB.QueryRowContext(ctx, `SELECT enabled FROM social_connections WHERE tenant_id = $1 AND kind = $2::social_provider_kind`, tenantID, sub).Scan(&enabled)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return enabled, err
	case "oidc":
		if sub == "" {
			return false, nil
		}
		var enabled bool
		err := s.DB.QueryRowContext(ctx, `SELECT enabled FROM oidc_connections WHERE tenant_id = $1 AND slug = $2`, tenantID, sub).Scan(&enabled)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return enabled, err
	default:
		if !knownAuthMethod(method) {
			return false, nil
		}
		var enabled bool
		err := s.DB.QueryRowContext(ctx, `SELECT enabled FROM tenant_auth_methods WHERE tenant_id = $1 AND method = $2::tenant_auth_method`, tenantID, method).Scan(&enabled)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return enabled, err
	}
}

func (s *Server) getDefaultAuthMethod(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	method, err := s.lookupDefaultAuthMethod(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if method == "" {
		writeJSON(w, http.StatusOK, defaultAuthMethodPayload{})
		return
	}
	writeJSON(w, http.StatusOK, defaultAuthMethodPayload{Method: &method})
}

func (s *Server) lookupDefaultAuthMethod(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var method sql.NullString
	if err := s.DB.QueryRowContext(ctx, `SELECT default_auth_method FROM tenants WHERE id = $1`, tenantID).Scan(&method); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if !method.Valid {
		return "", nil
	}
	return method.String, nil
}

func validateProviderConfig(method string, raw json.RawMessage) error {
	switch method {
	case authpolicy.MethodPassword:
		return decodeStrict(raw, &authpolicy.PasswordPolicy{})
	case authpolicy.MethodMagicLink:
		return decodeStrict(raw, &authpolicy.MagicLinkPolicy{})
	case authpolicy.MethodPasskey:
		return decodeStrict(raw, &authpolicy.PasskeyPolicy{})
	case authpolicy.MethodTOTP:
		return decodeStrict(raw, &authpolicy.TOTPPolicy{})
	case authpolicy.MethodGoogle:
		return decodeStrict(raw, &authpolicy.GooglePolicy{})
	case authpolicy.MethodOIDCUpstream:
		return decodeStrict(raw, &authpolicy.OIDCUpstreamPolicy{})
	default:
		return fmt.Errorf("unknown method %q", method)
	}
}

func decodeStrict(raw json.RawMessage, dest any) error {
	dec := json.NewDecoder(bytes.NewReader([]byte(raw)))
	dec.DisallowUnknownFields()
	return dec.Decode(dest)
}

func knownAuthMethod(method string) bool {
	for _, candidate := range tenantAuthMethods {
		if method == candidate {
			return true
		}
	}
	return false
}
