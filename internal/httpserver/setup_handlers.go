package httpserver

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/backupcodes"
	"github.com/watzon/cypra/internal/auth/webauthn"
	"github.com/watzon/cypra/internal/bootstrap"
	"github.com/watzon/cypra/internal/hostedlogin"
)

type setupTokenPayload struct {
	Token string `json:"token"`
}

type setupCompletePayload struct {
	Token       string          `json:"token"`
	Email       string          `json:"email"`
	DisplayName string          `json:"display_name"`
	CeremonyID  string          `json:"ceremony_id"`
	Response    json.RawMessage `json:"response"`
}

func (s *Server) setupVerify(w http.ResponseWriter, r *http.Request) {
	var payload setupTokenPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "setup.payload_invalid")
		return
	}
	if err := bootstrap.NewService(s.DB, s.Logger).ValidateSetupToken(r.Context(), strings.TrimSpace(payload.Token)); err != nil {
		if errors.Is(err, bootstrap.ErrInvalidSetupToken) {
			writeError(w, http.StatusUnauthorized, "setup.token_invalid")
			return
		}
		writeError(w, http.StatusInternalServerError, "setup.verify_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) setupPasskeyBegin(w http.ResponseWriter, r *http.Request) {
	var payload setupCompletePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "setup.payload_invalid")
		return
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "setup.email_invalid")
		return
	}
	if err := bootstrap.NewService(s.DB, s.Logger).ValidateSetupToken(r.Context(), strings.TrimSpace(payload.Token)); err != nil {
		if errors.Is(err, bootstrap.ErrInvalidSetupToken) {
			writeError(w, http.StatusUnauthorized, "setup.token_invalid")
			return
		}
		writeError(w, http.StatusInternalServerError, "setup.verify_failed")
		return
	}
	displayName := strings.TrimSpace(payload.DisplayName)
	if displayName == "" {
		displayName = email
	}
	adminID := uuid.New()
	options, ceremonyID, err := (webauthn.Service{DB: s.DB}).BeginInstanceAdminRegistration(r.Context(), webauthn.BeginInstanceAdminRegistrationRequest{InstanceAdminID: adminID, RPID: s.installRPID(), Email: email, DisplayName: displayName})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup.passkey_begin_failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"admin_id": adminID, "ceremony_id": ceremonyID, "options": options})
}

func (s *Server) setupComplete(w http.ResponseWriter, r *http.Request) {
	var payload setupCompletePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "setup.payload_invalid")
		return
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "setup.email_invalid")
		return
	}
	displayName := strings.TrimSpace(payload.DisplayName)
	if displayName == "" {
		displayName = email
	}
	ceremonyID, err := uuid.Parse(strings.TrimSpace(payload.CeremonyID))
	if err != nil || len(payload.Response) == 0 {
		writeError(w, http.StatusBadRequest, "setup.passkey_required")
		return
	}
	webauthnService := webauthn.Service{DB: s.DB}
	registration, err := webauthnService.FinishInstanceAdminRegistration(r.Context(), webauthn.FinishInstanceAdminRegistrationRequest{RPID: s.installRPID(), Origins: []string{requestOrigin(r)}, CeremonyID: ceremonyID, Response: webauthnResponseRequest(r, payload.Response)})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "setup.passkey_invalid")
		return
	}
	service := bootstrap.NewService(s.DB, s.Logger)
	setupTokenID, err := service.RedeemSetupToken(r.Context(), strings.TrimSpace(payload.Token))
	if err != nil {
		if errors.Is(err, bootstrap.ErrInvalidSetupToken) {
			writeError(w, http.StatusUnauthorized, "setup.token_invalid")
			return
		}
		writeError(w, http.StatusInternalServerError, "setup.redeem_failed")
		return
	}
	adminID, err := service.CreateFirstInstanceAdminWithID(r.Context(), registration.InstanceAdminID, setupTokenID, email, displayName)
	if err != nil {
		if errors.Is(err, bootstrap.ErrInvalidSetupToken) {
			writeError(w, http.StatusUnauthorized, "setup.token_invalid")
			return
		}
		if errors.Is(err, bootstrap.ErrBootstrapUnavailable) {
			writeError(w, http.StatusConflict, "setup.unavailable")
			return
		}
		writeError(w, http.StatusInternalServerError, "setup.admin_create_failed")
		return
	}
	if err := webauthnService.StoreInstanceAdminCredential(r.Context(), adminID, s.installHost, registration.Credential); err != nil {
		writeError(w, http.StatusInternalServerError, "setup.passkey_store_failed")
		return
	}
	codes, err := (backupcodes.Service{DB: s.DB}).RegenerateForInstanceAdmin(r.Context(), adminID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup.backup_codes_failed")
		return
	}
	if _, err := s.establishInstanceAdminSession(w, r, adminID); err != nil {
		writeError(w, http.StatusInternalServerError, "setup.session_failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"admin_id": adminID, "backup_codes": codes})
}

// setupInstructionsPage explains to a fresh-install operator how to mint a
// setup token. It deliberately does NOT mint a token itself: the token is a
// bearer secret that grants instance-admin creation, and we only emit it via
// the CLI where access implies ownership of the host. If an instance admin
// already exists, we redirect anyone who lands here away from setup.
func (s *Server) setupInstructionsPage(w http.ResponseWriter, r *http.Request) {
	if s.hasInstanceAdmin(r.Context()) {
		s.redirectAfterSetup(w, r)
		return
	}
	data := hostedlogin.PageData{
		Title: "Set up Cypra",
		Route: "setup_instructions",
		Theme: hostedlogin.Theme{DisplayName: "Cypra", Accent: "#0F766E", PoweredBy: false},
	}
	if err := s.hostedRenderer().Render(w, "setup_instructions", data); err != nil {
		writeError(w, http.StatusInternalServerError, "hosted_login.render_failed")
	}
}

func (s *Server) installRPID() string {
	host, _, err := net.SplitHostPort(s.installHost)
	if err == nil {
		return host
	}
	return strings.Split(s.installHost, ":")[0]
}

func requestOrigin(r *http.Request) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	return scheme + "://" + r.Host
}
