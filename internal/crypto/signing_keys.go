package crypto

//revive:disable:exported

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

const (
	SigningAlgRS256 = "RS256"
	SigningAlgES256 = "ES256"
)

var ErrUnsupportedSigningAlgorithm = errors.New("unsupported signing algorithm")

type SigningKey struct {
	TenantID            uuid.UUID
	KID                 string
	Algorithm           string
	PublicJWK           json.RawMessage
	PrivateKeyEncrypted []byte
	ActivatedAt         time.Time
	RetiresAt           time.Time
}

func GenerateSigningKey(tenantID uuid.UUID, seq int, algorithm string, kek []byte, activatedAt time.Time) (SigningKey, error) {
	if tenantID == uuid.Nil || seq <= 0 {
		return SigningKey{}, fmt.Errorf("invalid signing key identity")
	}
	if algorithm == "" {
		algorithm = SigningAlgRS256
	}
	kid := fmt.Sprintf("%s:%d", tenantID.String(), seq)
	privateDER, publicJWK, err := generateKeyMaterial(kid, algorithm)
	if err != nil {
		return SigningKey{}, err
	}
	ciphertext, encryptedDEK, err := Encrypt(privateDER, kek)
	if err != nil {
		return SigningKey{}, err
	}
	privateEnvelope, err := marshalEnvelope(ciphertext, encryptedDEK)
	if err != nil {
		return SigningKey{}, err
	}
	return SigningKey{
		TenantID:            tenantID,
		KID:                 kid,
		Algorithm:           algorithm,
		PublicJWK:           publicJWK,
		PrivateKeyEncrypted: privateEnvelope,
		ActivatedAt:         activatedAt,
		RetiresAt:           activatedAt.Add(90 * 24 * time.Hour),
	}, nil
}

func DecryptSigningPrivateKey(privateEnvelope, kek []byte) ([]byte, error) {
	ciphertext, encryptedDEK, err := unmarshalEnvelope(privateEnvelope)
	if err != nil {
		return nil, err
	}
	return Decrypt(ciphertext, encryptedDEK, kek)
}

func generateKeyMaterial(kid, algorithm string) ([]byte, json.RawMessage, error) {
	switch algorithm {
	case SigningAlgRS256:
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, fmt.Errorf("generate rsa signing key: %w", err)
		}
		privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal rsa private key: %w", err)
		}
		publicJWK, err := rsaPublicJWK(kid, privateKey.PublicKey)
		if err != nil {
			return nil, nil, err
		}
		return privateDER, publicJWK, nil
	case SigningAlgES256:
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, fmt.Errorf("generate ecdsa signing key: %w", err)
		}
		privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal ecdsa private key: %w", err)
		}
		publicJWK, err := ecdsaPublicJWK(kid, privateKey.PublicKey)
		if err != nil {
			return nil, nil, err
		}
		return privateDER, publicJWK, nil
	default:
		return nil, nil, ErrUnsupportedSigningAlgorithm
	}
}

func rsaPublicJWK(kid string, key rsa.PublicKey) (json.RawMessage, error) {
	return json.Marshal(map[string]string{
		"kty": "RSA",
		"use": "sig",
		"alg": SigningAlgRS256,
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	})
}

func ecdsaPublicJWK(kid string, key ecdsa.PublicKey) (json.RawMessage, error) {
	return json.Marshal(map[string]string{
		"kty": "EC",
		"use": "sig",
		"alg": SigningAlgES256,
		"kid": kid,
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(key.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(key.Y.Bytes()),
	})
}

type encryptedEnvelope struct {
	Ciphertext   string `json:"ciphertext"`
	EncryptedDEK string `json:"encrypted_dek"`
}

func marshalEnvelope(ciphertext, encryptedDEK []byte) ([]byte, error) {
	return json.Marshal(encryptedEnvelope{
		Ciphertext:   base64.RawStdEncoding.EncodeToString(ciphertext),
		EncryptedDEK: base64.RawStdEncoding.EncodeToString(encryptedDEK),
	})
}

func unmarshalEnvelope(raw []byte) ([]byte, []byte, error) {
	var envelope encryptedEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, nil, fmt.Errorf("unmarshal encrypted envelope: %w", err)
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, nil, fmt.Errorf("decode envelope ciphertext: %w", err)
	}
	encryptedDEK, err := base64.RawStdEncoding.DecodeString(envelope.EncryptedDEK)
	if err != nil {
		return nil, nil, fmt.Errorf("decode envelope dek: %w", err)
	}
	return ciphertext, encryptedDEK, nil
}
