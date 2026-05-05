package crypto_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"testing"

	cypra "github.com/watzon/cypra/internal/crypto"
)

func TestEnvelopeEncryptDecryptRoundTrip(t *testing.T) {
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	ciphertext, encryptedDEK, err := cypra.Encrypt([]byte("secret payload"), kek)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plaintext, err := cypra.Decrypt(ciphertext, encryptedDEK, kek)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plaintext) != "secret payload" {
		t.Fatalf("plaintext = %q", plaintext)
	}
}

func TestEnvelopeDecryptWithDifferentKEKFails(t *testing.T) {
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	wrong := bytes.Repeat([]byte{2}, cypra.MasterKeyBytes)
	ciphertext, encryptedDEK, err := cypra.Encrypt([]byte("secret payload"), kek)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	_, err = cypra.Decrypt(ciphertext, encryptedDEK, wrong)
	if !errors.Is(err, cypra.ErrDecryptFailed) {
		t.Fatalf("decrypt error = %v, want %v", err, cypra.ErrDecryptFailed)
	}
}

func TestLoadMasterKey(t *testing.T) {
	key := bytes.Repeat([]byte{7}, cypra.MasterKeyBytes)
	loaded, err := cypra.LoadMasterKey(base64.StdEncoding.EncodeToString(key), "")
	if err != nil {
		t.Fatalf("load env key: %v", err)
	}
	if !bytes.Equal(loaded, key) {
		t.Fatal("loaded key mismatch")
	}

	path := t.TempDir() + "/master.key"
	encoded := base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	loaded, err = cypra.LoadMasterKey("", path)
	if err != nil {
		t.Fatalf("load file key: %v", err)
	}
	if !bytes.Equal(loaded, key) {
		t.Fatal("loaded file key mismatch")
	}
}

func TestLoadMasterKeyAcceptsHexAndRawKeys(t *testing.T) {
	key := bytes.Repeat([]byte{8}, cypra.MasterKeyBytes)
	loaded, err := cypra.LoadMasterKey(hex.EncodeToString(key), "")
	if err != nil {
		t.Fatalf("load hex key: %v", err)
	}
	if !bytes.Equal(loaded, key) {
		t.Fatal("loaded hex key mismatch")
	}

	loaded, err = cypra.LoadMasterKey(string(key), "")
	if err != nil {
		t.Fatalf("load raw key: %v", err)
	}
	if !bytes.Equal(loaded, key) {
		t.Fatal("loaded raw key mismatch")
	}
}

func TestLoadMasterKeyRejectsInvalidSources(t *testing.T) {
	if _, err := cypra.LoadMasterKey("", ""); !errors.Is(err, cypra.ErrInvalidMasterKey) {
		t.Fatalf("empty key error = %v", err)
	}
	if _, err := cypra.LoadMasterKey("short", ""); !errors.Is(err, cypra.ErrInvalidMasterKey) {
		t.Fatalf("short key error = %v", err)
	}
	if _, err := cypra.LoadMasterKey("", t.TempDir()+"/missing.key"); err == nil {
		t.Fatal("missing key file unexpectedly loaded")
	}
}

func TestEnvelopeRejectsInvalidKeysAndCiphertexts(t *testing.T) {
	kek := bytes.Repeat([]byte{1}, cypra.MasterKeyBytes)
	newKEK := bytes.Repeat([]byte{2}, cypra.MasterKeyBytes)
	if _, _, err := cypra.Encrypt([]byte("payload"), []byte("short")); !errors.Is(err, cypra.ErrInvalidMasterKey) {
		t.Fatalf("encrypt error = %v, want invalid key", err)
	}
	if _, err := cypra.Decrypt(nil, nil, []byte("short")); !errors.Is(err, cypra.ErrInvalidMasterKey) {
		t.Fatalf("decrypt key error = %v, want invalid key", err)
	}
	if _, err := cypra.RewrapDEK(nil, []byte("short"), newKEK); !errors.Is(err, cypra.ErrInvalidMasterKey) {
		t.Fatalf("rewrap old key error = %v, want invalid key", err)
	}

	ciphertext, encryptedDEK, err := cypra.Encrypt([]byte("payload"), kek)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := cypra.Decrypt(ciphertext[:4], encryptedDEK, kek); !errors.Is(err, cypra.ErrDecryptFailed) {
		t.Fatalf("short ciphertext error = %v, want decrypt failed", err)
	}
	rewrapped, err := cypra.RewrapDEK(encryptedDEK, kek, newKEK)
	if err != nil {
		t.Fatalf("rewrap dek: %v", err)
	}
	plaintext, err := cypra.Decrypt(ciphertext, rewrapped, newKEK)
	if err != nil {
		t.Fatalf("decrypt rewrapped: %v", err)
	}
	if string(plaintext) != "payload" {
		t.Fatalf("plaintext = %q", plaintext)
	}
}
