package oidc

//revive:disable:exported

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	cypra "github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/sessions"
	"github.com/watzon/cypra/internal/storage"
)

var (
	ErrInvalidRequest      = errors.New("invalid_request")
	ErrInvalidClient       = errors.New("invalid_client")
	ErrInvalidGrant        = errors.New("invalid_grant")
	ErrInvalidRedirectURI  = errors.New("invalid_redirect_uri")
	ErrInvalidScope        = errors.New("invalid_scope")
	ErrUnsupportedGrant    = errors.New("unsupported_grant_type")
	ErrUnsupportedToken    = errors.New("unsupported_token_type")
	ErrUnsupportedPKCEMode = errors.New("invalid_pkce_method")
)

type Provider struct {
	DB            *sql.DB
	KEK           []byte
	InstallDomain string
	Storage       storage.Store
	Now           func() time.Time
}

type Client struct {
	ID                      uuid.UUID
	TenantID                uuid.UUID
	ClientID                string
	ClientSecretEncrypted   []byte
	RedirectURIs            []string
	AllowedScopes           []string
	TokenEndpointAuthMethod string
}

type AuthorizeRequest struct {
	TenantID            uuid.UUID
	UserID              uuid.UUID
	ClientID            string
	RedirectURI         string
	Scope               []string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type TokenRequest struct {
	TenantID        uuid.UUID
	TenantSlug      string
	ClientID        string
	ClientSecret    string
	GrantType       string
	Code            string
	RefreshToken    string
	RedirectURI     string
	CodeVerifier    string
	Authenticated   bool
	AuthenticatedID string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func (p Provider) Authorize(ctx context.Context, req AuthorizeRequest) (string, error) {
	client, err := p.ValidateAuthorizeRequest(ctx, req)
	if err != nil {
		return "", err
	}
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	_, err = p.DB.ExecContext(ctx, `INSERT INTO oidc_authorization_codes (code_hash, tenant_id, oidc_client_uuid, user_id, redirect_uri, scope, pkce_challenge, pkce_method, nonce, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, 'S256', $8, $9)`, hashOpaque(code), req.TenantID, client.ID, req.UserID, req.RedirectURI, pq.Array(req.Scope), req.CodeChallenge, nullString(req.Nonce), p.now().Add(time.Minute))
	if err != nil {
		return "", fmt.Errorf("insert authorization code: %w", err)
	}
	return code, nil
}

func (p Provider) ValidateAuthorizeRequest(ctx context.Context, req AuthorizeRequest) (Client, error) {
	client, err := p.LoadClient(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return Client{}, err
	}
	if !RedirectURIMatches(client.RedirectURIs, req.RedirectURI) {
		return Client{}, ErrInvalidRedirectURI
	}
	if req.CodeChallenge == "" || req.CodeChallengeMethod != "S256" {
		return Client{}, ErrUnsupportedPKCEMode
	}
	if !scopeAllowed(req.Scope, client.AllowedScopes) {
		return Client{}, ErrInvalidScope
	}
	return client, nil
}

func (p Provider) Token(ctx context.Context, req TokenRequest) (TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return p.exchangeAuthorizationCode(ctx, req)
	case "refresh_token":
		return p.exchangeRefreshToken(ctx, req)
	default:
		return TokenResponse{}, ErrUnsupportedGrant
	}
}

func (p Provider) exchangeAuthorizationCode(ctx context.Context, req TokenRequest) (TokenResponse, error) {
	client, err := p.LoadClient(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return TokenResponse{}, err
	}
	if err := p.validateClientSecret(client, req.ClientSecret); err != nil {
		return TokenResponse{}, err
	}
	tx, err := p.DB.BeginTx(ctx, nil)
	if err != nil {
		return TokenResponse{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var codeRow authorizationCode
	err = tx.QueryRowContext(ctx, `UPDATE oidc_authorization_codes SET consumed_at = $2 WHERE code_hash = $1 AND consumed_at IS NULL AND expires_at > $2 RETURNING tenant_id, oidc_client_uuid, user_id, redirect_uri, scope, pkce_challenge, nonce`, hashOpaque(req.Code), p.now()).Scan(&codeRow.tenantID, &codeRow.clientUUID, &codeRow.userID, &codeRow.redirectURI, pq.Array(&codeRow.scope), &codeRow.pkceChallenge, &codeRow.nonce)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenResponse{}, ErrInvalidGrant
	}
	if err != nil {
		return TokenResponse{}, err
	}
	if codeRow.clientUUID != client.ID {
		return TokenResponse{}, fmt.Errorf("%w: client mismatch", ErrInvalidGrant)
	}
	if codeRow.redirectURI != req.RedirectURI {
		return TokenResponse{}, fmt.Errorf("%w: redirect mismatch", ErrInvalidGrant)
	}
	if !validPKCE(req.CodeVerifier, codeRow.pkceChallenge) {
		return TokenResponse{}, fmt.Errorf("%w: pkce mismatch", ErrInvalidGrant)
	}
	if err := tx.Commit(); err != nil {
		return TokenResponse{}, err
	}
	return p.mintTokens(ctx, req.TenantSlug, client, codeRow.userID, codeRow.scope)
}

func (p Provider) exchangeRefreshToken(ctx context.Context, req TokenRequest) (TokenResponse, error) {
	client, err := p.LoadClient(ctx, req.TenantID, req.ClientID)
	if err != nil {
		return TokenResponse{}, err
	}
	if err := p.validateClientSecret(client, req.ClientSecret); err != nil {
		return TokenResponse{}, err
	}
	refresh, err := sessions.NewRefreshService(p.DB).Consume(ctx, req.RefreshToken, p.now().Add(30*24*time.Hour))
	if err != nil || refresh.TenantID != req.TenantID || refresh.ClientID != client.ID {
		return TokenResponse{}, ErrInvalidGrant
	}
	response, err := p.mintAccessPair(ctx, req.TenantSlug, client, refresh.UserID, refresh.Scope)
	if err != nil {
		return TokenResponse{}, err
	}
	response.RefreshToken = refresh.Plaintext
	return response, nil
}

func (p Provider) mintTokens(ctx context.Context, tenantSlug string, client Client, userID uuid.UUID, scope []string) (TokenResponse, error) {
	response, err := p.mintAccessPair(ctx, tenantSlug, client, userID, scope)
	if err != nil {
		return TokenResponse{}, err
	}
	refresh, err := sessions.NewRefreshService(p.DB).Mint(ctx, client.TenantID, client.ID, userID, scope, nil, p.now().Add(30*24*time.Hour))
	if err != nil {
		return TokenResponse{}, err
	}
	response.RefreshToken = refresh.Plaintext
	return response, nil
}

func (p Provider) mintAccessPair(ctx context.Context, tenantSlug string, client Client, userID uuid.UUID, scope []string) (TokenResponse, error) {
	key, err := p.activeSigningKey(ctx, client.TenantID)
	if err != nil {
		return TokenResponse{}, err
	}
	access, _, err := sessions.MintAccessToken(tenantSlug, p.InstallDomain, client.TenantID, userID, client.ClientID, scope, key, p.KEK, p.now())
	if err != nil {
		return TokenResponse{}, err
	}
	idToken, _, err := sessions.MintAccessToken(tenantSlug, p.InstallDomain, client.TenantID, userID, client.ClientID, []string{"openid"}, key, p.KEK, p.now())
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: access, IDToken: idToken, TokenType: "Bearer", ExpiresIn: int(sessions.AccessTokenTTL.Seconds())}, nil
}

func (p Provider) UserInfo(ctx context.Context, tenantSlug string, tenantID uuid.UUID, token string) (map[string]any, error) {
	audience, err := claimsAudience(token)
	if err != nil || audience == "" {
		return nil, sessions.ErrInvalidToken
	}
	if _, err := p.LoadClient(ctx, tenantID, audience); err != nil {
		return nil, sessions.ErrInvalidToken
	}
	keys, err := p.SigningKeys(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	claims, err := sessions.VerifyAccessToken(token, keys, sessions.IssuerURL(tenantSlug, p.InstallDomain), audience, p.now())
	if err != nil {
		return nil, err
	}
	tenantText, userText, ok := strings.Cut(claims.Subject, ":")
	if !ok {
		return nil, sessions.ErrInvalidToken
	}
	subjectTenantID, err := uuid.Parse(tenantText)
	if err != nil || subjectTenantID != tenantID {
		return nil, sessions.ErrInvalidToken
	}
	userID, err := uuid.Parse(userText)
	if err != nil {
		return nil, err
	}
	var email string
	var verified sql.NullTime
	var name string
	var pictureKey sql.NullString
	if err := p.DB.QueryRowContext(ctx, `SELECT u.email, u.email_verified_at, COALESCE(u.metadata->>'name', ''), o.key FROM users u LEFT JOIN storage_objects o ON o.tenant_id = u.tenant_id AND o.id = u.profile_picture_object_id AND o.deleted_at IS NULL WHERE u.tenant_id = $1 AND u.id = $2 AND u.deleted_at IS NULL`, tenantID, userID).Scan(&email, &verified, &name, &pictureKey); err != nil {
		return nil, err
	}
	response := map[string]any{"sub": claims.Subject, "email": email, "email_verified": verified.Valid, "updated_at": p.now().Unix()}
	if name != "" {
		response["name"] = name
	}
	if pictureKey.Valid && p.Storage != nil {
		signed, err := p.Storage.SignedURL(pictureKey.String, 5*time.Minute)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(signed, "/") {
			signed = "https://" + tenantSlug + "." + p.InstallDomain + signed
		}
		response["picture"] = signed
	}
	return response, nil
}

func (p Provider) Revoke(ctx context.Context, token string) error {
	if strings.Count(token, ".") == 2 {
		return nil
	}
	result, err := p.DB.ExecContext(ctx, `UPDATE oidc_refresh_tokens SET revoked_at = now(), revoke_reason = 'revoked' WHERE family_id = (SELECT family_id FROM oidc_refresh_tokens WHERE token_hash = $1) AND revoked_at IS NULL`, hashOpaque(token))
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUnsupportedToken
	}
	return nil
}

func (p Provider) RecordConsent(ctx context.Context, tenantID, userID, clientUUID uuid.UUID, scopes []string) error {
	_, err := p.DB.ExecContext(ctx, `INSERT INTO oidc_consents (tenant_id, user_id, oidc_client_uuid, scopes) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, oidc_client_uuid) DO UPDATE SET scopes = EXCLUDED.scopes, granted_at = now(), revoked_at = NULL`, tenantID, userID, clientUUID, pq.Array(scopes))
	return err
}

func (p Provider) ConsentCovers(ctx context.Context, tenantID, userID, clientUUID uuid.UUID, requested []string) (bool, bool, error) {
	var granted []string
	err := p.DB.QueryRowContext(ctx, `SELECT scopes FROM oidc_consents WHERE tenant_id = $1 AND user_id = $2 AND oidc_client_uuid = $3 AND revoked_at IS NULL`, tenantID, userID, clientUUID).Scan(pq.Array(&granted))
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return scopeAllowed(requested, granted), true, nil
}

func (p Provider) LoadClient(ctx context.Context, tenantID uuid.UUID, clientID string) (Client, error) {
	var client Client
	err := p.DB.QueryRowContext(ctx, `SELECT id, tenant_id, client_id, client_secret_encrypted, redirect_uris, allowed_scopes, token_endpoint_auth_method FROM oidc_clients WHERE tenant_id = $1 AND client_id = $2 AND deleted_at IS NULL`, tenantID, clientID).Scan(&client.ID, &client.TenantID, &client.ClientID, &client.ClientSecretEncrypted, pq.Array(&client.RedirectURIs), pq.Array(&client.AllowedScopes), &client.TokenEndpointAuthMethod)
	if errors.Is(err, sql.ErrNoRows) {
		return Client{}, ErrInvalidClient
	}
	return client, err
}

func (p Provider) SigningKeys(ctx context.Context, tenantID uuid.UUID) ([]sessions.SigningKey, error) {
	rows, err := p.DB.QueryContext(ctx, `SELECT kid, algorithm, public_key_jwk, private_key_encrypted FROM oidc_signing_keys WHERE tenant_id = $1 AND (state IN ('active', 'overlap') OR (state = 'sunsetting' AND sunset_until > $2)) ORDER BY activated_at DESC`, tenantID, p.now())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var keys []sessions.SigningKey
	for rows.Next() {
		var key sessions.SigningKey
		if err := rows.Scan(&key.KID, &key.Algorithm, &key.PublicJWK, &key.PrivateKeyEncrypted); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (p Provider) activeSigningKey(ctx context.Context, tenantID uuid.UUID) (sessions.SigningKey, error) {
	keys, err := p.SigningKeys(ctx, tenantID)
	if err != nil || len(keys) == 0 {
		return sessions.SigningKey{}, err
	}
	return keys[0], nil
}

func (p Provider) validateClientSecret(client Client, provided string) error {
	if client.TokenEndpointAuthMethod == "none" {
		return nil
	}
	if provided == "" || len(client.ClientSecretEncrypted) == 0 || len(client.ClientSecretEncrypted) < 60 {
		return ErrInvalidClient
	}
	secret, err := cypra.Decrypt(client.ClientSecretEncrypted[60:], client.ClientSecretEncrypted[:60], p.KEK)
	if err != nil {
		return ErrInvalidClient
	}
	storedDigest := sha256.Sum256(secret)
	providedDigest := sha256.Sum256([]byte(provided))
	if !hmac.Equal(storedDigest[:], providedDigest[:]) {
		return ErrInvalidClient
	}
	return nil
}

func RedirectURIMatches(registered []string, requested string) bool {
	req, err := url.Parse(requested)
	if err != nil || req.Fragment != "" || req.Scheme == "" || req.Host == "" {
		return false
	}
	for _, raw := range registered {
		candidate, err := url.Parse(raw)
		if err != nil || candidate.Fragment != "" {
			continue
		}
		if candidate.Scheme == req.Scheme && strings.EqualFold(candidate.Hostname(), req.Hostname()) && candidate.Port() == req.Port() && candidate.EscapedPath() == req.EscapedPath() && candidate.RawQuery == req.RawQuery {
			return true
		}
	}
	return false
}

func EncryptClientSecret(secret string, kek []byte) ([]byte, error) {
	ciphertext, encryptedDEK, err := cypra.Encrypt([]byte(secret), kek)
	if err != nil {
		return nil, err
	}
	return append(encryptedDEK, ciphertext...), nil
}

func scopeAllowed(requested, allowed []string) bool {
	for _, scope := range requested {
		if !slices.Contains(allowed, scope) {
			return false
		}
	}
	return true
}

func validPKCE(verifier, challenge string) bool {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:]) == challenge
}

func hashOpaque(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

func randomCode() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func nullString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func (p Provider) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now().UTC()
}

func claimsAudience(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", sessions.ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", sessions.ErrInvalidToken
	}
	var claims struct {
		Audience string `json:"aud"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", sessions.ErrInvalidToken
	}
	return claims.Audience, nil
}

type authorizationCode struct {
	tenantID      uuid.UUID
	clientUUID    uuid.UUID
	userID        uuid.UUID
	redirectURI   string
	scope         []string
	pkceChallenge string
	nonce         sql.NullString
}
