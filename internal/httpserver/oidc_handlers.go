package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/oidc"
)

const oidcAuthorizeContinuationTTL = 10 * time.Minute

var errOIDCContinuationInvalid = errors.New("oidc authorize continuation invalid")

type oidcAuthorizeContinuation struct {
	TenantID            string   `json:"tenant_id"`
	ClientID            string   `json:"client_id"`
	RedirectURI         string   `json:"redirect_uri"`
	Scope               []string `json:"scope"`
	State               string   `json:"state,omitempty"`
	Nonce               string   `json:"nonce,omitempty"`
	CodeChallenge       string   `json:"code_challenge"`
	CodeChallengeMethod string   `json:"code_challenge_method"`
	ExpiresAt           int64    `json:"expires_at"`
}

func (s *Server) oidcDiscovery(w http.ResponseWriter, r *http.Request) {
	_, ok := TenantFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusNotFound, "tenant.not_found")
		return
	}
	issuer := requestOrigin(r)
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
	if blocked, err := s.tenantAuthBlocked(r.Context(), tenant.ID); err != nil || blocked {
		writeOIDCError(w, http.StatusForbidden, "access_denied")
		return
	}
	query := r.URL.Query()
	if token := query.Get("continue"); token != "" {
		continuation, err := s.parseOIDCAuthorizeContinuation(token)
		if err != nil || continuation.TenantID != tenant.ID.String() {
			writeOIDCError(w, http.StatusBadRequest, "invalid_request")
			return
		}
		userID, ok := s.currentUserSession(r, tenant.ID)
		if !ok {
			http.Redirect(w, r, "/login?continue="+url.QueryEscape(token), http.StatusFound)
			return
		}
		s.finishOIDCAuthorize(w, r, tenant, userID, continuation.authorizeRequest(tenant.ID))
		return
	}
	authorizeReq := oidc.AuthorizeRequest{
		TenantID:            tenant.ID,
		ClientID:            query.Get("client_id"),
		RedirectURI:         query.Get("redirect_uri"),
		Scope:               strings.Fields(query.Get("scope")),
		State:               query.Get("state"),
		Nonce:               query.Get("nonce"),
		CodeChallenge:       query.Get("code_challenge"),
		CodeChallengeMethod: query.Get("code_challenge_method"),
	}
	if _, err := s.oidcProvider().ValidateAuthorizeRequest(r.Context(), authorizeReq); err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	userID, ok := s.currentUserSession(r, tenant.ID)
	if !ok {
		token, err := s.issueOIDCAuthorizeContinuation(tenant.ID, authorizeReq)
		if err != nil {
			writeOIDCError(w, http.StatusInternalServerError, "server_error")
			return
		}
		http.Redirect(w, r, "/login?continue="+url.QueryEscape(token), http.StatusFound)
		return
	}
	s.finishOIDCAuthorize(w, r, tenant, userID, authorizeReq)
}

func (s *Server) finishOIDCAuthorize(w http.ResponseWriter, r *http.Request, tenant Tenant, userID uuid.UUID, req oidc.AuthorizeRequest) {
	client, err := s.oidcProvider().ValidateAuthorizeRequest(r.Context(), req)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	covered, existing, err := s.oidcProvider().ConsentCovers(r.Context(), tenant.ID, userID, client.ID, req.Scope)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, "server_error")
		return
	}
	if !covered {
		token, err := s.issueOIDCAuthorizeContinuation(tenant.ID, req)
		if err != nil {
			writeOIDCError(w, http.StatusInternalServerError, "server_error")
			return
		}
		values := url.Values{"continue": {token}, "scope": {strings.Join(req.Scope, " ")}}
		if existing {
			values.Set("upgraded", "true")
		}
		http.Redirect(w, r, "/oidc/consent?"+values.Encode(), http.StatusFound)
		return
	}
	req.UserID = userID
	code, err := s.oidcProvider().Authorize(r.Context(), req)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	redirectURL, _ := url.Parse(req.RedirectURI)
	values := redirectURL.Query()
	values.Set("code", code)
	if state := req.State; state != "" {
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
	if !s.allowOIDCRate(w, r, &tenant.ID, "oidc_token:client", clientID, 60, time.Minute) {
		return
	}
	response, err := s.oidcProviderForRequest(r).Token(r.Context(), oidc.TokenRequest{TenantID: tenant.ID, TenantSlug: tenant.Slug, ClientID: clientID, ClientSecret: clientSecret, GrantType: r.Form.Get("grant_type"), Code: r.Form.Get("code"), RefreshToken: r.Form.Get("refresh_token"), RedirectURI: r.Form.Get("redirect_uri"), CodeVerifier: r.Form.Get("code_verifier")})
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return
	}
	recordOIDCTokenIssued(r.Form.Get("grant_type"))
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
	claims, err := s.oidcProviderForRequest(r).UserInfo(r.Context(), tenant.Slug, tenant.ID, token)
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
		UserID       string   `json:"user_id"`
		ClientID     string   `json:"client_id"`
		Scopes       []string `json:"scopes"`
		Decision     string   `json:"decision"`
		Continuation string   `json:"continue"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeOIDCError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if payload.Decision != "" && payload.Decision != "allow" {
		writeOIDCError(w, http.StatusForbidden, "access_denied")
		return
	}
	userID, client, scopes, redirectTo, ok := s.resolveConsentPayload(w, r, tenant, payload.UserID, payload.ClientID, payload.Scopes, payload.Continuation)
	if !ok {
		return
	}
	if err := s.oidcProvider().RecordConsent(r.Context(), tenant.ID, userID, client.ID, scopes); err != nil {
		writeOIDCError(w, http.StatusInternalServerError, "server_error")
		return
	}
	response := map[string]string{"status": "ok"}
	if redirectTo != "" {
		response["redirect_to"] = redirectTo
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) resolveConsentPayload(w http.ResponseWriter, r *http.Request, tenant Tenant, rawUserID, rawClientID string, scopes []string, token string) (uuid.UUID, oidc.Client, []string, string, bool) {
	if token != "" {
		continuation, err := s.parseOIDCAuthorizeContinuation(token)
		if err != nil || continuation.TenantID != tenant.ID.String() {
			writeOIDCError(w, http.StatusBadRequest, "invalid_request")
			return uuid.Nil, oidc.Client{}, nil, "", false
		}
		userID, ok := s.currentUserSession(r, tenant.ID)
		if !ok {
			writeOIDCError(w, http.StatusUnauthorized, "login_required")
			return uuid.Nil, oidc.Client{}, nil, "", false
		}
		client, err := s.oidcProvider().LoadClient(r.Context(), tenant.ID, continuation.ClientID)
		if err != nil {
			writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
			return uuid.Nil, oidc.Client{}, nil, "", false
		}
		return userID, client, continuation.Scope, "/oidc/authorize?continue=" + url.QueryEscape(token), true
	}
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, "invalid_request")
		return uuid.Nil, oidc.Client{}, nil, "", false
	}
	client, err := s.oidcProvider().LoadClient(r.Context(), tenant.ID, rawClientID)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, oidcErrorCode(err))
		return uuid.Nil, oidc.Client{}, nil, "", false
	}
	return userID, client, scopes, "", true
}

func (c oidcAuthorizeContinuation) authorizeRequest(tenantID uuid.UUID) oidc.AuthorizeRequest {
	return oidc.AuthorizeRequest{
		TenantID:            tenantID,
		ClientID:            c.ClientID,
		RedirectURI:         c.RedirectURI,
		Scope:               c.Scope,
		State:               c.State,
		Nonce:               c.Nonce,
		CodeChallenge:       c.CodeChallenge,
		CodeChallengeMethod: c.CodeChallengeMethod,
	}
}

func (s *Server) issueOIDCAuthorizeContinuation(tenantID uuid.UUID, req oidc.AuthorizeRequest) (string, error) {
	continuation := oidcAuthorizeContinuation{
		TenantID:            tenantID.String(),
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		State:               req.State,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().UTC().Add(oidcAuthorizeContinuationTTL).Unix(),
	}
	payload, err := json.Marshal(continuation)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.signOIDCAuthorizeContinuation(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *Server) parseOIDCAuthorizeContinuation(token string) (oidcAuthorizeContinuation, error) {
	encodedPayload, encodedSignature, ok := strings.Cut(token, ".")
	if !ok || encodedPayload == "" || encodedSignature == "" {
		return oidcAuthorizeContinuation{}, errOIDCContinuationInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return oidcAuthorizeContinuation{}, errOIDCContinuationInvalid
	}
	if !hmac.Equal(signature, s.signOIDCAuthorizeContinuation(encodedPayload)) {
		return oidcAuthorizeContinuation{}, errOIDCContinuationInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return oidcAuthorizeContinuation{}, errOIDCContinuationInvalid
	}
	var continuation oidcAuthorizeContinuation
	if err := json.Unmarshal(payload, &continuation); err != nil {
		return oidcAuthorizeContinuation{}, errOIDCContinuationInvalid
	}
	if time.Now().UTC().Unix() > continuation.ExpiresAt {
		return oidcAuthorizeContinuation{}, errOIDCContinuationInvalid
	}
	return continuation, nil
}

func (s *Server) signOIDCAuthorizeContinuation(encodedPayload string) []byte {
	mac := hmac.New(sha256.New, s.MasterKey)
	_, _ = mac.Write([]byte(encodedPayload))
	return mac.Sum(nil)
}

func (s *Server) oidcProvider() oidc.Provider {
	return oidc.Provider{DB: s.DB, KEK: s.MasterKey, InstallDomain: strings.Split(s.installHost, ":")[0], Storage: s.Storage}
}

func (s *Server) oidcProviderForRequest(r *http.Request) oidc.Provider {
	return oidc.Provider{DB: s.DB, KEK: s.MasterKey, InstallDomain: requestOrigin(r), Storage: s.Storage}
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
	switch {
	case errors.Is(err, oidc.ErrInvalidRequest):
		return "invalid_request"
	case errors.Is(err, oidc.ErrInvalidClient):
		return "invalid_client"
	case errors.Is(err, oidc.ErrInvalidGrant):
		return "invalid_grant"
	case errors.Is(err, oidc.ErrInvalidRedirectURI):
		return "invalid_redirect_uri"
	case errors.Is(err, oidc.ErrInvalidScope):
		return "invalid_scope"
	case errors.Is(err, oidc.ErrUnsupportedGrant):
		return "unsupported_grant_type"
	case errors.Is(err, oidc.ErrUnsupportedToken):
		return "unsupported_token_type"
	case errors.Is(err, oidc.ErrUnsupportedPKCEMode):
		return "invalid_pkce_method"
	default:
		return "server_error"
	}
}

func writeOIDCError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code, "error_description": oidcErrorDescription(code)})
}

func oidcErrorDescription(code string) string {
	switch code {
	case "invalid_request":
		return "The authorization request is invalid."
	case "invalid_client":
		return "Client authentication failed."
	case "invalid_grant":
		return "The authorization grant is invalid or expired."
	case "invalid_redirect_uri":
		return "The redirect URI is not registered for this client."
	case "invalid_scope":
		return "One or more requested scopes are not allowed."
	case "unsupported_grant_type":
		return "The requested grant type is not supported."
	case "unsupported_token_type":
		return "The token type is not supported."
	case "invalid_pkce_method":
		return "PKCE with S256 is required."
	case "invalid_token":
		return "The access token is invalid or expired."
	case "access_denied":
		return "The resource owner denied the request."
	case "login_required":
		return "User authentication is required."
	default:
		return "The request could not be completed."
	}
}
