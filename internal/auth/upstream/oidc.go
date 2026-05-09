package upstream

import (
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"strings"
)

type IDTokenClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Nonce         string `json:"nonce"`
	Picture       string `json:"picture"`
	Audience      any    `json:"aud"`
	Issuer        string `json:"iss"`
}

func ParseIDTokenClaims(idToken string) (IDTokenClaims, map[string]any, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) < 2 {
		return IDTokenClaims{}, nil, ErrNonceMismatch
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return IDTokenClaims{}, nil, ErrNonceMismatch
	}
	var typed IDTokenClaims
	if err := json.Unmarshal(payload, &typed); err != nil {
		return IDTokenClaims{}, nil, ErrNonceMismatch
	}
	var raw map[string]any
	_ = json.Unmarshal(payload, &raw)
	return typed, raw, nil
}

func ValidateIDTokenNonce(idToken, want string) error {
	claims, _, err := ParseIDTokenClaims(idToken)
	if err != nil || claims.Nonce == "" || !hmac.Equal([]byte(claims.Nonce), []byte(want)) {
		return ErrNonceMismatch
	}
	return nil
}
