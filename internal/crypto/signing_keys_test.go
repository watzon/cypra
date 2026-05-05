package crypto_test

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	cypra "github.com/watzon/cypra/internal/crypto"
)

func TestGenerateSigningKeyRS256(t *testing.T) {
	tenantID := uuid.New()
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	key, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgRS256, kek, time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	if key.KID != tenantID.String()+":1" {
		t.Fatalf("kid = %q", key.KID)
	}
	var jwk map[string]string
	if err := json.Unmarshal(key.PublicJWK, &jwk); err != nil {
		t.Fatalf("unmarshal jwk: %v", err)
	}
	if jwk["kty"] != "RSA" || jwk["alg"] != cypra.SigningAlgRS256 || jwk["kid"] != key.KID {
		t.Fatalf("unexpected jwk: %#v", jwk)
	}
	privateDER, err := cypra.DecryptSigningPrivateKey(key.PrivateKeyEncrypted, kek)
	if err != nil {
		t.Fatalf("decrypt private key: %v", err)
	}
	if _, err := x509.ParsePKCS8PrivateKey(privateDER); err != nil {
		t.Fatalf("parse private key: %v", err)
	}
}

func TestGenerateSigningKeyES256(t *testing.T) {
	tenantID := uuid.New()
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	key, err := cypra.GenerateSigningKey(tenantID, 2, cypra.SigningAlgES256, kek, time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	var jwk map[string]string
	if err := json.Unmarshal(key.PublicJWK, &jwk); err != nil {
		t.Fatalf("unmarshal jwk: %v", err)
	}
	if jwk["kty"] != "EC" || jwk["alg"] != cypra.SigningAlgES256 || jwk["kid"] != key.KID {
		t.Fatalf("unexpected jwk: %#v", jwk)
	}
}
