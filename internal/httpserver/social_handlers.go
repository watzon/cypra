package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"

	"github.com/watzon/cypra/internal/auth/upstream"
	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/observability"
)

// socialKinds is the closed set of social_provider_kind enum values. Update
// alongside any future migration that widens the enum.
var socialKinds = []string{"google", "microsoft", "apple", "github", "discord"}

func isSocialKind(kind string) bool {
	for _, k := range socialKinds {
		if k == kind {
			return true
		}
	}
	return false
}

type socialConnectionPayload struct {
	Enabled        bool            `json:"enabled"`
	ClientID       string          `json:"client_id"`
	ClientSecret   string          `json:"client_secret"`
	AllowedDomains []string        `json:"allowed_domains"`
	AllowSignup    *bool           `json:"allow_signup,omitempty"`
	Config         json.RawMessage `json:"config,omitempty"`
}

type socialConnectionRecord struct {
	Kind           string         `json:"kind"`
	Label          string         `json:"label"`
	IconURL        string         `json:"icon_url"`
	Configured     bool           `json:"configured"`
	Enabled        bool           `json:"enabled"`
	ClientIDSet    bool           `json:"client_id_set"`
	AllowedDomains []string       `json:"allowed_domains"`
	AllowSignup    *bool          `json:"allow_signup,omitempty"`
	Config         map[string]any `json:"config"`
	UpdatedAt      *time.Time     `json:"updated_at,omitempty"`
}

func (s *Server) listSocialConnections(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	rows, err := s.loadSocialRows(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	configured := make(map[string]socialDBRow, len(rows))
	for _, row := range rows {
		configured[row.Kind] = row
	}
	out := make([]socialConnectionRecord, 0, len(socialKinds))
	for _, kind := range socialKinds {
		display := upstream.Display{Label: titleKind(kind)}
		if p, err := upstream.Resolve(kind); err == nil {
			display = p.Display()
		}
		row, present := configured[kind]
		rec := socialConnectionRecord{Kind: kind, Label: display.Label, IconURL: display.IconURL, Configured: present, Enabled: row.Enabled}
		if present {
			policy := unmarshalSocialPolicy(row.Config)
			rec.AllowedDomains = policy.AllowedDomains
			rec.AllowSignup = policy.AllowSignup
			rec.Config = jsonbToMap(row.Config)
			rec.ClientIDSet = len(row.ClientIDEncrypted) > 0
			updatedAt := row.UpdatedAt
			rec.UpdatedAt = &updatedAt
		} else {
			rec.Config = map[string]any{}
		}
		out = append(out, rec)
	}
	writeJSON(w, http.StatusOK, map[string]any{"connections": out})
}

func (s *Server) getSocialConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	if !isSocialKind(kind) {
		writeError(w, http.StatusBadRequest, "auth_providers.social_unknown_kind")
		return
	}
	rec, err := s.loadSocialRecord(r.Context(), tenant.ID, kind)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) saveSocialConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	if !isSocialKind(kind) {
		writeError(w, http.StatusBadRequest, "auth_providers.social_unknown_kind")
		return
	}
	var payload socialConnectionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	existing, err := s.loadSocialRow(r.Context(), tenant.ID, kind)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	clientID := strings.TrimSpace(payload.ClientID)
	clientSecret := strings.TrimSpace(payload.ClientSecret)
	var encryptedID, encryptedSecret []byte
	if clientID != "" {
		encryptedID, err = s.encryptProviderSecret([]byte(clientID))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth_providers.social_save_failed")
			return
		}
	} else if existing != nil {
		encryptedID = existing.ClientIDEncrypted
	}
	if clientSecret != "" {
		encryptedSecret, err = s.encryptProviderSecret([]byte(clientSecret))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth_providers.social_save_failed")
			return
		}
	} else if existing != nil {
		encryptedSecret = existing.ClientSecretEncrypted
	}
	if len(encryptedID) == 0 || len(encryptedSecret) == 0 {
		writeError(w, http.StatusBadRequest, "auth_providers.social_credentials_required")
		return
	}
	policy := authpolicy.SocialPolicy{AllowedDomains: normalizeDomains(payload.AllowedDomains), AllowSignup: payload.AllowSignup}
	configBytes, err := json.Marshal(policy)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.social_save_failed")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		stmt := `INSERT INTO social_connections (tenant_id, kind, enabled, client_id_encrypted, client_secret_encrypted, config)
		         VALUES (?, ?::social_provider_kind, ?, ?, ?, ?::jsonb)
		         ON CONFLICT (tenant_id, kind) DO UPDATE SET
		             enabled = EXCLUDED.enabled,
		             client_id_encrypted = EXCLUDED.client_id_encrypted,
		             client_secret_encrypted = EXCLUDED.client_secret_encrypted,
		             config = EXCLUDED.config,
		             updated_at = now()`
		if err := tx.Exec(stmt, tenant.ID, kind, payload.Enabled, encryptedID, encryptedSecret, string(configBytes)).Error; err != nil {
			return err
		}
		// Back-compat: keep upstream_providers and tenant_auth_methods.google
		// in lockstep so the existing GoogleURL hosted-login path keeps working
		// until callers fully migrate to social_connections.
		if kind == "google" {
			if err := tx.Exec(`DELETE FROM upstream_providers WHERE tenant_id = ? AND kind = 'google'`, tenant.ID).Error; err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO upstream_providers (tenant_id, kind, client_id_encrypted, client_secret_encrypted, enabled) VALUES (?, 'google'::upstream_provider_kind, ?, ?, ?)`, tenant.ID, encryptedID, encryptedSecret, payload.Enabled).Error; err != nil {
				return err
			}
			if err := tx.Exec(`INSERT INTO tenant_auth_methods (tenant_id, method, enabled, config) VALUES (?, 'google'::tenant_auth_method, ?, ?::jsonb) ON CONFLICT (tenant_id, method) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now()`, tenant.ID, payload.Enabled, string(configBytes)).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.social_save_failed")
		return
	}
	rec, err := s.loadSocialRecord(r.Context(), tenant.ID, kind)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) deleteSocialConnection(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	if !isSocialKind(kind) {
		writeError(w, http.StatusBadRequest, "auth_providers.social_unknown_kind")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM social_connections WHERE tenant_id = ? AND kind = ?::social_provider_kind`, tenant.ID, kind).Error; err != nil {
			return err
		}
		if kind == "google" {
			if err := tx.Exec(`DELETE FROM upstream_providers WHERE tenant_id = ? AND kind = 'google'`, tenant.ID).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.social_delete_failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// authSocialStart issues an authorization-code redirect URL for the chosen kind.
func (s *Server) authSocialStart(w http.ResponseWriter, r *http.Request) {
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	ctx, span := observability.StartSpan(r.Context(), "oauth.social.start", attribute.String("oauth.provider", kind))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	provider, err := upstream.Resolve(kind)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.social_unknown_kind")
		return
	}
	creds, err := s.loadSocialCredentials(r.Context(), tenant.ID, kind)
	if err != nil {
		writeError(w, http.StatusPreconditionRequired, "auth_providers.social_not_configured")
		return
	}
	var payload struct {
		ReturnURL string `json:"return_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	creds.RedirectURI = "https://" + s.installHost + "/api/v1/auth/social/" + kind + "/callback"
	req, err := provider.AuthCodeURL(r.Context(), creds, upstream.AuthParams{TenantID: tenant.ID, ReturnURL: payload.ReturnURL})
	if err != nil {
		observability.RecordError(span, err)
		writeError(w, http.StatusInternalServerError, "auth_providers.social_start_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": req.URL, "state": req.State, "nonce": req.Nonce})
}

// authSocialCallback completes the OAuth flow and creates / signs in the user.
func (s *Server) authSocialCallback(w http.ResponseWriter, r *http.Request) {
	kind := strings.ToLower(chi.URLParam(r, "kind"))
	ctx, span := observability.StartSpan(r.Context(), "oauth.social.callback", attribute.String("oauth.provider", kind))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	provider, err := upstream.Resolve(kind)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.social_unknown_kind")
		return
	}
	creds, err := s.loadSocialCredentials(r.Context(), tenant.ID, kind)
	if err != nil {
		writeError(w, http.StatusPreconditionRequired, "auth_providers.social_not_configured")
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
	state, err := upstream.ValidateState(creds.StateSecret, payload.State, time.Now().UTC())
	if err != nil || state.TenantID != tenant.ID {
		recordAuthAttempt(kind, "fail")
		writeError(w, http.StatusBadRequest, "auth_providers.social_invalid_state")
		return
	}
	creds.RedirectURI = "https://" + s.installHost + "/api/v1/auth/social/" + kind + "/callback"
	identity, err := provider.Exchange(r.Context(), creds, upstream.ExchangeParams{Code: payload.Code, State: payload.State, Nonce: state.Nonce, RedirectURI: creds.RedirectURI, Now: time.Now().UTC()})
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt(kind, "fail")
		writeError(w, http.StatusBadRequest, "auth_providers.social_exchange_failed")
		return
	}
	policy, err := authpolicy.LoadSocial(r.Context(), s.DB, tenant.ID, kind)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if !policy.AllowsEmail(identity.Email) {
		recordAuthAttempt(kind, "fail")
		writeError(w, http.StatusForbidden, "auth_providers.social_email_not_allowed")
		return
	}
	userID, err := s.findOrCreateSocialUser(r.Context(), tenant.ID, kind, identity)
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt(kind, "fail")
		if isSignupError(err) {
			writeError(w, signupHTTPStatus(err), err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "auth_providers.social_user_failed")
		return
	}
	sessionID, err := s.establishUserSession(w, r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt(kind, "success")
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": state.TenantID, "return_url": state.ReturnURL, "user_id": userID, "session_id": sessionID})
}

func (s *Server) findOrCreateSocialUser(ctx context.Context, tenantID uuid.UUID, kind string, identity upstream.Identity) (uuid.UUID, error) {
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenantID, &rows, tenantID, identity.Email); err != nil {
		return uuid.Nil, err
	}
	if len(rows) > 0 {
		return rows[0].ID, nil
	}
	if err := authpolicy.CheckSignup(ctx, s.DB, tenantID, authpolicy.SignupContext{Method: authpolicy.MethodForSocial(kind), Email: identity.Email}); err != nil {
		return uuid.Nil, err
	}
	userID := uuid.New()
	metadata := map[string]any{
		"social": map[string]any{
			kind: map[string]any{
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

type socialDBRow struct {
	Kind                  string    `gorm:"column:kind"`
	Enabled               bool      `gorm:"column:enabled"`
	ClientIDEncrypted     []byte    `gorm:"column:client_id_encrypted"`
	ClientSecretEncrypted []byte    `gorm:"column:client_secret_encrypted"`
	Config                []byte    `gorm:"column:config"`
	UpdatedAt             time.Time `gorm:"column:updated_at"`
}

func (s *Server) loadSocialRows(ctx context.Context, tenantID uuid.UUID) ([]socialDBRow, error) {
	var rows []socialDBRow
	if err := s.TenantDB.RawScan(ctx, `SELECT kind::text AS kind, enabled, client_id_encrypted, client_secret_encrypted, config, updated_at FROM social_connections WHERE tenant_id = ?`, tenantID, &rows, tenantID); err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Kind < rows[j].Kind })
	return rows, nil
}

func (s *Server) loadSocialRow(ctx context.Context, tenantID uuid.UUID, kind string) (*socialDBRow, error) {
	var rows []socialDBRow
	if err := s.TenantDB.RawScan(ctx, `SELECT kind::text AS kind, enabled, client_id_encrypted, client_secret_encrypted, config, updated_at FROM social_connections WHERE tenant_id = ? AND kind = ?::social_provider_kind LIMIT 1`, tenantID, &rows, tenantID, kind); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (s *Server) loadSocialRecord(ctx context.Context, tenantID uuid.UUID, kind string) (socialConnectionRecord, error) {
	display := upstream.Display{Label: titleKind(kind)}
	if p, err := upstream.Resolve(kind); err == nil {
		display = p.Display()
	}
	row, err := s.loadSocialRow(ctx, tenantID, kind)
	if err != nil {
		return socialConnectionRecord{}, err
	}
	rec := socialConnectionRecord{Kind: kind, Label: display.Label, IconURL: display.IconURL, Configured: row != nil, Config: map[string]any{}}
	if row != nil {
		policy := unmarshalSocialPolicy(row.Config)
		rec.Enabled = row.Enabled
		rec.ClientIDSet = len(row.ClientIDEncrypted) > 0
		rec.AllowedDomains = policy.AllowedDomains
		rec.AllowSignup = policy.AllowSignup
		rec.Config = jsonbToMap(row.Config)
		updatedAt := row.UpdatedAt
		rec.UpdatedAt = &updatedAt
	}
	return rec, nil
}

func (s *Server) loadSocialCredentials(ctx context.Context, tenantID uuid.UUID, kind string) (upstream.Credentials, error) {
	row, err := s.loadSocialRow(ctx, tenantID, kind)
	if err != nil {
		return upstream.Credentials{}, err
	}
	if row == nil || !row.Enabled {
		return upstream.Credentials{}, errors.New("social connection not configured")
	}
	clientID, err := s.decryptProviderSecret(row.ClientIDEncrypted)
	if err != nil {
		return upstream.Credentials{}, err
	}
	clientSecret, err := s.decryptProviderSecret(row.ClientSecretEncrypted)
	if err != nil {
		return upstream.Credentials{}, err
	}
	return upstream.Credentials{ClientID: string(clientID), ClientSecret: string(clientSecret), StateSecret: s.GoogleSecret}, nil
}

func unmarshalSocialPolicy(raw []byte) authpolicy.SocialPolicy {
	if len(raw) == 0 {
		return authpolicy.SocialPolicy{}.WithDefaults()
	}
	var policy authpolicy.SocialPolicy
	_ = json.Unmarshal(raw, &policy)
	return policy.WithDefaults()
}

func jsonbToMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func normalizeDomains(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		d := strings.ToLower(strings.TrimSpace(raw))
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}
