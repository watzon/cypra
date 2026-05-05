package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/auth/magiclink"
	passwordauth "github.com/watzon/cypra/internal/auth/password"
	"github.com/watzon/cypra/internal/auth/totp"
	"github.com/watzon/cypra/internal/auth/upstream/google"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/auth/webauthn2fa"
	"github.com/watzon/cypra/internal/email"
)

type botPayload struct {
	BotToken string `json:"bot_token"`
}

func (p botPayload) GetBotToken() string { return p.BotToken }

type passwordPayload struct {
	botPayload
	Email    string `json:"email"`
	Password string `json:"password"`
	UserID   string `json:"user_id"`
}

type tokenPayload struct {
	botPayload
	Token string `json:"token"`
}

type invitePayload struct {
	botPayload
	Token       string `json:"token"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	RedirectURL string `json:"redirect_url"`
	DisplayName string `json:"display_name"`
}

type credentialPayload struct {
	botPayload
	UserID       string `json:"user_id"`
	CredentialID string `json:"credential_id"`
	PublicKey    string `json:"public_key"`
}

type totpPayload struct {
	botPayload
	UserID string `json:"user_id"`
	Code   string `json:"code"`
}

func (s *Server) authPasswordSignup(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload passwordPayload
	if !s.decodeWithBot(w, r, &payload) {
		return
	}
	userID := uuid.New()
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, $3, '{}'::jsonb)`, userID, tenant.ID, payload.Email); err != nil {
		writeError(w, http.StatusConflict, "user.conflict")
		return
	}
	if err := (passwordauth.Service{DB: s.DB}).SetPassword(r.Context(), tenant.ID, userID, payload.Password); err != nil {
		writeError(w, http.StatusBadRequest, "auth.password_rejected")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": userID})
}

func (s *Server) authPasswordSignin(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload passwordPayload
	if !s.decodeWithBot(w, r, &payload) {
		return
	}
	var userID uuid.UUID
	if err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM users WHERE tenant_id = $1 AND email = $2 AND deleted_at IS NULL`, tenant.ID, payload.Email).Scan(&userID); err != nil {
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials")
		return
	}
	if err := (passwordauth.Service{DB: s.DB}).Verify(r.Context(), userID, payload.Password); err != nil {
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID})
}

func (s *Server) authPasswordReset(w http.ResponseWriter, r *http.Request) {
	var payload passwordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	userID, err := (passwordauth.Service{DB: s.DB}).ResetPassword(r.Context(), payload.UserID, payload.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth.password_reset_invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID})
}

func (s *Server) authMagicLinkIssue(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload passwordPayload
	if !s.decodeWithBot(w, r, &payload) {
		return
	}
	if !s.requireEmailProvider(w, r, tenant.ID) {
		return
	}
	token, err := (magiclink.Service{DB: s.DB}).Issue(r.Context(), tenant.ID, nil, payload.Email, "", 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.magic_link_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (s *Server) authMagicLinkVerify(w http.ResponseWriter, r *http.Request) {
	var payload tokenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	userID, emailAddress, err := (magiclink.Service{DB: s.DB}).Consume(r.Context(), payload.Token)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth.magic_link_invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "email": emailAddress})
}

func (s *Server) adminInvite(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload invitePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if !s.requireEmailProvider(w, r, tenant.ID) {
		return
	}
	token, id, err := (invite.Service{DB: s.DB}).Issue(r.Context(), invite.IssueRequest{TenantID: &tenant.ID, Email: payload.Email, Role: payload.Role, RedirectURL: payload.RedirectURL, CreatedByKind: "tenant_admin"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.issue_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "token": token})
}

func (s *Server) instanceInvite(w http.ResponseWriter, r *http.Request) {
	var payload invitePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	token, id, err := (invite.Service{DB: s.DB}).Issue(r.Context(), invite.IssueRequest{Email: payload.Email, Role: "instance_admin", RedirectURL: payload.RedirectURL, CreatedByKind: "instance_admin"})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.issue_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "token": token})
}

func (s *Server) authInviteRedeem(w http.ResponseWriter, r *http.Request) {
	var payload invitePayload
	if !s.decodeWithBot(w, r, &payload) {
		return
	}
	redeemed, err := (invite.Service{DB: s.DB}).Redeem(r.Context(), payload.Token, payload.DisplayName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invite.invalid")
		return
	}
	writeJSON(w, http.StatusOK, redeemed)
}

func (s *Server) authPasskeyRegister(w http.ResponseWriter, r *http.Request) {
	tenant, userID, credentialID, publicKey, ok := s.decodeCredential(w, r)
	if !ok {
		return
	}
	if err := (webauthn.Service{DB: s.DB}).Register(r.Context(), tenant.ID, userID, s.rpID(tenant), credentialID, publicKey); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.passkey_register_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) authPasskeyAssert(w http.ResponseWriter, r *http.Request) {
	tenant, _, credentialID, _, ok := s.decodeCredential(w, r)
	if !ok {
		return
	}
	userID, err := (webauthn.Service{DB: s.DB}).Assert(r.Context(), tenant.ID, s.rpID(tenant), credentialID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth.passkey_invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID})
}

func (s *Server) authTOTPEnroll(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload totpPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	secret, uri, err := (totp.Service{DB: s.DB, KEK: s.MasterKey}).Enroll(r.Context(), tenant.ID, userID, "Cypra", payload.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.totp_enroll_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"secret": secret, "uri": uri})
}

func (s *Server) authTOTPVerify(w http.ResponseWriter, r *http.Request) {
	var payload totpPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	ok, err := (totp.Service{DB: s.DB, KEK: s.MasterKey}).Verify(r.Context(), userID, payload.Code, time.Now().UTC())
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "auth.totp_invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) authWebAuthn2FAEnroll(w http.ResponseWriter, r *http.Request) {
	tenant, userID, credentialID, publicKey, ok := s.decodeCredential(w, r)
	if !ok {
		return
	}
	if err := (webauthn2fa.Service{Primary: webauthn.Service{DB: s.DB}}).Enroll(r.Context(), tenant.ID, userID, s.rpID(tenant), credentialID, publicKey); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.webauthn2fa_enroll_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) authWebAuthn2FAVerify(w http.ResponseWriter, r *http.Request) {
	tenant, _, credentialID, _, ok := s.decodeCredential(w, r)
	if !ok {
		return
	}
	userID, err := (webauthn2fa.Service{Primary: webauthn.Service{DB: s.DB}}).Verify(r.Context(), tenant.ID, s.rpID(tenant), credentialID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "auth.webauthn2fa_invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID})
}

func (s *Server) authBackupCodesRegenerate(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload passwordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	codes, err := (backupcodes.Service{DB: s.DB}).RegenerateForUser(r.Context(), tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.backup_codes_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"codes": codes})
}

func (s *Server) authBackupCodesConsume(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		UserID string `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	ok, err := (backupcodes.Service{DB: s.DB}).ConsumeForUser(r.Context(), userID, payload.Code)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "auth.backup_code_invalid")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) authGoogleStart(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload struct {
		ReturnURL string `json:"return_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	url, state, err := (google.Client{ClientID: "configured-later", RedirectURI: "https://" + s.installHost + "/api/v1/auth/google/callback", Secret: s.GoogleSecret}).AuthCodeURL(tenant.ID, payload.ReturnURL, 10*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.google_start_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": url, "nonce": state.Nonce})
}

func (s *Server) authGoogleCallback(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		State   string `json:"state"`
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	state, err := (google.Client{Secret: s.GoogleSecret}).ValidateState(payload.State, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusBadRequest, google.ErrStateMismatch.Error())
		return
	}
	if err := google.ValidateIDTokenNonce(payload.IDToken, state.Nonce); err != nil {
		writeError(w, http.StatusBadRequest, google.ErrNonceMismatch.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": state.TenantID, "return_url": state.ReturnURL})
}

func (s *Server) decodeWithBot(w http.ResponseWriter, r *http.Request, dest interface{ GetBotToken() string }) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return false
	}
	if err := s.BotVerifier.Verify(r.Context(), dest.GetBotToken()); err != nil {
		writeError(w, http.StatusForbidden, "bot.verify_failed")
		return false
	}
	return true
}

func (s *Server) decodeCredential(w http.ResponseWriter, r *http.Request) (Tenant, uuid.UUID, []byte, []byte, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return Tenant{}, uuid.Nil, nil, nil, false
	}
	var payload credentialPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return Tenant{}, uuid.Nil, nil, nil, false
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return Tenant{}, uuid.Nil, nil, nil, false
	}
	credentialID, err := base64.RawURLEncoding.DecodeString(payload.CredentialID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "credential.invalid")
		return Tenant{}, uuid.Nil, nil, nil, false
	}
	publicKey, _ := base64.RawURLEncoding.DecodeString(payload.PublicKey)
	return tenant, userID, credentialID, publicKey, true
}

func (s *Server) requireEmailProvider(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) bool {
	_, err := (email.Resolver{DB: s.DB, KEK: s.MasterKey}).Resolve(r.Context(), tenantID)
	if errors.Is(err, email.ErrProviderRequired) {
		writeError(w, http.StatusPreconditionRequired, email.ErrProviderRequired.Error())
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "email.provider_error")
		return false
	}
	return true
}

func (s *Server) rpID(tenant Tenant) string {
	return tenant.Slug + "." + strings.Split(s.installHost, ":")[0]
}
