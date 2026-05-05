package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/oidc"
)

func (s *Server) oidcDiscovery(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	issuer := "https://" + tenant.Slug + "." + strings.Split(s.installHost, ":")[0]
	w.Header().Set("Cache-Control", "public, max-age=600, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oidc/authorize",
		"token_endpoint":                        issuer + "/oidc/token",
		"userinfo_endpoint":                     issuer + "/oidc/userinfo",
		"revocation_endpoint":                   issuer + "/oidc/revoke",
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"id_token_signing_alg_values_supported": []string{"RS256", "ES256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
	})
}

func (s *Server) oidcJWKS(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	keys, err := s.oidcProvider().SigningKeys(r.Context(), tenant.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error")
		return
	}
	rawKeys := make([]json.RawMessage, 0, len(keys))
	for _, key := range keys {
		rawKeys = append(rawKeys, key.PublicJWK)
	}
	w.Header().Set("Cache-Control", "public, max-age=300, must-revalidate")
	writeJSON(w, http.StatusOK, map[string]any{"keys": rawKeys})
}

func (s *Server) oidcAuthorize(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeOIDCError(w, http.StatusNotFound, "invalid_request")
		return
	}
	query := r.URL.Query()
	userID, err := uuid.Parse(query.Get("user_id"))
	if err != nil {
		writeOIDCError(w, http.StatusUnauthorized, "login_required")
		return
	}
	code, err := s.oidcProvider().Authorize(r.Context(), oidc.AuthorizeRequest{
		TenantID:            tenant.ID,
		UserID:              userID,
		ClientID:            query.Get("client_id"),
		RedirectURI:         query.Get("redirect_uri"),
		Scope:               strings.Fields(query.Get("scope")),
		State:               query.Get("state"),
		Nonce:               query.Get("nonce"),
		CodeChallenge:       query.Get("code_challenge"),
		CodeChallengeMethod: query.Get("code_challenge_method"),
	})
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	redirectURL, _ := url.Parse(query.Get("redirect_uri"))
	values := redirectURL.Query()
	values.Set("code", code)
	if state := query.Get("state"); state != "" {
		values.Set("state", state)
	}
	redirectURL.RawQuery = values.Encode()
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) oidcToken(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeOIDCError(w, http.StatusNotFound, "invalid_request")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		writeOIDCError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	clientID, clientSecret := clientAuth(r)
	if clientID == "" {
		clientID = r.Form.Get("client_id")
		clientSecret = r.Form.Get("client_secret")
	}
	response, err := s.oidcProvider().Token(r.Context(), oidc.TokenRequest{TenantSlug: tenant.Slug, ClientID: clientID, ClientSecret: clientSecret, GrantType: r.Form.Get("grant_type"), Code: r.Form.Get("code"), RefreshToken: r.Form.Get("refresh_token"), RedirectURI: r.Form.Get("redirect_uri"), CodeVerifier: r.Form.Get("code_verifier")})
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) oidcUserInfo(w http.ResponseWriter, r *http.Request) {
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeOIDCError(w, http.StatusNotFound, "invalid_request")
		return
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	claims, err := s.oidcProvider().UserInfo(r.Context(), tenant.Slug, tenant.ID, token)
	if err != nil {
		writeOIDCError(w, http.StatusUnauthorized, "invalid_token")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, claims)
}

func (s *Server) oidcRevoke(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		writeOIDCError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := s.oidcProvider().Revoke(r.Context(), r.Form.Get("token")); err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) oidcConsent(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		s.hostedConsentDecision(w, r)
		return
	}
	tenant, ok := TenantFromContext(r.Context())
	if !ok {
		writeOIDCError(w, http.StatusNotFound, "invalid_request")
		return
	}
	var payload struct {
		UserID   string   `json:"user_id"`
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeOIDCError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	client, err := s.oidcProvider().LoadClient(r.Context(), tenant.ID, payload.ClientID)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	if err := s.oidcProvider().RecordConsent(r.Context(), tenant.ID, userID, client.ID, payload.Scopes); err != nil {
		writeOIDCError(w, http.StatusInternalServerError, "server_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) oidcProvider() oidc.Provider {
	return oidc.Provider{DB: s.DB, KEK: s.MasterKey, InstallDomain: strings.Split(s.installHost, ":")[0]}
}

func clientAuth(r *http.Request) (string, string) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Basic ") {
		return "", ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authz, "Basic "))
	if err != nil {
		return "", ""
	}
	clientID, secret, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", ""
	}
	return clientID, secret
}

func oidcErrorCode(err error) string {
	if err == nil {
		return "server_error"
	}
	return err.Error()
}

func writeOIDCError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
