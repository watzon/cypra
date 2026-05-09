package crypto_test

import (
	"errors"
	"strings"
	"testing"

	cypra "github.com/watzon/cypra/internal/crypto"
)

func TestArgon2idHashVerifyRoundTrip(t *testing.T) {
	encoded, err := cypra.HashPassword("correct horse battery staple", cypra.RejectCommonPasswords)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Fatalf("hash prefix = %q, want argon2id PHC", encoded)
	}
	if err := cypra.VerifyPassword(encoded, "correct horse battery staple"); err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if err := cypra.VerifyPassword(encoded, "wrong"); !errors.Is(err, cypra.ErrPasswordMismatch) {
		t.Fatalf("wrong password error = %v, want %v", err, cypra.ErrPasswordMismatch)
	}
}

func TestArgon2idRejectsCommonPasswords(t *testing.T) {
	_, err := cypra.HashPassword("password", cypra.RejectCommonPasswords)
	if !errors.Is(err, cypra.ErrPasswordRejected) {
		t.Fatalf("common password error = %v, want %v", err, cypra.ErrPasswordRejected)
	}
}

func TestArgon2idRejectsInvalidHash(t *testing.T) {
	if err := cypra.VerifyPassword("not-a-phc-hash", "password"); !errors.Is(err, cypra.ErrInvalidHash) {
		t.Fatalf("invalid hash error = %v, want %v", err, cypra.ErrInvalidHash)
	}
}

func TestArgon2idRejectsMalformedPHCParts(t *testing.T) {
	badHashes := []string{
		"$argon2id$v=19$m=65536,t=3$bad$hash",
		"$argon2id$v=19$m=bad,t=3,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=bad,p=4$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=bad$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=4$%%%$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$%%%",
		"$argon2id$v=19$m=65536,t=3,p=4$$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$",
	}

	for _, hash := range badHashes {
		if err := cypra.VerifyPassword(hash, "password"); !errors.Is(err, cypra.ErrInvalidHash) {
			t.Fatalf("%q error = %v, want %v", hash, err, cypra.ErrInvalidHash)
		}
	}
}
