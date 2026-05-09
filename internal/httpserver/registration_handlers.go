package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/lib/pq"
	"github.com/watzon/cypra/internal/authpolicy"
)

const allowlistMaxEntries = 256

type registrationSettingsResponse struct {
	Mode           string   `json:"mode"`
	Allowlist      []string `json:"allowlist"`
	InvitesEnabled bool     `json:"invites_enabled"`
}

type registrationSettingsPayload struct {
	Mode           string   `json:"mode"`
	Allowlist      []string `json:"allowlist"`
	InvitesEnabled bool     `json:"invites_enabled"`
}

func (s *Server) getRegistrationSettings(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	settings, err := authpolicy.LoadSignupSettings(r.Context(), s.DB, tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.registration_load_failed")
		return
	}
	writeJSON(w, http.StatusOK, registrationSettingsResponse{
		Mode:           settings.Mode,
		Allowlist:      settings.Allowlist,
		InvitesEnabled: settings.InvitesEnabled,
	})
}

func (s *Server) saveRegistrationSettings(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	var payload registrationSettingsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.payload_invalid")
		return
	}
	mode, err := normalizeRegistrationMode(payload.Mode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.registration_mode_invalid")
		return
	}
	allowlist, err := normalizeAllowlist(payload.Allowlist)
	if err != nil {
		writeError(w, http.StatusBadRequest, "auth_providers.registration_allowlist_invalid")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(), `UPDATE tenants SET signup_mode = $1::tenant_signup_mode, signup_allowlist = $2, invites_enabled = $3 WHERE id = $4`, mode, pq.StringArray(allowlist), payload.InvitesEnabled, tenant.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "auth_providers.registration_save_failed")
		return
	}
	writeJSON(w, http.StatusOK, registrationSettingsResponse{Mode: mode, Allowlist: allowlist, InvitesEnabled: payload.InvitesEnabled})
}

func normalizeRegistrationMode(raw string) (string, error) {
	switch raw {
	case authpolicy.SignupModeOpen, authpolicy.SignupModeRestricted, authpolicy.SignupModeClosed:
		return raw, nil
	default:
		return "", errInvalidMode
	}
}

func normalizeAllowlist(input []string) ([]string, error) {
	if len(input) > allowlistMaxEntries {
		return nil, errAllowlistTooLarge
	}
	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, raw := range input {
		entry := authpolicy.NormalizeAllowlistEntry(raw)
		if entry == "" {
			continue
		}
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		out = append(out, entry)
	}
	return out, nil
}

var (
	errInvalidMode       = jsonErr("invalid mode")
	errAllowlistTooLarge = jsonErr("allowlist too large")
)

type jsonErr string

func (e jsonErr) Error() string { return string(e) }
