package crypto_test

import (
	"bytes"
	"encoding/base64"
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
