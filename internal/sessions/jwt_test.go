package sessions_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	cypra "github.com/watzon/cypra/internal/crypto"
	"github.com/watzon/cypra/internal/sessions"
)

func TestMintAndVerifyAccessToken(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	generated, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgRS256, kek, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	key := sessions.SigningKey{KID: generated.KID, Algorithm: generated.Algorithm, PublicJWK: generated.PublicJWK, PrivateKeyEncrypted: generated.PrivateKeyEncrypted}
	now := time.Unix(100, 0).UTC()
	token, claims, err := sessions.MintAccessToken("acme", "cypra.localhost", tenantID, userID, "client", []string{"openid", "email"}, key, kek, now)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	if claims.Issuer != "https://acme.cypra.localhost" {
		t.Fatalf("issuer = %q", claims.Issuer)
	}
	if claims.Subject != tenantID.String()+":"+userID.String() {
		t.Fatalf("subject = %q", claims.Subject)
	}
	verified, err := sessions.VerifyAccessToken(token, []sessions.SigningKey{key}, "https://acme.cypra.localhost", "client", now.Add(30*time.Second))
	if err != nil {
		t.Fatalf("verify access token: %v", err)
	}
	if verified.Subject != claims.Subject || verified.Expires != now.Add(sessions.AccessTokenTTL).Unix() {
		t.Fatalf("verified claims mismatch: %#v", verified)
	}
}

func TestMintAndVerifyAccessTokenES256(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	kek := bytes.Repeat([]byte{2}, cypra.MasterKeyBytes)
	generated, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgES256, kek, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	key := sessions.SigningKey{KID: generated.KID, Algorithm: generated.Algorithm, PublicJWK: generated.PublicJWK, PrivateKeyEncrypted: generated.PrivateKeyEncrypted}
	now := time.Unix(100, 0).UTC()
	token, claims, err := sessions.MintAccessToken("acme", "cypra.localhost", tenantID, userID, "client", []string{"openid"}, key, kek, now)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	verified, err := sessions.VerifyAccessToken(token, []sessions.SigningKey{key}, claims.Issuer, "client", now)
	if err != nil {
		t.Fatalf("verify access token: %v", err)
	}
	if verified.Subject != claims.Subject {
		t.Fatalf("subject = %q, want %q", verified.Subject, claims.Subject)
	}
}

func TestVerifyAccessTokenRejectsMalformedTokens(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	kek := bytes.Repeat([]byte{3}, cypra.MasterKeyBytes)
	generated, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgRS256, kek, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	key := sessions.SigningKey{KID: generated.KID, Algorithm: generated.Algorithm, PublicJWK: generated.PublicJWK, PrivateKeyEncrypted: generated.PrivateKeyEncrypted}
	now := time.Unix(100, 0).UTC()
	token, _, err := sessions.MintAccessToken("acme", "cypra.localhost", tenantID, userID, "client", []string{"openid"}, key, kek, now)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	parts := strings.Split(token, ".")

	badHeader, _ := json.Marshal(map[string]string{"alg": cypra.SigningAlgRS256, "typ": "JWT", "kid": "missing"})
	unknownKID := base64.RawURLEncoding.EncodeToString(badHeader) + "." + parts[1] + "." + parts[2]
	tampered := parts[0] + "." + parts[1] + "." + base64.RawURLEncoding.EncodeToString([]byte("bad-signature"))

	for _, raw := range []string{
		"one.two",
		"%%%" + "." + parts[1] + "." + parts[2],
		parts[0] + ".%%% ." + parts[2],
		parts[0] + "." + parts[1] + ".%%%",
		unknownKID,
		tampered,
	} {
		if _, err := sessions.VerifyAccessToken(raw, []sessions.SigningKey{key}, "https://acme.cypra.localhost", "client", now); !errors.Is(err, sessions.ErrInvalidToken) {
			t.Fatalf("%q error = %v, want %v", raw, err, sessions.ErrInvalidToken)
		}
	}
}

func TestVerifyAccessTokenRejectsBadJWKAndUnsupportedAlgorithm(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	claims := sessions.AccessTokenClaims{
		Issuer:    "https://acme.cypra.localhost",
		Subject:   uuid.NewString(),
		IssuedAt:  now.Unix(),
		Expires:   now.Add(time.Hour).Unix(),
		NotBefore: now.Unix(),
		Audience:  "client",
		Scope:     []string{"openid"},
	}

	badJWKToken := unsignedToken(t, map[string]string{"alg": cypra.SigningAlgRS256, "typ": "JWT", "kid": "bad"}, claims)
	_, err := sessions.VerifyAccessToken(badJWKToken, []sessions.SigningKey{{KID: "bad", Algorithm: cypra.SigningAlgRS256, PublicJWK: []byte(`{}`)}}, claims.Issuer, "client", now)
	if !errors.Is(err, sessions.ErrInvalidToken) {
		t.Fatalf("bad jwk error = %v, want invalid token", err)
	}

	unsupportedToken := unsignedToken(t, map[string]string{"alg": "HS256", "typ": "JWT", "kid": "bad"}, claims)
	_, err = sessions.VerifyAccessToken(unsupportedToken, []sessions.SigningKey{{KID: "bad", Algorithm: "HS256", PublicJWK: []byte(`{}`)}}, claims.Issuer, "client", now)
	if !errors.Is(err, cypra.ErrUnsupportedSigningAlgorithm) {
		t.Fatalf("unsupported algorithm error = %v", err)
	}
}

func TestMintAccessTokenRejectsMalformedPrivateEnvelope(t *testing.T) {
	_, _, err := sessions.MintAccessToken(
		"acme",
		"cypra.localhost",
		uuid.New(),
		uuid.New(),
		"client",
		[]string{"openid"},
		sessions.SigningKey{KID: "bad", Algorithm: cypra.SigningAlgRS256, PrivateKeyEncrypted: []byte("not-json")},
		bytes.Repeat([]byte{1}, cypra.MasterKeyBytes),
		time.Unix(100, 0).UTC(),
	)
	if err == nil {
		t.Fatal("malformed private envelope unexpectedly minted")
	}
}

func unsignedToken(t *testing.T, header map[string]string, claims sessions.AccessTokenClaims) string {
	t.Helper()
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON) + "." + base64.RawURLEncoding.EncodeToString([]byte("signature"))
}

func TestVerifyAccessTokenRejectsWrongIssuer(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	generated, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgRS256, kek, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	key := sessions.SigningKey{KID: generated.KID, Algorithm: generated.Algorithm, PublicJWK: generated.PublicJWK, PrivateKeyEncrypted: generated.PrivateKeyEncrypted}
	token, _, err := sessions.MintAccessToken("acme", "cypra.localhost", tenantID, userID, "client", []string{"openid"}, key, kek, time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	if _, err := sessions.VerifyAccessToken(token, []sessions.SigningKey{key}, "https://bravo.cypra.localhost", "client", time.Unix(100, 0).UTC()); err == nil {
		t.Fatal("wrong issuer unexpectedly verified")
	}
}

func TestVerifyAccessTokenHonorsClockSkew(t *testing.T) {
	tenantID := uuid.New()
	userID := uuid.New()
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	generated, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgRS256, kek, time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	key := sessions.SigningKey{KID: generated.KID, Algorithm: generated.Algorithm, PublicJWK: generated.PublicJWK, PrivateKeyEncrypted: generated.PrivateKeyEncrypted}
	now := time.Unix(100, 0).UTC()
	token, _, err := sessions.MintAccessToken("acme", "cypra.localhost", tenantID, userID, "client", []string{"openid"}, key, kek, now)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	if _, err := sessions.VerifyAccessToken(token, []sessions.SigningKey{key}, "https://acme.cypra.localhost", "client", now.Add(sessions.AccessTokenTTL+sessions.ClockLeeway)); err != nil {
		t.Fatalf("token within leeway rejected: %v", err)
	}
	if _, err := sessions.VerifyAccessToken(token, []sessions.SigningKey{key}, "https://acme.cypra.localhost", "client", now.Add(sessions.AccessTokenTTL+sessions.ClockLeeway+time.Second)); err == nil {
		t.Fatal("token beyond leeway unexpectedly verified")
	}
}
