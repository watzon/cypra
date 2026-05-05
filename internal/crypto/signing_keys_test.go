package crypto_test

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
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

func TestGenerateSigningKeyDefaultsAndRejectsInvalidInputs(t *testing.T) {
	tenantID := uuid.New()
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	key, err := cypra.GenerateSigningKey(tenantID, 3, "", kek, time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("generate default signing key: %v", err)
	}
	if key.Algorithm != cypra.SigningAlgRS256 {
		t.Fatalf("algorithm = %q", key.Algorithm)
	}

	if _, err := cypra.GenerateSigningKey(uuid.Nil, 1, cypra.SigningAlgRS256, kek, time.Unix(100, 0).UTC()); err == nil {
		t.Fatal("nil tenant signing key unexpectedly generated")
	}
	if _, err := cypra.GenerateSigningKey(tenantID, 0, cypra.SigningAlgRS256, kek, time.Unix(100, 0).UTC()); err == nil {
		t.Fatal("zero sequence signing key unexpectedly generated")
	}
	if _, err := cypra.GenerateSigningKey(tenantID, 1, "HS256", kek, time.Unix(100, 0).UTC()); !errors.Is(err, cypra.ErrUnsupportedSigningAlgorithm) {
		t.Fatalf("unsupported algorithm error = %v", err)
	}
}

func TestDecryptSigningPrivateKeyRejectsMalformedEnvelope(t *testing.T) {
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	badEnvelopes := [][]byte{
		[]byte("not-json"),
		[]byte(`{"ciphertext":"%%%","encrypted_dek":"abc"}`),
		[]byte(`{"ciphertext":"abc","encrypted_dek":"%%%"}`),
	}

	for _, envelope := range badEnvelopes {
		if _, err := cypra.DecryptSigningPrivateKey(envelope, kek); err == nil {
			t.Fatalf("envelope %s unexpectedly decrypted", envelope)
		}
	}
}
