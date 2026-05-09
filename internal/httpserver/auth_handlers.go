package httpserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/audit"
	"github.com/watzon/cypra/internal/auth"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/auth/magiclink"
	passwordauth "github.com/watzon/cypra/internal/auth/password"
	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/auth/totp"
	"github.com/watzon/cypra/internal/auth/upstream/google"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/auth/webauthn2fa"
	"github.com/watzon/cypra/internal/email"
	"github.com/watzon/cypra/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
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
	Token          string `json:"token"`
	ContinuationID string `json:"continuation_id"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	RedirectURL    string `json:"redirect_url"`
	DisplayName    string `json:"display_name"`
	Password       string `json:"password"`
}

type credentialPayload struct {
	botPayload
	UserID       string          `json:"user_id"`
	CredentialID string          `json:"credential_id"`
	PublicKey    string          `json:"public_key"`
	CeremonyID   string          `json:"ceremony_id"`
	Response     json.RawMessage `json:"response"`
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
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeError(w, http.StatusForbidden, "tenant.suspended")
		return
	}
	var payload passwordPayload
	if !s.decodeWithBot(w, r, &payload) {
		return
	}
	if !s.allowRate(w, r, &tenant.ID, "signup:ip", clientRateKey(r), 5, time.Minute) {
		return
	}
	if err := authpolicy.CheckSignup(r.Context(), s.DB, tenant.ID, authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: strings.TrimSpace(payload.Email)}); err != nil {
		recordAuthAttempt("password", "fail")
		writeError(w, signupHTTPStatus(err), err.Error())
		return
	}
	userID := uuid.New()
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES (?, ?, ?, '{}'::jsonb)`, userID, tenant.ID, payload.Email).Error
	}); err != nil {
		recordAuthAttempt("password", "fail")
		writeError(w, http.StatusConflict, "user.conflict")
		return
	}
	if err := (passwordauth.Service{DB: s.DB}).SetPassword(r.Context(), tenant.ID, userID, payload.Password); err != nil {
		recordAuthAttempt("password", "fail")
		writeError(w, http.StatusBadRequest, "auth.password_rejected")
		return
	}
	recordAuthAttempt("password", "success")
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": userID})
}

func (s *Server) authPasswordSignin(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeError(w, http.StatusForbidden, "tenant.suspended")
		return
	}
	var payload passwordPayload
	if !s.decodeWithBot(w, r, &payload) {
		return
	}
	if !s.allowRate(w, r, &tenant.ID, "login:ip", clientRateKey(r), 10, time.Minute) || !s.allowRate(w, r, &tenant.ID, "login:account", payload.Email, 5, time.Minute) {
		return
	}
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenant.ID, &rows, tenant.ID, payload.Email); err != nil || len(rows) == 0 {
		recordAuthAttempt("password", "fail")
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials")
		return
	}
	userID := rows[0].ID
	if err := (passwordauth.Service{DB: s.DB}).Verify(r.Context(), userID, payload.Password); err != nil {
		recordAuthAttempt("password", "fail")
		writeError(w, http.StatusUnauthorized, "auth.invalid_credentials")
		return
	}
	sessionID, err := s.establishUserSession(w, r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("password", "success")
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "session_id": sessionID})
}

func (s *Server) authPasswordReset(w http.ResponseWriter, r *http.Request) {
	var payload passwordPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if !s.allowRate(w, r, nil, "password_reset:account", payload.UserID, 5, time.Hour) {
		return
	}
	userID, err := (passwordauth.Service{DB: s.DB}).ResetPassword(r.Context(), payload.UserID, payload.Password)
	if err != nil {
		recordAuthAttempt("password", "fail")
		writeError(w, http.StatusBadRequest, "auth.password_reset_invalid")
		return
	}
	recordAuthAttempt("password", "success")
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
	if !s.allowRate(w, r, &tenant.ID, "magic_link:account", payload.Email, 5, time.Hour) {
		return
	}
	if !s.requireEmailProvider(w, r, tenant.ID) {
		return
	}
	userID, err := s.userIDByEmail(r.Context(), tenant.ID, payload.Email)
	if err != nil {
		recordAuthAttempt("magic_link", "fail")
		writeError(w, http.StatusBadRequest, "auth.magic_link_invalid")
		return
	}
	policy, err := authpolicy.LoadMagicLink(r.Context(), s.DB, tenant.ID)
	if err != nil {
		recordAuthAttempt("magic_link", "fail")
		writeError(w, http.StatusInternalServerError, "auth.magic_link_failed")
		return
	}
	token, err := (magiclink.Service{DB: s.DB}).Issue(r.Context(), tenant.ID, &userID, payload.Email, "", policy)
	if errors.Is(err, magiclink.ErrTooManyActive) {
		recordAuthAttempt("magic_link", "fail")
		writeError(w, http.StatusTooManyRequests, "auth.magic_link_too_many_active")
		return
	}
	if err != nil {
		recordAuthAttempt("magic_link", "fail")
		writeError(w, http.StatusInternalServerError, "auth.magic_link_failed")
		return
	}
	recordAuthAttempt("magic_link", "success")
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
		recordAuthAttempt("magic_link", "fail")
		writeError(w, http.StatusBadRequest, "auth.magic_link_invalid")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	if !s.userBelongsToTenant(r.Context(), tenant.ID, userID) {
		recordAuthAttempt("magic_link", "fail")
		writeError(w, http.StatusBadRequest, "auth.magic_link_invalid")
		return
	}
	sessionID, err := s.establishUserSession(w, r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("magic_link", "success")
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "email": emailAddress, "session_id": sessionID})
}

func (s *Server) userIDByEmail(ctx context.Context, tenantID uuid.UUID, email string) (uuid.UUID, error) {
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenantID, &rows, tenantID, strings.TrimSpace(email)); err != nil {
		return uuid.Nil, err
	}
	if len(rows) == 0 {
		return uuid.Nil, sql.ErrNoRows
	}
	return rows[0].ID, nil
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

func (s *Server) listTenantInvites(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, email, role::text, expires_at, created_at FROM pending_invitations WHERE tenant_id = $1 AND redeemed_at IS NULL ORDER BY created_at DESC`, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.list_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var email, role string
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&id, &email, &role, &expiresAt, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, "invite.list_failed")
			return
		}
		items = append(items, map[string]any{"id": id, "email": email, "role": role, "expires_at": expiresAt.Format(time.RFC3339), "created_at": createdAt.Format(time.RFC3339)})
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) revokeTenantInvite(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invite.id_invalid")
		return
	}
	result, err := s.DB.ExecContext(r.Context(), `DELETE FROM pending_invitations WHERE id = $1 AND tenant_id = $2 AND redeemed_at IS NULL`, id, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.revoke_failed")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusNotFound, "invite.not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
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

func (s *Server) listInstanceInvites(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `SELECT id, email, role::text, expires_at, created_at FROM pending_invitations WHERE tenant_id IS NULL AND redeemed_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.list_failed")
		return
	}
	defer func() { _ = rows.Close() }()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var email, role string
		var expiresAt, createdAt time.Time
		if err := rows.Scan(&id, &email, &role, &expiresAt, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, "invite.list_failed")
			return
		}
		items = append(items, map[string]any{"id": id, "email": email, "role": role, "expires_at": expiresAt.Format(time.RFC3339), "created_at": createdAt.Format(time.RFC3339)})
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "invite.list_failed")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) revokeInstanceInvite(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invite.id_invalid")
		return
	}
	result, err := s.DB.ExecContext(r.Context(), `DELETE FROM pending_invitations WHERE id = $1 AND tenant_id IS NULL AND redeemed_at IS NULL`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invite.revoke_failed")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusNotFound, "invite.not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (s *Server) authInviteRedeem(w http.ResponseWriter, r *http.Request) {
	payload, form, ok := s.decodeInvitePayload(w, r)
	if !ok {
		return
	}
	rateKey := payload.Token
	if rateKey == "" {
		rateKey = payload.ContinuationID
	}
	if !s.allowRate(w, r, nil, "invite_redeem:token", rateKey, 5, time.Minute) {
		return
	}
	redeemed, err := s.redeemInvitePayload(r, payload)
	if err != nil {
		if isSignupError(err) {
			if form {
				writeHTMXError(w, signupHTTPStatus(err), err.Error())
			} else {
				writeError(w, signupHTTPStatus(err), err.Error())
			}
			return
		}
		if form {
			writeHTMXError(w, http.StatusBadRequest, "Invite link is invalid or expired.")
		} else {
			writeError(w, http.StatusBadRequest, "invite.invalid")
		}
		return
	}
	if redeemed.TenantID != nil && redeemed.UserID != uuid.Nil && strings.TrimSpace(payload.Password) != "" {
		if err := (passwordauth.Service{DB: s.DB}).SetPassword(r.Context(), *redeemed.TenantID, redeemed.UserID, payload.Password); err != nil {
			writeError(w, http.StatusBadRequest, "auth.password_rejected")
			return
		}
	}
	if redeemed.TenantID != nil && redeemed.UserID != uuid.Nil {
		sessionID, err := s.establishUserSession(w, r, *redeemed.TenantID, redeemed.UserID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.session_failed")
			return
		}
		if form {
			writeHTMXMessage(w, "Invite accepted. Add a passkey to finish setup.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invite": redeemed, "session_id": sessionID})
		return
	}
	if redeemed.AdminID != uuid.Nil {
		sessionID, err := s.establishInstanceAdminSession(w, r, redeemed.AdminID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.session_failed")
			return
		}
		if form {
			writeHTMXMessage(w, "Invite accepted. Add a passkey to finish setup.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"invite": redeemed, "session_id": sessionID})
		return
	}
	writeJSON(w, http.StatusOK, redeemed)
}

func (s *Server) decodeInvitePayload(w http.ResponseWriter, r *http.Request) (invitePayload, bool, bool) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
		if err := r.ParseForm(); err != nil {
			writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
			return invitePayload{}, true, false
		}
		return invitePayload{Token: r.FormValue("token"), ContinuationID: r.FormValue("continuation_id"), DisplayName: r.FormValue("display_name"), Password: r.FormValue("password")}, true, true
	}
	var payload invitePayload
	if !s.decodeWithBot(w, r, &payload) {
		return invitePayload{}, false, false
	}
	return payload, false, true
}

func (s *Server) redeemInvitePayload(r *http.Request, payload invitePayload) (invite.Redeemed, error) {
	if strings.TrimSpace(payload.ContinuationID) == "" {
		if err := s.gateInviteByTenant(r.Context(), payload.Token, ""); err != nil {
			return invite.Redeemed{}, err
		}
		return (invite.Service{DB: s.DB}).Redeem(r.Context(), payload.Token, payload.DisplayName)
	}
	continuationID, err := uuid.Parse(payload.ContinuationID)
	if err != nil {
		return invite.Redeemed{}, invite.ErrInvalidToken
	}
	cookie, err := r.Cookie("cypra_invite_continuation")
	if err != nil || cookie.Value != continuationID.String() {
		return invite.Redeemed{}, invite.ErrInvalidToken
	}
	if err := s.gateInviteByTenant(r.Context(), "", continuationID.String()); err != nil {
		return invite.Redeemed{}, err
	}
	return (invite.Service{DB: s.DB}).RedeemContinuation(r.Context(), continuationID, payload.DisplayName)
}

// gateInviteByTenant looks up the invite (without consuming it) and rejects
// the redemption if the owning tenant has disabled invite-based registration.
// Instance-admin invites (tenant_id = NULL) are exempt.
func (s *Server) gateInviteByTenant(ctx context.Context, token, continuationID string) error {
	var tenantID *uuid.UUID
	var email string
	var err error
	if continuationID != "" {
		var raw uuid.UUID
		err = s.DB.QueryRowContext(ctx, `SELECT tenant_id, email FROM invite_continuations WHERE id = $1 AND consumed_at IS NULL AND expires_at > now()`, continuationID).Scan(&tenantID, &email)
		_ = raw
	} else if token != "" {
		err = s.DB.QueryRowContext(ctx, `SELECT tenant_id, email FROM pending_invitations WHERE token_hash = $1 AND redeemed_at IS NULL AND expires_at > now()`, hashInviteToken(token)).Scan(&tenantID, &email)
	}
	if errors.Is(err, sql.ErrNoRows) {
		// Surface as invalid token; the redeem call will handle it.
		return nil
	}
	if err != nil {
		return err
	}
	if tenantID == nil {
		return nil
	}
	return authpolicy.CheckSignup(ctx, s.DB, *tenantID, authpolicy.SignupContext{ViaInvite: true, Email: email})
}

// hashInviteToken matches the hashing used in invite.Service (SHA-256 of the token).
func hashInviteToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func (s *Server) authPasskeyRegister(w http.ResponseWriter, r *http.Request) {
	tenant, payload, ok := s.decodeCredentialPayload(w, r)
	if !ok {
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	service := webauthn.Service{DB: s.DB}
	if strings.TrimSpace(payload.CeremonyID) == "" {
		if strings.TrimSpace(payload.CredentialID) != "" || strings.TrimSpace(payload.PublicKey) != "" {
			writeError(w, http.StatusBadRequest, "webauthn.ceremony_required")
			return
		}
		options, ceremonyID, err := service.BeginRegistration(r.Context(), webauthn.BeginRegistrationRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.passkey_register_failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ceremony_id": ceremonyID, "options": options})
		return
	}
	ceremonyID, err := uuid.Parse(payload.CeremonyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "webauthn.ceremony_invalid")
		return
	}
	if _, err := service.FinishRegistration(r.Context(), webauthn.FinishRegistrationRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant), Origins: []string{requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)}); err != nil {
		if s.DevOpenAPI {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "auth.passkey_register_failed", "detail": err.Error()})
			return
		}
		writeError(w, http.StatusInternalServerError, "auth.passkey_register_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) authPasskeyAssert(w http.ResponseWriter, r *http.Request) {
	tenant, payload, ok := s.decodeCredentialPayload(w, r)
	if !ok {
		return
	}
	service := webauthn.Service{DB: s.DB}
	if strings.TrimSpace(payload.CeremonyID) == "" {
		if strings.TrimSpace(payload.CredentialID) != "" {
			writeError(w, http.StatusUnauthorized, "auth.passkey_invalid")
			return
		}
		options, ceremonyID, err := service.BeginAssertion(r.Context(), webauthn.BeginAssertionRequest{TenantID: tenant.ID, RPID: s.rpID(tenant)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.passkey_assert_failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ceremony_id": ceremonyID, "options": options})
		return
	}
	ceremonyID, err := uuid.Parse(payload.CeremonyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "webauthn.ceremony_invalid")
		return
	}
	userID, _, err := service.FinishAssertion(r.Context(), webauthn.FinishAssertionRequest{TenantID: tenant.ID, RPID: s.rpID(tenant), Origins: []string{requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)})
	if err != nil {
		recordAuthAttempt("passkey", "fail")
		writeError(w, http.StatusUnauthorized, "auth.passkey_invalid")
		return
	}
	sessionID, err := s.establishUserSession(w, r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("passkey", "success")
	writeJSON(w, http.StatusOK, map[string]any{"user_id": userID, "session_id": sessionID})
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
		recordAuthAttempt("totp", "fail")
		writeError(w, http.StatusUnauthorized, "auth.totp_invalid")
		return
	}
	recordAuthAttempt("totp", "success")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) authWebAuthn2FAEnroll(w http.ResponseWriter, r *http.Request) {
	tenant, payload, ok := s.decodeCredentialPayload(w, r)
	if !ok {
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	service := webauthn2fa.Service{Primary: webauthn.Service{DB: s.DB}}
	if strings.TrimSpace(payload.CeremonyID) == "" {
		if strings.TrimSpace(payload.CredentialID) != "" || strings.TrimSpace(payload.PublicKey) != "" {
			writeError(w, http.StatusBadRequest, "webauthn.ceremony_required")
			return
		}
		options, ceremonyID, err := service.BeginEnroll(r.Context(), webauthn2fa.BeginEnrollRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.webauthn2fa_enroll_failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ceremony_id": ceremonyID, "options": options})
		return
	}
	ceremonyID, err := uuid.Parse(payload.CeremonyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "webauthn.ceremony_invalid")
		return
	}
	if _, err := service.FinishEnroll(r.Context(), webauthn2fa.FinishEnrollRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant), Origins: []string{requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.webauthn2fa_enroll_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) authWebAuthn2FAVerify(w http.ResponseWriter, r *http.Request) {
	tenant, payload, ok := s.decodeCredentialPayload(w, r)
	if !ok {
		return
	}
	service := webauthn2fa.Service{Primary: webauthn.Service{DB: s.DB}}
	if strings.TrimSpace(payload.CeremonyID) == "" {
		if strings.TrimSpace(payload.CredentialID) != "" {
			writeError(w, http.StatusUnauthorized, "auth.webauthn2fa_invalid")
			return
		}
		var userID uuid.UUID
		if strings.TrimSpace(payload.UserID) != "" {
			parsed, err := uuid.Parse(payload.UserID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "user.id_invalid")
				return
			}
			userID = parsed
		} else if current, ok := s.currentUserSession(r, tenant.ID); ok {
			userID = current
		}
		options, ceremonyID, err := service.BeginVerify(r.Context(), webauthn2fa.BeginVerifyRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant)})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.webauthn2fa_verify_failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ceremony_id": ceremonyID, "options": options})
		return
	}
	ceremonyID, err := uuid.Parse(payload.CeremonyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "webauthn.ceremony_invalid")
		return
	}
	userID, _, err := service.FinishVerify(r.Context(), webauthn2fa.FinishVerifyRequest{TenantID: tenant.ID, RPID: s.rpID(tenant), Origins: []string{requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)})
	if err != nil {
		recordAuthAttempt("webauthn2fa", "fail")
		writeError(w, http.StatusUnauthorized, "auth.webauthn2fa_invalid")
		return
	}
	recordAuthAttempt("webauthn2fa", "success")
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
	ctx, span := observability.StartSpan(r.Context(), "oauth.google.start", attribute.String("oauth.provider", "google"))
	defer span.End()
	r = r.WithContext(ctx)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload struct {
		ReturnURL string `json:"return_url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	provider, err := s.googleProvider(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusPreconditionRequired, "tenant.google_provider_required")
		return
	}
	url, state, err := (google.Client{ClientID: provider.ClientID, RedirectURI: "https://" + s.installHost + "/api/v1/auth/google/callback", Secret: s.GoogleSecret, AuthURL: s.GoogleAuthURL}).AuthCodeURL(tenant.ID, payload.ReturnURL, 10*time.Minute)
	if err != nil {
		observability.RecordError(span, err)
		writeError(w, http.StatusInternalServerError, "auth.google_start_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": url, "nonce": state.Nonce})
}

func (s *Server) authGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx, span := observability.StartSpan(r.Context(), "oauth.google.callback", attribute.String("oauth.provider", "google"))
	defer span.End()
	r = r.WithContext(ctx)
	var payload struct {
		State   string `json:"state"`
		IDToken string `json:"id_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	if _, err := s.googleProvider(r.Context(), tenant.ID); err != nil {
		writeError(w, http.StatusPreconditionRequired, "tenant.google_provider_required")
		return
	}
	state, err := (google.Client{Secret: s.GoogleSecret}).ValidateState(payload.State, time.Now().UTC())
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("google", "fail")
		writeError(w, http.StatusBadRequest, "auth.google_invalid")
		return
	}
	if !ok || tenant.ID != state.TenantID {
		writeError(w, http.StatusBadRequest, "auth.google_invalid")
		return
	}
	if err := google.ValidateIDTokenNonce(payload.IDToken, state.Nonce); err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("google", "fail")
		writeError(w, http.StatusBadRequest, "auth.google_invalid")
		return
	}
	claims, err := google.ParseIDTokenClaims(payload.IDToken)
	if err != nil || strings.TrimSpace(claims.Email) == "" {
		observability.RecordError(span, err)
		recordAuthAttempt("google", "fail")
		writeError(w, http.StatusBadRequest, "auth.google_invalid")
		return
	}
	userID, err := s.findOrCreateUpstreamUser(r.Context(), tenant.ID, claims)
	if err != nil {
		observability.RecordError(span, err)
		recordAuthAttempt("google", "fail")
		if isSignupError(err) {
			writeError(w, signupHTTPStatus(err), err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "auth.google_user_failed")
		return
	}
	sessionID, err := s.establishUserSession(w, r, tenant.ID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	recordAuthAttempt("google", "success")
	writeJSON(w, http.StatusOK, map[string]any{"tenant_id": state.TenantID, "return_url": state.ReturnURL, "user_id": userID, "session_id": sessionID})
}

type googleProviderConfig struct {
	ClientID     string
	ClientSecret string
}

func (s *Server) googleProvider(ctx context.Context, tenantID uuid.UUID) (googleProviderConfig, error) {
	var rows []struct {
		ClientID     []byte `gorm:"column:client_id_encrypted"`
		ClientSecret []byte `gorm:"column:client_secret_encrypted"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT client_id_encrypted, client_secret_encrypted FROM upstream_providers WHERE tenant_id = ? AND kind = 'google' AND enabled = true LIMIT 1`, tenantID, &rows, tenantID); err != nil {
		return googleProviderConfig{}, err
	}
	if len(rows) == 0 {
		return googleProviderConfig{}, sql.ErrNoRows
	}
	clientID, err := s.decryptProviderSecret(rows[0].ClientID)
	if err != nil {
		return googleProviderConfig{}, err
	}
	clientSecret, err := s.decryptProviderSecret(rows[0].ClientSecret)
	if err != nil {
		return googleProviderConfig{}, err
	}
	return googleProviderConfig{ClientID: string(clientID), ClientSecret: string(clientSecret)}, nil
}

func (s *Server) findOrCreateUpstreamUser(ctx context.Context, tenantID uuid.UUID, claims google.IDTokenClaims) (uuid.UUID, error) {
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenantID, &rows, tenantID, claims.Email); err != nil {
		return uuid.Nil, err
	}
	if len(rows) > 0 {
		return rows[0].ID, nil
	}
	if err := authpolicy.CheckSignup(ctx, s.DB, tenantID, authpolicy.SignupContext{Method: authpolicy.MethodGoogle, Email: claims.Email}); err != nil {
		return uuid.Nil, err
	}
	userID := uuid.New()
	metadata, _ := json.Marshal(map[string]string{"google_sub": claims.Subject, "name": claims.Name})
	if err := s.TenantDB.Transaction(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES (?, ?, ?, ?::jsonb)`, userID, tenantID, claims.Email, string(metadata)).Error
	}); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (s *Server) decodeWithBot(w http.ResponseWriter, r *http.Request, dest interface{ GetBotToken() string }) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return false
	}
	if err := s.BotVerifier.Verify(r.Context(), dest.GetBotToken()); err != nil {
		recordAuthAttempt(authKindFromPath(r.URL.Path), "bot_blocked")
		s.recordBotMitigationAudit(r)
		writeError(w, http.StatusForbidden, "bot.verify_failed")
		return false
	}
	return true
}

func (s *Server) recordBotMitigationAudit(r *http.Request) {
	entry := audit.Entry{ActorKind: "system", Action: "bot.verify_failed", ResourceKind: "bot_mitigation", StateAfter: map[string]string{"path": r.URL.Path}}
	if tenant, ok := TenantFromContext(r.Context()); ok {
		entry.TenantID = &tenant.ID
	}
	s.recordMutationAudit(r.Context(), entry)
}

func (s *Server) decodeCredentialPayload(w http.ResponseWriter, r *http.Request) (Tenant, credentialPayload, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return Tenant{}, credentialPayload{}, false
	}
	var payload credentialPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return Tenant{}, credentialPayload{}, false
	}
	return tenant, payload, true
}

func webauthnResponseRequest(r *http.Request, raw json.RawMessage) *http.Request {
	clone := r.Clone(r.Context())
	clone.Body = http.NoBody
	if len(raw) != 0 {
		clone.Body = ioNopCloser{bytes.NewReader(raw)}
		clone.ContentLength = int64(len(raw))
	}
	return clone
}

type ioNopCloser struct{ *bytes.Reader }

func (c ioNopCloser) Close() error { return nil }

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

func (s *Server) authLogout(w http.ResponseWriter, r *http.Request) {
	// Revoke the session row in the DB so a leftover cookie cannot resurrect the
	// session. The cypra_session cookie is shared between user and instance-admin
	// flows (different storage tables), so revoke from both.
	if cookie, err := r.Cookie("cypra_session"); err == nil {
		if sessionID, perr := uuid.Parse(cookie.Value); perr == nil {
			_, _ = s.DB.ExecContext(r.Context(), `UPDATE sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, sessionID)
			_, _ = s.DB.ExecContext(r.Context(), `UPDATE instance_admin_sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, sessionID)
		}
	}
	for _, name := range []string{"cypra_session", "cypra_csrf"} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   r.TLS != nil,
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

// authRevokeOtherSessions revokes every active session for the authenticated
// user except the one making the request. The acting user is identified by the
// X-Cypra-User-Id header (set by auth.MiddlewareWithPAT); the current session is
// identified by either the X-Cypra-Session-Id header or the cypra_session cookie.
// Instance admins revoke their parallel session table.
func (s *Server) authRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	currentSession := r.Header.Get("X-Cypra-Session-Id")
	if currentSession == "" {
		if cookie, cerr := r.Cookie("cypra_session"); cerr == nil {
			currentSession = cookie.Value
		}
	}
	if actor.InstanceAdmin && actor.InstanceAdminID != uuid.Nil {
		if _, err := s.DB.ExecContext(
			r.Context(),
			`UPDATE instance_admin_sessions SET revoked_at = now() WHERE instance_admin_id = $1 AND revoked_at IS NULL AND id::text <> $2`,
			actor.InstanceAdminID, currentSession,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "auth.session_revoke_failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	userIDStr := r.Header.Get("X-Cypra-User-Id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		userID = actor.UserID
	}
	if userID == uuid.Nil {
		writeError(w, http.StatusUnauthorized, "auth.unauthorized")
		return
	}
	if err := s.TenantDB.Transaction(r.Context(), tenant.ID, func(tx *gorm.DB) error {
		return tx.Exec(
			`UPDATE sessions SET revoked_at = now() WHERE tenant_id = ? AND subject_id = ? AND revoked_at IS NULL AND id::text <> ?`,
			tenant.ID, userID, currentSession,
		).Error
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_revoke_failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) rpID(tenant Tenant) string {
	return tenant.Slug + "." + strings.Split(s.installHost, ":")[0]
}
