package upstream

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrStateMismatch = errors.New("auth.upstream_state_mismatch")
	ErrNonceMismatch = errors.New("auth.upstream_nonce_mismatch")
)

type State struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	Provider  string    `json:"provider,omitempty"`
	Slug      string    `json:"slug,omitempty"`
	ReturnURL string    `json:"return_url"`
	Nonce     string    `json:"nonce"`
	ExpiresAt int64     `json:"exp"`
}

func SignState(secret []byte, state State) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + signSegment(secret, encoded), nil
}

func ValidateState(secret []byte, raw string, now time.Time) (State, error) {
	encoded, sig, ok := strings.Cut(raw, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(signSegment(secret, encoded))) {
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

func RandomNonce() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate upstream nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func signSegment(secret []byte, encodedPayload string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(encodedPayload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
