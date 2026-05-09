package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"

	"github.com/watzon/cypra/internal/auth/upstream"
	"github.com/watzon/cypra/internal/auth/upstream/oidcgeneric"
	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/observability"
)

var oidcSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

type oidcConnectionPayload struct {
	Slug           string   `json:"slug"`
	DisplayName    string   `json:"display_name"`
	IssuerURL      string   `json:"issuer_url"`
	ClientID       string   `json:"client_id"`
	ClientSecret   string   `json:"client_secret"`
	Scopes         []string `json:"scopes"`
	Enabled        bool     `json:"enabled"`
	AllowedDomains []string `json:"allowed_domains"`
	AllowSignup    *bool    `json:"allow_signup,omitempty"`
}

type oidcConnectionRecord struct {
	ID             uuid.UUID  `json:"id"`
	Slug           string     `json:"slug"`
	DisplayName    string     `json:"display_name"`
	IssuerURL      string     `json:"issuer_url"`
	Scopes         []string   `json:"scopes"`
	Enabled        bool       `json:"enabled"`
	ClientIDSet    bool       `json:"client_id_set"`
	AllowedDomains []string   `json:"allowed_domains"`
	AllowSignup    *bool      `json:"allow_signup,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type oidcDBRow struct {
	ID                    uuid.UUID      `gorm:"column:id"`
	Slug                  string         `gorm:"column:slug"`
	DisplayName           string         `gorm:"column:display_name"`
	IssuerURL             string         `gorm:"column:issuer_url"`
	ClientIDEncrypted     []byte         `gorm:"column:client_id_encrypted"`
	ClientSecretEncrypted []byte         `gorm:"column:client_secret_encrypted"`
	Scopes                pq.StringArray `gorm:"column:scopes"`
	Config                []byte         `gorm:"column:config"`
	Enabled               bool           `gorm:"column:enabled"`
	UpdatedAt             time.Time      `gorm:"column:updated_at"`
}

func (s *Server) listOIDCConnections(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	rows, err := s.loadOIDCConnections(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	out := make([]oidcConnectionRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, oidcRowToRecord(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"connections": out})
}

func (s *Server) createOIDCConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload oidcConnectionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(payload.Slug))
	if !oidcSlugPattern.MatchString(slug) {
		writeError(w, http.StatusBadRequest, "auth_providers.oidc_invalid_slug")
		return
	}
	if strings.TrimSpace(payload.DisplayName) == "" || strings.TrimSpace(payload.IssuerURL) == "" || strings.TrimSpace(payload.ClientID) == "" || strings.TrimSpace(payload.ClientSecret) == "" {
		writeError(w, http.StatusBadRequest, "auth_providers.oidc_invalid_payload")
		return
	}
	encryptedID, err := s.encryptProviderSecret([]byte(payload.ClientID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_save_failed")
		return
	}
	encryptedSecret, err := s.encryptProviderSecret([]byte(payload.ClientSecret))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_save_failed")
		return
	}
	scopes := normalizeScopes(payload.Scopes)
	policy := authpolicy.OIDCConnectionPolicy{AllowedDomains: normalizeDomains(payload.AllowedDomains), AllowSignup: payload.AllowSignup}
	configBytes, _ := json.Marshal(policy)
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		stmt := `INSERT INTO oidc_connections (tenant_id, slug, display_name, issuer_url, client_id_encrypted, client_secret_encrypted, scopes, config, enabled)
		         VALUES (?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?)`
		return tx.Exec(stmt, tenant.ID, slug, strings.TrimSpace(payload.DisplayName), strings.TrimSpace(payload.IssuerURL), encryptedID, encryptedSecret, pq.StringArray(scopes), string(configBytes), payload.Enabled).Error
	}); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			writeError(w, http.StatusConflict, "auth_providers.oidc_slug_taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_save_failed")
		return
	}
	rec, err := s.loadOIDCRecord(r.Context(), tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) getOIDCConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	rec, err := s.loadOIDCRecord(r.Context(), tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if !rec.Configured() {
		writeError(w, http.StatusNotFound, "auth_providers.oidc_not_found")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) updateOIDCConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	var payload oidcConnectionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	existing, err := s.loadOIDCRow(r.Context(), tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "auth_providers.oidc_not_found")
		return
	}
	display := strings.TrimSpace(payload.DisplayName)
	if display == "" {
		display = existing.DisplayName
	}
	issuer := strings.TrimSpace(payload.IssuerURL)
	if issuer == "" {
		issuer = existing.IssuerURL
	}
	encryptedID := existing.ClientIDEncrypted
	if v := strings.TrimSpace(payload.ClientID); v != "" {
		encryptedID, err = s.encryptProviderSecret([]byte(v))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth_providers.oidc_save_failed")
			return
		}
	}
	encryptedSecret := existing.ClientSecretEncrypted
	if v := strings.TrimSpace(payload.ClientSecret); v != "" {
		encryptedSecret, err = s.encryptProviderSecret([]byte(v))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth_providers.oidc_save_failed")
			return
		}
	}
	scopes := normalizeScopes(payload.Scopes)
	if len(scopes) == 0 {
		scopes = []string(existing.Scopes)
	}
	policy := authpolicy.OIDCConnectionPolicy{AllowedDomains: normalizeDomains(payload.AllowedDomains), AllowSignup: payload.AllowSignup}
	configBytes, _ := json.Marshal(policy)
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		stmt := `UPDATE oidc_connections SET display_name = ?, issuer_url = ?, client_id_encrypted = ?, client_secret_encrypted = ?, scopes = ?, config = ?::jsonb, enabled = ?, updated_at = now() WHERE tenant_id = ? AND slug = ?`
		return tx.Exec(stmt, display, issuer, encryptedID, encryptedSecret, pq.StringArray(scopes), string(configBytes), payload.Enabled, tenant.ID, slug).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_save_failed")
		return
	}
	rec, err := s.loadOIDCRecord(r.Context(), tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) deleteOIDCConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`DELETE FROM oidc_connections WHERE tenant_id = ? AND slug = ?`, tenant.ID, slug).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_delete_failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authOIDCStart(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	ctx, span := observability.StartSpan(r.Context(), "oauth.oidc.start", attribute.String("oidc.slug", slug))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	row, err := s.loadOIDCRow(r.Context(), tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if row == nil || !row.Enabled {
		writeError(w, http.StatusPreconditionRequired, "auth_providers.oidc_not_configured")
		return
	}
	clientID, clientSecret, err := s.decryptOIDCRow(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_decrypt_failed")
		return
	}
	provider := oidcgeneric.Provider{Slug: slug, DisplayName: row.DisplayName, Issuer: row.IssuerURL, DefaultScopes: []string(row.Scopes)}
	creds := upstream.Credentials{ClientID: clientID, ClientSecret: clientSecret, StateSecret: s.GoogleSecret, RedirectURI: "https://" + s.installHost + "/api/v1/auth/oidc/" + slug + "/callback"}
	var payload struct {
		ReturnURL string `json:"return_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	req, err := provider.AuthCodeURL(r.Context(), creds, upstream.AuthParams{TenantID: tenant.ID, ReturnURL: payload.ReturnURL})
	if err != nil {
		observability.RecordError(span, err)
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_start_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": req.URL, "state": req.State, "nonce": req.Nonce})
}

func (s *Server) authOIDCCallback(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	ctx, span := observability.StartSpan(r.Context(), "oauth.oidc.callback", attribute.String("oidc.slug", slug))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	row, err := s.loadOIDCRow(r.Context(), tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if row == nil || !row.Enabled {
		writeError(w, http.StatusPreconditionRequired, "auth_providers.oidc_not_configured")
		return
	}
	clientID, clientSecret, err := s.decryptOIDCRow(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_decrypt_failed")
		return
	}
	var payload struct {
		Code  string `json:"code"`
		State string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	state, err := upstream.ValidateState(s.GoogleSecret, payload.State, time.Now().UTC())
	if err != nil || state.TenantID != tenant.ID || state.Slug != slug {
		recordAuthAttempt("oidc:"+slug, "fail")
		writeError(w, http.StatusBadRequest, "auth_providers.oidc_invalid_state")
		return
	}
	provider := oidcgeneric.Provider{Slug: slug, DisplayName: row.DisplayName, Issuer: row.IssuerURL, DefaultScopes: []string(row.Scopes)}
	creds := upstream.Credentials{ClientID: clientID, ClientSecret: clientSecret, StateSecret: s.GoogleSecret, RedirectURI: "https://" + s.installHost + "/api/v1/auth/oidc/" + slug + "/callback"}
	identity, err := provider.Exchange(r.Context(), creds, upstream.ExchangeParams{Code: payload.Code, State: payload.State, Nonce: state.Nonce, RedirectURI: creds.RedirectURI, Now: time.Now().UTC()})
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("oidc:"+slug, "fail")
		writeError(w, http.StatusBadRequest, "auth_providers.oidc_exchange_failed")
		return
	}
	policy, err := authpolicy.LoadOIDCConnection(r.Context(), s.DB, tenant.ID, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if !policy.AllowsEmail(identity.Email) {
		recordAuthAttempt("oidc:"+slug, "fail")
		writeError(w, http.StatusForbidden, "auth_providers.oidc_email_not_allowed")
		return
	}
	userID, err := s.findOrCreateOIDCUser(r.Context(), tenant.ID, slug, identity)
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("oidc:"+slug, "fail")
		if isSignupError(err) {
			writeError(w, signupHTTPStatus(err), err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "auth_providers.oidc_user_failed")
		return
	}
	sessionID, err := s.establishUserSession(w, r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("oidc:"+slug, "success")
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": state.TenantID, "return_url": state.ReturnURL, "user_id": userID, "session_id": sessionID})
}

func (s *Server) findOrCreateOIDCUser(ctx context.Context, tenantID uuid.UUID, slug string, identity upstream.Identity) (uuid.UUID, error) {
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenantID, &rows, tenantID, identity.Email); err != nil {
		return uuid.Nil, err
	}
	if len(rows) > 0 {
		return rows[0].ID, nil
	}
	if err := authpolicy.CheckSignup(ctx, s.DB, tenantID, authpolicy.SignupContext{Method: authpolicy.MethodForOIDC(slug), Email: identity.Email}); err != nil {
		return uuid.Nil, err
	}
	userID := uuid.New()
	metadata := map[string]any{
		"oidc": map[string]any{
			slug: map[string]any{
				"subject": identity.Subject,
				"name":    identity.Name,
			},
		},
	}
	metadataBytes, _ := json.Marshal(metadata)
	if err := s.TenantDB.Transaction(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES (?, ?, ?, ?::jsonb)`, userID, tenantID, identity.Email, string(metadataBytes)).Error
	}); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (s *Server) loadOIDCConnections(ctx context.Context, tenantID uuid.UUID) ([]oidcDBRow, error) {
	var rows []oidcDBRow
	if err := s.TenantDB.RawScan(ctx, `SELECT id, slug, display_name, issuer_url, client_id_encrypted, client_secret_encrypted, scopes, config, enabled, updated_at FROM oidc_connections WHERE tenant_id = ? ORDER BY slug`, tenantID, &rows, tenantID); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *Server) loadOIDCRow(ctx context.Context, tenantID uuid.UUID, slug string) (*oidcDBRow, error) {
	var rows []oidcDBRow
	if err := s.TenantDB.RawScan(ctx, `SELECT id, slug, display_name, issuer_url, client_id_encrypted, client_secret_encrypted, scopes, config, enabled, updated_at FROM oidc_connections WHERE tenant_id = ? AND slug = ? LIMIT 1`, tenantID, &rows, tenantID, slug); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *Server) loadOIDCRecord(ctx context.Context, tenantID uuid.UUID, slug string) (oidcConnectionRecord, error) {
	row, err := s.loadOIDCRow(ctx, tenantID, slug)
	if err != nil {
		return oidcConnectionRecord{}, err
	}
	if row == nil {
		return oidcConnectionRecord{Slug: slug}, nil
	}
	return oidcRowToRecord(*row), nil
}

func (s *Server) decryptOIDCRow(row *oidcDBRow) (string, string, error) {
	if row == nil {
		return "", "", errors.New("oidc connection missing")
	}
	clientID, err := s.decryptProviderSecret(row.ClientIDEncrypted)
	if err != nil {
		return "", "", err
	}
	clientSecret, err := s.decryptProviderSecret(row.ClientSecretEncrypted)
	if err != nil {
		return "", "", err
	}
	return string(clientID), string(clientSecret), nil
}

func oidcRowToRecord(row oidcDBRow) oidcConnectionRecord {
	policy := authpolicy.OIDCConnectionPolicy{}
	if len(row.Config) > 0 {
		_ = json.Unmarshal(row.Config, &policy)
	}
	policy = policy.WithDefaults()
	updatedAt := row.UpdatedAt
	return oidcConnectionRecord{
		ID:             row.ID,
		Slug:           row.Slug,
		DisplayName:    row.DisplayName,
		IssuerURL:      row.IssuerURL,
		Scopes:         []string(row.Scopes),
		Enabled:        row.Enabled,
		ClientIDSet:    len(row.ClientIDEncrypted) > 0,
		AllowedDomains: policy.AllowedDomains,
		AllowSignup:    policy.AllowSignup,
		UpdatedAt:      &updatedAt,
	}
}

// Configured reports whether the record describes a real, persisted row.
func (r oidcConnectionRecord) Configured() bool { return r.ID != uuid.Nil }

func normalizeScopes(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
