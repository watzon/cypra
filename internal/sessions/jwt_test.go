package sessions_test

import (
	"bytes"
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
