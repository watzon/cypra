// Package google implements the Google upstream OAuth state and nonce checks.
package google

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

//revive:disable:exported

var (
	ErrStateMismatch = errors.New("auth.upstream_state_mismatch")
	ErrNonceMismatch = errors.New("auth.upstream_nonce_mismatch")
)

type Client struct {
	ClientID    string
	RedirectURI string
	Secret      []byte
	AuthURL     string
}

type State struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	ReturnURL string    `json:"return_url"`
	Nonce     string    `json:"nonce"`
	ExpiresAt int64     `json:"exp"`
}

type IDTokenClaims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Nonce   string `json:"nonce"`
}

func (c Client) AuthCodeURL(tenantID uuid.UUID, returnURL string, ttl time.Duration) (string, State, error) {
	nonce, err := randomNonce()
	if err != nil {
		return "", State{}, err
	}
	state := State{TenantID: tenantID, ReturnURL: returnURL, Nonce: nonce, ExpiresAt: time.Now().UTC().Add(ttl).Unix()}
	rawState, err := c.SignState(state)
	if err != nil {
		return "", State{}, err
	}
	authURL := c.AuthURL
	if authURL == "" {
		authURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	values := url.Values{}
	values.Set("client_id", c.ClientID)
	values.Set("redirect_uri", c.RedirectURI)
	values.Set("response_type", "code")
	values.Set("scope", "openid email profile")
	values.Set("state", rawState)
	values.Set("nonce", nonce)
	return authURL + "?" + values.Encode(), state, nil
}

func (c Client) SignState(state State) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + c.sign(encoded), nil
}

func (c Client) ValidateState(raw string, now time.Time) (State, error) {
	encoded, sig, ok := strings.Cut(raw, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(c.sign(encoded))) {
		return State{}, ErrStateMismatch
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return State{}, ErrStateMismatch
	}
	var state State
	if err := json.Unmarshal(payload, &state); err != nil || now.Unix() > state.ExpiresAt || state.Nonce == "" {
		return State{}, ErrStateMismatch
	}
	return state, nil
}

func ValidateIDTokenNonce(idToken, want string) error {
	claims, err := ParseIDTokenClaims(idToken)
	if err != nil || claims.Nonce == "" || !hmac.Equal([]byte(claims.Nonce), []byte(want)) {
		return ErrNonceMismatch
	}
	return nil
}

func ParseIDTokenClaims(idToken string) (IDTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return IDTokenClaims{}, ErrNonceMismatch
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return IDTokenClaims{}, ErrNonceMismatch
	}
	var claims IDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return IDTokenClaims{}, ErrNonceMismatch
	}
	return claims, nil
}

func (c Client) sign(encodedPayload string) string {
	mac := hmac.New(sha256.New, c.Secret)
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func randomNonce() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate google oauth nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
