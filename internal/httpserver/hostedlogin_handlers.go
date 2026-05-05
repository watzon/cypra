package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/magiclink"
	passwordauth "github.com/watzon/cypra/internal/auth/password"
	"github.com/watzon/cypra/internal/hostedlogin"
)

const uniformAuthError = "Sign in didn't work. Check your details and try again."

func (s *Server) hostedPage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) hostedPasswordSignIn(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	var userID uuid.UUID
	err := s.DB.QueryRowContext(r.Context(), `SELECT id FROM users WHERE tenant_id = $1 AND email = $2 AND deleted_at IS NULL`, tenant.ID, r.FormValue("email")).Scan(&userID)
	if err != nil || (passwordauth.Service{DB: s.DB}).Verify(r.Context(), userID, r.FormValue("password")) != nil {
		writeHTMXError(w, http.StatusUnauthorized, uniformAuthError)
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
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	if _, err := (magiclink.Service{DB: s.DB}).Issue(r.Context(), tenant.ID, nil, r.FormValue("email"), "", 15*time.Minute); err != nil {
		writeHTMXError(w, http.StatusInternalServerError, uniformAuthError)
		return
	}
	writeHTMXMessage(w, "Check your email.")
}

func (s *Server) hostedSignup(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeHTMXError(w, http.StatusNotFound, "Tenant not found.")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	userID := uuid.New()
	if _, err := s.DB.ExecContext(r.Context(), `INSERT INTO users (id, tenant_id, email, metadata) VALUES ($1, $2, $3, '{}'::jsonb)`, userID, tenant.ID, r.FormValue("email")); err != nil {
		writeHTMXError(w, http.StatusConflict, uniformAuthError)
		return
	}
	if err := (passwordauth.Service{DB: s.DB}).SetPassword(r.Context(), tenant.ID, userID, r.FormValue("password")); err != nil {
		writeHTMXError(w, http.StatusBadRequest, uniformAuthError)
		return
	}
	writeHTMXMessage(w, "Account created. Continue to sign in.")
}

func (s *Server) hostedFactor(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
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
	var tenantName string
	var branding []byte
	if err := s.DB.QueryRowContext(r.Context(), `SELECT name, branding FROM tenants WHERE id = $1 AND deleted_at IS NULL`, tenant.ID).Scan(&tenantName, &branding); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "tenant.not_found")
			return hostedlogin.PageData{}, false
		}
		writeError(w, http.StatusInternalServerError, "db.error")
		return hostedlogin.PageData{}, false
	}
	theme, err := hostedlogin.ThemeFromJSON(tenant.Slug, tenantName, branding)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "tenant.branding_invalid")
		return hostedlogin.PageData{}, false
	}
	return hostedlogin.PageData{Title: hostedTitle(name), Route: name, Theme: theme, RPID: s.rpID(tenant)}, true
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
		return "Consent"
	case "error":
		return "Sign-in error"
	default:
		return "Sign in"
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
	_, _ = w.Write([]byte(`<p>` + message + `</p>`))
}

func writeHTMXError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`<p class="error">` + message + `</p>`))
}
