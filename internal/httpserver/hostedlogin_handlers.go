package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/auth/invite"
	"github.com/watzon/cypra/internal/auth/magiclink"
	passwordauth "github.com/watzon/cypra/internal/auth/password"
	"github.com/watzon/cypra/internal/auth/totp"
	"github.com/watzon/cypra/internal/auth/upstream"
	"github.com/watzon/cypra/internal/auth/upstream/google"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/authpolicy"
	"github.com/watzon/cypra/internal/hostedlogin"
	"gorm.io/gorm"
)

const uniformAuthError = "Sign in didn't work. Check your details and try again."

func (s *Server) hostedPage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostHeader := s.requestHost(r)
		host := strings.Split(hostHeader, ":")[0]
		installHost := strings.Split(s.installHost, ":")[0]
		if host == installHost {
			// /login on the install host renders the instance-admin sign-in form.
			// Other hosted pages have no install-host equivalent (they need a
			// tenant) so we fall back to the dashboard root.
			if name == "login" {
				s.renderInstanceAdminLogin(w, r)
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		data, ok := s.hostedPageData(w, r, name)
		if !ok {
			return
		}
		if name == "consent" {
			data.Scopes = defaultScopes(r.URL.Query().Get("scope"), r.URL.Query().Get("upgraded") == "true")
		}
		if name == "error" {
			data.Error = r.URL.Query().Get("message")
		}
		if name == "reset" {
			data.ResetToken = r.URL.Query().Get("token")
		}
		if err := s.hostedRenderer().Render(w, name, data); err != nil {
			writeError(w, http.StatusInternalServerError, "hosted_login.render_failed")
		}
	}
}

func (s *Server) hostedStaticPasskey(w http.ResponseWriter, _ *http.Request) {
	content, err := hostedlogin.PasskeyJS()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "static.unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(content)
}

func (s *Server) hostedStaticInstanceAdminLogin(w http.ResponseWriter, _ *http.Request) {
	content, err := hostedlogin.InstanceAdminLoginJS()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "static.unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(content)
}

func (s *Server) hostedStaticHTMX(w http.ResponseWriter, _ *http.Request) {
	content, err := hostedlogin.HTMXJS()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "static.unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(content)
}

func (s *Server) renderInstanceAdminLogin(w http.ResponseWriter, r *http.Request) {
	data := hostedlogin.PageData{
		Title: "Sign in",
		Route: "instance_admin_login",
		Theme: hostedlogin.Theme{DisplayName: "Cypra", Accent: "#0F766E", PoweredBy: false},
		RPID:  s.installRPID(),
		Email: r.URL.Query().Get("email"),
	}
	if err := s.hostedRenderer().Render(w, "instance_admin_login", data); err != nil {
		writeError(w, http.StatusInternalServerError, "hosted_login.render_failed")
	}
}

func (s *Server) hostedInvite(w http.ResponseWriter, r *http.Request) {
	data, ok := s.hostedPageData(w, r, "invite")
	if !ok {
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		data.Error = "Invite link is missing."
	} else {
		continuation, err := (invite.Service{DB: s.DB}).StartContinuation(r.Context(), token, 15*time.Minute)
		if err != nil {
			data.Error = "Invite link is invalid or expired."
		} else {
			data.InviteID = continuation.ID.String()
			s.setSessionCookie(w, r, "cypra_invite_continuation", continuation.ID.String(), 15*time.Minute)
		}
	}
	if err := s.hostedRenderer().Render(w, "invite", data); err != nil {
		writeError(w, http.StatusInternalServerError, "hosted_login.render_failed")
	}
}

func (s *Server) hostedPasswordSignIn(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeHTMXError(w, http.StatusForbidden, "Tenant is suspended.")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	if !s.allowHostedRate(w, r, &tenant.ID, "login:ip", s.clientRateKey(r), 10, time.Minute) || !s.allowHostedRate(w, r, &tenant.ID, "login:account", r.FormValue("email"), 5, time.Minute) {
		return
	}
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	err := s.TenantDB.RawScan(r.Context(), `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenant.ID, &rows, tenant.ID, r.FormValue("email"))
	if err != nil || len(rows) == 0 || (passwordauth.Service{DB: s.DB}).Verify(r.Context(), rows[0].ID, r.FormValue("password")) != nil {
		writeHTMXError(w, http.StatusUnauthorized, uniformAuthError)
		return
	}
	if _, err := s.establishUserSession(w, r, tenant.ID, rows[0].ID); err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	if token := strings.TrimSpace(r.FormValue("continue")); token != "" {
		w.Header().Set("HX-Redirect", "/oidc/authorize?continue="+url.QueryEscape(token))
		writeHTMXMessage(w, "Signed in. Returning to the application.")
		return
	}
	writeHTMXMessage(w, "Signed in. Continue to your application.")
}

func (s *Server) hostedMagicLink(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeHTMXError(w, http.StatusForbidden, "Tenant is suspended.")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	if !s.allowHostedRate(w, r, &tenant.ID, "magic_link:account", r.FormValue("email"), 5, time.Hour) {
		return
	}
	if !s.requireEmailProvider(w, r, tenant.ID) {
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	userID, err := s.userIDByEmail(r.Context(), tenant.ID, email)
	if err != nil {
		writeHTMXMessage(w, "Check your email.")
		return
	}
	policy, err := authpolicy.LoadMagicLink(r.Context(), s.DB, tenant.ID)
	if err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	if _, err := (magiclink.Service{DB: s.DB}).Issue(r.Context(), tenant.ID, &userID, email, "", policy); err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	writeHTMXMessage(w, "Check your email.")
}

func (s *Server) hostedPasswordReset(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeHTMXError(w, http.StatusForbidden, "Tenant is suspended.")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	token := strings.TrimSpace(r.FormValue("token"))
	password := r.FormValue("password")
	if token != "" && password != "" {
		if _, err := (passwordauth.Service{DB: s.DB}).ResetPassword(r.Context(), token, password); err != nil {
			writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
			return
		}
		writeHTMXMessage(w, "Password reset. Return to sign in.")
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	if email == "" {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	if !s.allowHostedRate(w, r, &tenant.ID, "password_reset:account", email, 5, time.Hour) {
		return
	}
	if !s.requireEmailProvider(w, r, tenant.ID) {
		return
	}
	var rows []struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT id FROM users WHERE tenant_id = ? AND email = ? AND deleted_at IS NULL`, tenant.ID, &rows, tenant.ID, email); err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	if len(rows) > 0 {
		resetToken, err := (passwordauth.Service{DB: s.DB}).IssueResetToken(r.Context(), tenant.ID, rows[0].ID, 15*time.Minute)
		if err != nil {
			writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
			return
		}
		resetURL := "https://" + strings.TrimRight(r.Host, "/") + "/reset?token=" + url.QueryEscape(resetToken)
		payload, _ := json.Marshal(map[string]string{"reset_url": resetURL})
		if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO email_outbox (tenant_id, to_address, template, payload) VALUES ($1, $2, 'password-reset', $3)`, tenant.ID, email, payload); err != nil {
			writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
			return
		}
	}
	writeHTMXMessage(w, "If that account exists, check your email.")
}

func (s *Server) hostedSignup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeHTMXError(w, http.StatusForbidden, "Tenant is suspended.")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	if !s.allowHostedRate(w, r, &tenant.ID, "signup:ip", s.clientRateKey(r), 5, time.Minute) {
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	method := strings.TrimSpace(r.FormValue("method"))
	if method == "" {
		method = authpolicy.MethodPassword
	}
	switch method {
	case authpolicy.MethodPassword:
		s.hostedSignupPassword(w, r, tenant.ID, email)
	case authpolicy.MethodMagicLink:
		s.hostedSignupMagicLink(w, r, tenant.ID, email)
	default:
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
	}
}

func (s *Server) hostedSignupPassword(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, email string) {
	if err := authpolicy.CheckSignup(r.Context(), s.DB, tenantID, authpolicy.SignupContext{Method: authpolicy.MethodPassword, Email: email}); err != nil {
		writeHTMXError(w, signupHTTPStatus(err), err.Error())
		return
	}
	userID, err := s.createSignupUser(r.Context(), tenantID, email)
	if err != nil {
		writeHTMXError(w, http.StatusConflict, uniformAuthError)
		return
	}
	if err := (passwordauth.Service{DB: s.DB}).SetPassword(r.Context(), tenantID, userID, r.Form.Get("password")); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	writeHTMXMessage(w, "Account created. Continue to sign in.")
}

func (s *Server) hostedSignupMagicLink(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, email string) {
	if err := authpolicy.CheckSignup(r.Context(), s.DB, tenantID, authpolicy.SignupContext{Method: authpolicy.MethodMagicLink, Email: email}); err != nil {
		writeHTMXError(w, signupHTTPStatus(err), err.Error())
		return
	}
	if !s.requireEmailProvider(w, r, tenantID) {
		return
	}
	userID, err := s.createSignupUser(r.Context(), tenantID, email)
	if err != nil {
		writeHTMXError(w, http.StatusConflict, uniformAuthError)
		return
	}
	policy, err := authpolicy.LoadMagicLink(r.Context(), s.DB, tenantID)
	if err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	if _, err := (magiclink.Service{DB: s.DB}).Issue(r.Context(), tenantID, &userID, email, "", policy); err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	writeHTMXMessage(w, "Account created. Check your email for a sign-in link.")
}

// createSignupUser inserts a new user row for self-serve registration. Returns
// the assigned user ID. Any insert failure (typically a unique-email conflict)
// is surfaced to the caller as an opaque error.
func (s *Server) createSignupUser(ctx context.Context, tenantID uuid.UUID, email string) (uuid.UUID, error) {
	userID := uuid.New()
	if err := s.TenantDB.Transaction(ctx, tenantID, func(tx *gorm.DB) error {
		return tx.Exec(`INSERT INTO users (id, tenant_id, email, metadata) VALUES (?, ?, ?, '{}'::jsonb)`, userID, tenantID, email).Error
	}); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

// hostedSignupPasskey performs the two-phase WebAuthn registration ceremony
// for self-serve passkey signup. Phase 1 (no ceremony_id) creates the user and
// returns registration options; phase 2 finishes the ceremony and establishes
// a session.
func (s *Server) hostedSignupPasskey(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeError(w, http.StatusForbidden, "tenant.suspended")
		return
	}
	if !s.allowRate(w, r, &tenant.ID, "signup:ip", s.clientRateKey(r), 5, time.Minute) {
		return
	}
	var payload struct {
		Email        string          `json:"email"`
		CeremonyID   string          `json:"ceremony_id"`
		UserID       string          `json:"user_id"`
		Response     json.RawMessage `json:"response"`
		ContinueURL  string          `json:"continue"`
		CredentialID string          `json:"credential_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	service := webauthn.Service{DB: s.DB}
	if strings.TrimSpace(payload.CeremonyID) == "" {
		email := strings.TrimSpace(payload.Email)
		if err := authpolicy.CheckSignup(r.Context(), s.DB, tenant.ID, authpolicy.SignupContext{Method: authpolicy.MethodPasskey, Email: email}); err != nil {
			writeError(w, signupHTTPStatus(err), err.Error())
			return
		}
		userID, err := s.createSignupUser(r.Context(), tenant.ID, email)
		if err != nil {
			writeError(w, http.StatusConflict, "user.conflict")
			return
		}
		options, ceremonyID, err := service.BeginRegistration(r.Context(), webauthn.BeginRegistrationRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant), UserName: email})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "auth.passkey_register_failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ceremony_id": ceremonyID, "user_id": userID, "options": options})
		return
	}
	ceremonyID, err := uuid.Parse(payload.CeremonyID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "webauthn.ceremony_invalid")
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "user.id_invalid")
		return
	}
	if _, err := service.FinishRegistration(r.Context(), webauthn.FinishRegistrationRequest{TenantID: tenant.ID, UserID: userID, RPID: s.rpID(tenant), Origins: []string{s.requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)}); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.passkey_register_failed")
		return
	}
	if _, err := s.establishUserSession(w, r, tenant.ID, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "auth.session_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user_id": userID, "status": "ok"})
}

// isSignupError reports whether err is one of the authpolicy signup sentinels.
func isSignupError(err error) bool {
	return errors.Is(err, authpolicy.ErrSignupClosed) ||
		errors.Is(err, authpolicy.ErrSignupRestricted) ||
		errors.Is(err, authpolicy.ErrSignupDisabledForMethod) ||
		errors.Is(err, authpolicy.ErrInvitesDisabled)
}

// signupHTTPStatus maps an authpolicy signup error to its HTTP status.
func signupHTTPStatus(err error) int {
	if isSignupError(err) {
		return http.StatusForbidden
	}
	return http.StatusInternalServerError
}

func (s *Server) hostedFactor(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	userID, ok := s.currentUserSession(r, tenant.ID)
	if !ok {
		writeHTMXError(w, http.StatusUnauthorized, uniformAuthError)
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	code := r.FormValue("code")
	switch r.FormValue("factor") {
	case "totp":
		ok, err := (totp.Service{DB: s.DB, KEK: s.MasterKey}).Verify(r.Context(), userID, code, time.Now().UTC())
		if err != nil || !ok {
			writeHTMXError(w, http.StatusUnauthorized, "Invalid two-factor code.")
			return
		}
	case "backup":
		ok, err := (backupcodes.Service{DB: s.DB}).ConsumeForUser(r.Context(), userID, code)
		if err != nil || !ok {
			writeHTMXError(w, http.StatusUnauthorized, "Invalid backup code.")
			return
		}
	case "webauthn":
		writeHTMXError(w, http.StatusBadRequest, "Use the passkey challenge to verify WebAuthn.")
		return
	default:
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	writeHTMXMessage(w, "Factor accepted.")
}

func (s *Server) hostedConsentDecision(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, "Couldn't reach the sign-in service. Try again.")
		return
	}
	if r.FormValue("decision") != "allow" {
		writeHTMXError(w, http.StatusForbidden, "Consent was denied.")
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	token := strings.TrimSpace(r.FormValue("continue"))
	if token == "" {
		writeHTMXError(w, http.StatusBadRequest, "Couldn't reach the sign-in service. Try again.")
		return
	}
	userID, client, scopes, redirectTo, ok := s.resolveConsentPayload(w, r, tenant, token)
	if !ok {
		return
	}
	if err := s.oidcProvider().RecordConsent(r.Context(), tenant.ID, userID, client.ID, scopes); err != nil {
		writeHTMXError(w, http.StatusInternalServerError, "Couldn't save consent. Try again.")
		return
	}
	w.Header().Set("HX-Redirect", redirectTo)
	writeHTMXMessage(w, "Consent recorded. Returning to the application.")
}

func (s *Server) patchTenantBranding(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	var payload hostedlogin.Branding
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if strings.TrimSpace(payload.Accent) != "" {
		if err := hostedlogin.ValidateAccent(payload.Accent); err != nil {
			var validation hostedlogin.AccentValidationError
			if errors.As(err, &validation) {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "tenant.accent_contrast_failed", "pair": validation.Pair, "ratio": validation.Ratio})
				return
			}
			writeError(w, http.StatusBadRequest, "tenant.accent_invalid")
			return
		}
	}
	branding, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "request.invalid")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE tenants SET branding = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`, branding, id); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) hostedRenderer() hostedlogin.Renderer {
	renderer, err := hostedlogin.NewRenderer()
	if err != nil {
		panic(err)
	}
	return renderer
}

func (s *Server) hostedPageData(w http.ResponseWriter, r *http.Request, name string) (hostedlogin.PageData, bool) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return hostedlogin.PageData{}, false
	}
	var rows []struct {
		Name         string  `gorm:"column:name"`
		Branding     []byte  `gorm:"column:branding"`
		LogoObjectID *string `gorm:"column:logo_object_id"`
	}
	if err := s.TenantDB.RawScan(r.Context(), `SELECT name, branding, logo_object_id FROM tenants WHERE id = ? AND deleted_at IS NULL`, tenant.ID, &rows, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return hostedlogin.PageData{}, false
	}
	if len(rows) == 0 {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return hostedlogin.PageData{}, false
	}
	uploadedLogoURL := ""
	if rows[0].LogoObjectID != nil && *rows[0].LogoObjectID != "" {
		uploadedLogoURL = tenantLogoURL(tenant.Slug)
	}
	theme, err := hostedlogin.ThemeFromJSON(tenant.Slug, rows[0].Name, rows[0].Branding, uploadedLogoURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.branding_invalid")
		return hostedlogin.PageData{}, false
	}
	data := hostedlogin.PageData{Title: hostedTitle(name), Route: name, Theme: theme, RPID: s.rpID(tenant), ContinueURL: r.URL.Query().Get("continue")}
	if name == "login" || name == "signup" {
		methods, err := s.tenantEnabledAuthMethods(r.Context(), tenant.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return hostedlogin.PageData{}, false
		}
		data.PasskeyEnabled = methods["passkey"]
		data.PasswordEnabled = methods["password"]
		data.MagicLinkEnabled = methods["magic_link"]
		data.GoogleEnabled = methods["google"]
		social, err := s.loadSocialRows(r.Context(), tenant.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return hostedlogin.PageData{}, false
		}
		oidc, err := s.loadOIDCConnections(r.Context(), tenant.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return hostedlogin.PageData{}, false
		}
		signupOK, err := s.signupCapableMethod(r.Context(), tenant.ID, methods, social, oidc)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db.error")
			return hostedlogin.PageData{}, false
		}
		data.SignupEnabled = signupOK
		anySocial := false
		for _, row := range social {
			if row.Enabled {
				anySocial = true
				break
			}
		}
		anyOIDC := false
		for _, row := range oidc {
			if row.Enabled {
				anyOIDC = true
				break
			}
		}
		data.AnyMethodEnabled = data.PasskeyEnabled || data.PasswordEnabled || data.MagicLinkEnabled || data.GoogleEnabled || methods["totp"] || methods["oidc_upstream"] || anySocial || anyOIDC
		data.PrimaryMethod = s.resolvePrimaryMethod(r.Context(), tenant.ID, methods, social, oidc)
		returnURL := data.ContinueURL
		if returnURL == "" {
			returnURL = "/"
		}
		data.SocialButtons = s.buildSocialButtons(r.Context(), tenant.ID, social, returnURL)
		data.OIDCButtons = s.buildOIDCButtons(r.Context(), tenant.ID, oidc, returnURL)
		if data.GoogleEnabled {
			if provider, err := s.googleProvider(r.Context(), tenant.ID); err == nil {
				googleURL, _, err := (google.Client{ClientID: provider.ClientID, RedirectURI: "https://" + s.installHost + "/api/v1/auth/google/callback", Secret: s.GoogleSecret, AuthURL: s.GoogleAuthURL}).AuthCodeURL(tenant.ID, returnURL, 10*time.Minute)
				if err == nil {
					data.GoogleURL = googleURL
				}
			}
		}
	}
	return data, true
}

func (s *Server) buildSocialButtons(ctx context.Context, _ uuid.UUID, rows []socialDBRow, returnURL string) []hostedlogin.SocialButton {
	_ = ctx
	out := make([]hostedlogin.SocialButton, 0, len(rows))
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		// Skip Google when the legacy upstream_providers row already drives the
		// existing GoogleURL button so we don't render two side-by-side.
		if row.Kind == "google" {
			continue
		}
		display := upstream.Display{Label: titleKind(row.Kind)}
		if p, err := upstream.Resolve(row.Kind); err == nil {
			display = p.Display()
		}
		out = append(out, hostedlogin.SocialButton{
			Kind:    row.Kind,
			Label:   display.Label,
			IconURL: display.IconURL,
			URL:     "/login/social/" + url.PathEscape(row.Kind) + "?continue=" + url.QueryEscape(returnURL),
		})
	}
	return out
}

func (s *Server) buildOIDCButtons(_ context.Context, _ uuid.UUID, rows []oidcDBRow, returnURL string) []hostedlogin.OIDCButton {
	out := make([]hostedlogin.OIDCButton, 0, len(rows))
	for _, row := range rows {
		if !row.Enabled {
			continue
		}
		out = append(out, hostedlogin.OIDCButton{Slug: row.Slug, DisplayName: row.DisplayName, URL: "/login/oidc/" + url.PathEscape(row.Slug) + "?continue=" + url.QueryEscape(returnURL)})
	}
	return out
}

// signupCapableMethod reports whether at least one enabled identifier or
// social/OIDC connection also has allow_signup=true — i.e. whether self-serve
// registration can succeed today. Used to render the "Create account" link.
func (s *Server) signupCapableMethod(ctx context.Context, tenantID uuid.UUID, methods map[string]bool, social []socialDBRow, oidc []oidcDBRow) (bool, error) {
	if methods[authpolicy.MethodPassword] {
		p, err := authpolicy.LoadPassword(ctx, s.DB, tenantID)
		if err != nil {
			return false, err
		}
		if p.AllowsSignup() {
			return true, nil
		}
	}
	if methods[authpolicy.MethodMagicLink] {
		p, err := authpolicy.LoadMagicLink(ctx, s.DB, tenantID)
		if err != nil {
			return false, err
		}
		if p.AllowsSignup() {
			return true, nil
		}
	}
	if methods[authpolicy.MethodPasskey] {
		p, err := authpolicy.LoadPasskey(ctx, s.DB, tenantID)
		if err != nil {
			return false, err
		}
		if p.AllowsSignup() {
			return true, nil
		}
	}
	for _, row := range social {
		if !row.Enabled {
			continue
		}
		policy := unmarshalSocialPolicy(row.Config)
		if policy.AllowsSignup() {
			return true, nil
		}
	}
	for _, row := range oidc {
		if !row.Enabled {
			continue
		}
		var policy authpolicy.OIDCConnectionPolicy
		if len(row.Config) > 0 {
			_ = json.Unmarshal(row.Config, &policy)
		}
		if policy.WithDefaults().AllowsSignup() {
			return true, nil
		}
	}
	return false, nil
}

func (s *Server) resolvePrimaryMethod(ctx context.Context, tenantID uuid.UUID, methods map[string]bool, social []socialDBRow, oidc []oidcDBRow) string {
	configured, err := s.lookupDefaultAuthMethod(ctx, tenantID)
	if err == nil && configured != "" && primaryMethodAvailable(configured, methods, social, oidc) {
		return configured
	}
	for _, candidate := range []string{"passkey", "password", "magic_link"} {
		if methods[candidate] {
			return candidate
		}
	}
	for _, row := range social {
		if row.Enabled {
			return authpolicy.MethodForSocial(row.Kind)
		}
	}
	for _, row := range oidc {
		if row.Enabled {
			return authpolicy.MethodForOIDC(row.Slug)
		}
	}
	for _, candidate := range []string{"google", "totp", "oidc_upstream"} {
		if methods[candidate] {
			return candidate
		}
	}
	return ""
}

func primaryMethodAvailable(method string, methods map[string]bool, social []socialDBRow, oidc []oidcDBRow) bool {
	family, sub := authpolicy.SplitMethod(method)
	switch family {
	case "social":
		for _, row := range social {
			if row.Kind == sub && row.Enabled {
				return true
			}
		}
		return false
	case "oidc":
		for _, row := range oidc {
			if row.Slug == sub && row.Enabled {
				return true
			}
		}
		return false
	default:
		return methods[method]
	}
}

func (s *Server) tenantEnabledAuthMethods(ctx context.Context, tenantID uuid.UUID) (map[string]bool, error) {
	var rows []struct {
		Method  string `gorm:"column:method"`
		Enabled bool   `gorm:"column:enabled"`
	}
	if err := s.TenantDB.RawScan(ctx, `SELECT method::text AS method, enabled FROM tenant_auth_methods WHERE tenant_id = ?`, tenantID, &rows, tenantID); err != nil {
		return nil, err
	}
	methods := make(map[string]bool, len(rows))
	for _, row := range rows {
		methods[row.Method] = row.Enabled
	}
	return methods, nil
}

func hostedTitle(name string) string {
	switch name {
	case "signup":
		return "Create account"
	case "verify":
		return "Verify email"
	case "reset":
		return "Reset password"
	case "2fa":
		return "Two-factor authentication"
	case "consent":
		return "Review requested access"
	case "error":
		return "Sign-in error"
	case "invite":
		return "Accept invite"
	default:
		return "Sign in to continue"
	}
}

func defaultScopes(raw string, upgraded bool) []hostedlogin.Scope {
	descriptions := map[string]string{
		"openid":  "Confirm your identity.",
		"profile": "Share your profile details.",
		"email":   "Share your email address.",
	}
	if strings.TrimSpace(raw) == "" {
		raw = "openid profile email"
	}
	items := []hostedlogin.Scope{}
	for _, scope := range strings.Fields(raw) {
		description := descriptions[scope]
		if description == "" {
			description = "Share this application permission."
		}
		items = append(items, hostedlogin.Scope{Name: scope, Description: description, New: upgraded})
	}
	return items
}

func writeHTMXMessage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<p>` + html.EscapeString(message) + `</p>`))
}

func writeHTMXError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<p class="error">` + html.EscapeString(message) + `</p>`))
}

func titleKind(kind string) string {
	if kind == "" {
		return ""
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}
