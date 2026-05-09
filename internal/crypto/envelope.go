package crypto

//revive:disable:exported

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	MasterKeyBytes    = 32
	DEKBytes          = 32
	gcmNonceBytes     = 12
	encryptedDEKBytes = gcmNonceBytes + DEKBytes + 16
)

var (
	ErrInvalidMasterKey = errors.New("invalid master key")
	ErrDecryptFailed    = errors.New("decrypt failed")
)

func LoadMasterKey(envValue, filePath string) ([]byte, error) {
	var raw string
	if filePath != "" {
		content, err := os.ReadFile(filePath) // #nosec G304 -- operator-provided master key file path.
		if err != nil {
			return nil, fmt.Errorf("read master key file: %w", err)
		}
		raw = strings.TrimSpace(string(content))
	} else {
		raw = strings.TrimSpace(envValue)
	}
	key, err := decodeKey(raw)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func Encrypt(plaintext, kek []byte) ([]byte, []byte, error) {
	if len(kek) != MasterKeyBytes {
		return nil, nil, ErrInvalidMasterKey
	}
	dek := make([]byte, DEKBytes)
	if _, err := rand.Read(dek); err != nil {
		return nil, nil, fmt.Errorf("generate dek: %w", err)
	}
	ciphertext, err := seal(plaintext, dek)
	if err != nil {
		return nil, nil, err
	}
	wrappedDEK, err := seal(dek, kek)
	if err != nil {
		return nil, nil, err
	}
	return ciphertext, wrappedDEK, nil
}

func Decrypt(ciphertext, encryptedDEK, kek []byte) ([]byte, error) {
	if len(kek) != MasterKeyBytes {
		return nil, ErrInvalidMasterKey
	}
	dek, err := open(encryptedDEK, kek)
	if err != nil {
		return nil, err
	}
	return open(ciphertext, dek)
}

func RewrapDEK(encryptedDEK, oldKEK, newKEK []byte) ([]byte, error) {
	if len(oldKEK) != MasterKeyBytes || len(newKEK) != MasterKeyBytes {
		return nil, ErrInvalidMasterKey
	}
	dek, err := open(encryptedDEK, oldKEK)
	if err != nil {
		return nil, err
	}
	return seal(dek, newKEK)
}

func seal(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new aes-gcm: %w", err)
	}
	nonce := make([]byte, gcmNonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	sealed := aead.Seal(nil, nonce, plaintext, nil)
	return append(nonce, sealed...), nil
}

func open(ciphertext, key []byte) ([]byte, error) {
	if len(ciphertext) <= gcmNonceBytes {
		return nil, ErrDecryptFailed
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new aes-gcm: %w", err)
	}
	nonce := ciphertext[:gcmNonceBytes]
	payload := ciphertext[gcmNonceBytes:]
	plaintext, err := aead.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plaintext, nil
}

func decodeKey(raw string) ([]byte, error) {
	if raw == "" {
		return nil, ErrInvalidMasterKey
	}
	decoders := []func(string) ([]byte, error){
		base64.StdEncoding.DecodeString,
		base64.RawStdEncoding.DecodeString,
		hex.DecodeString,
	}
	for _, decoder := range decoders {
		key, err := decoder(raw)
		if err == nil && len(key) == MasterKeyBytes {
			return key, nil
		}
	}
	if len([]byte(raw)) == MasterKeyBytes {
		return []byte(raw), nil
	}
	return nil, ErrInvalidMasterKey
}
