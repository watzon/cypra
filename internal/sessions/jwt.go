// Package sessions manages Cypra access and refresh sessions.
package sessions

//revive:disable:exported

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	cypra "github.com/watzon/cypra/internal/crypto"
)

const (
	AccessTokenTTL = 15 * time.Minute
	ClockLeeway    = 60 * time.Second
)

var ErrInvalidToken = errors.New("invalid token")

type AccessTokenClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	IssuedAt  int64    `json:"iat"`
	Expires   int64    `json:"exp"`
	NotBefore int64    `json:"nbf"`
	Audience  string   `json:"aud"`
	Scope     []string `json:"scope"`
}

type SigningKey struct {
	KID                 string
	Algorithm           string
	PublicJWK           json.RawMessage
	PrivateKeyEncrypted []byte
}

func MintAccessToken(tenantSlug, installDomain string, tenantID, userID uuid.UUID, audience string, scope []string, key SigningKey, kek []byte, now time.Time) (string, AccessTokenClaims, error) {
	issuer := IssuerURL(tenantSlug, installDomain)
	claims := AccessTokenClaims{
		Issuer:    issuer,
		Subject:   tenantID.String() + ":" + userID.String(),
		IssuedAt:  now.Unix(),
		Expires:   now.Add(AccessTokenTTL).Unix(),
		NotBefore: now.Unix(),
		Audience:  audience,
		Scope:     scope,
	}
	token, err := signJWT(jwtHeader{Algorithm: key.Algorithm, Type: "JWT", KID: key.KID}, claims, key.PrivateKeyEncrypted, kek)
	return token, claims, err
}

// IssuerURL builds the tenant issuer URL. installDomain is normally the bare
// install host, but local e2e/proxy paths may pass a full request origin.
func IssuerURL(tenantSlug, installDomain string) string {
	if strings.HasPrefix(installDomain, "http://") || strings.HasPrefix(installDomain, "https://") {
		return strings.TrimRight(installDomain, "/")
	}
	return fmt.Sprintf("https://%s.%s", tenantSlug, installDomain)
}

func VerifyAccessToken(token string, keys []SigningKey, expectedIssuer, expectedAudience string, now time.Time) (AccessTokenClaims, error) {
	header, claims, signingInput, signature, err := parseJWT(token)
	if err != nil {
		return AccessTokenClaims{}, err
	}
	var selected *SigningKey
	for i := range keys {
		if keys[i].KID == header.KID && keys[i].Algorithm == header.Algorithm {
			selected = &keys[i]
			break
		}
	}
	if selected == nil {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	if err := verifySignature(header, signingInput, signature, selected.PublicJWK); err != nil {
		return AccessTokenClaims{}, err
	}
	if claims.Issuer != expectedIssuer || (expectedAudience != "" && claims.Audience != expectedAudience) {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	if now.Add(ClockLeeway).Unix() < claims.NotBefore || now.Add(-ClockLeeway).Unix() > claims.Expires {
		return AccessTokenClaims{}, ErrInvalidToken
	}
	return claims, nil
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KID       string `json:"kid"`
}

func signJWT(header jwtHeader, claims AccessTokenClaims, privateEnvelope, kek []byte) (string, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	privateDER, err := cypra.DecryptSigningPrivateKey(privateEnvelope, kek)
	if err != nil {
		return "", err
	}
	parsed, err := x509.ParsePKCS8PrivateKey(privateDER)
	if err != nil {
		return "", fmt.Errorf("parse signing private key: %w", err)
	}
	digest := sha256.Sum256([]byte(signingInput))
	var signature []byte
	switch header.Algorithm {
	case cypra.SigningAlgRS256:
		privateKey, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			return "", ErrInvalidToken
		}
		signature, err = rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	case cypra.SigningAlgES256:
		privateKey, ok := parsed.(*ecdsa.PrivateKey)
		if !ok {
			return "", ErrInvalidToken
		}
		r, s, signErr := ecdsa.Sign(rand.Reader, privateKey, digest[:])
		if signErr != nil {
			err = signErr
			break
		}
		signature = append(padP256(r), padP256(s)...)
	default:
		return "", cypra.ErrUnsupportedSigningAlgorithm
	}
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func parseJWT(token string) (jwtHeader, AccessTokenClaims, string, []byte, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtHeader{}, AccessTokenClaims{}, "", nil, ErrInvalidToken
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return jwtHeader{}, AccessTokenClaims{}, "", nil, ErrInvalidToken
	}
	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return jwtHeader{}, AccessTokenClaims{}, "", nil, ErrInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return jwtHeader{}, AccessTokenClaims{}, "", nil, ErrInvalidToken
	}
	var header jwtHeader
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		return jwtHeader{}, AccessTokenClaims{}, "", nil, ErrInvalidToken
	}
	var claims AccessTokenClaims
	if err := json.Unmarshal(claimsRaw, &claims); err != nil {
		return jwtHeader{}, AccessTokenClaims{}, "", nil, ErrInvalidToken
	}
	return header, claims, parts[0] + "." + parts[1], signature, nil
}

func verifySignature(header jwtHeader, signingInput string, signature []byte, publicJWK json.RawMessage) error {
	digest := sha256.Sum256([]byte(signingInput))
	var jwk map[string]string
	if err := json.Unmarshal(publicJWK, &jwk); err != nil {
		return ErrInvalidToken
	}
	switch header.Algorithm {
	case cypra.SigningAlgRS256:
		n, err := decodeBigInt(jwk["n"])
		if err != nil {
			return ErrInvalidToken
		}
		e, err := decodeBigInt(jwk["e"])
		if err != nil {
			return ErrInvalidToken
		}
		publicKey := rsa.PublicKey{N: n, E: int(e.Int64())}
		if err := rsa.VerifyPKCS1v15(&publicKey, crypto.SHA256, digest[:], signature); err != nil {
			return ErrInvalidToken
		}
		return nil
	case cypra.SigningAlgES256:
		if len(signature) != 64 {
			return ErrInvalidToken
		}
		x, err := decodeBigInt(jwk["x"])
		if err != nil {
			return ErrInvalidToken
		}
		y, err := decodeBigInt(jwk["y"])
		if err != nil {
			return ErrInvalidToken
		}
		publicKey := ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
		if !ecdsa.Verify(&publicKey, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
			return ErrInvalidToken
		}
		return nil
	default:
		return cypra.ErrUnsupportedSigningAlgorithm
	}
}

func decodeBigInt(raw string) (*big.Int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(decoded), nil
}

func padP256(value *big.Int) []byte {
	raw := value.Bytes()
	if len(raw) >= 32 {
		return raw
	}
	return append(make([]byte, 32-len(raw)), raw...)
}
