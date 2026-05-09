// Package crypto contains Cypra's password hashing and envelope encryption primitives.
package crypto

//revive:disable:exported

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	Argon2MemoryKiB uint32 = 19 * 1024
	Argon2Time      uint32 = 2
	Argon2Threads   uint8  = 1
	Argon2SaltBytes        = 16
	Argon2KeyBytes  uint32 = 32
)

var (
	ErrPasswordRejected = errors.New("password rejected")
	ErrPasswordMismatch = errors.New("password mismatch")
	ErrInvalidHash      = errors.New("invalid argon2id hash")
)

type PasswordDenylist func(password string) error

func RejectCommonPasswords(password string) error {
	common := map[string]struct{}{
		"123456":     {},
		"password":   {},
		"qwerty":     {},
		"letmein":    {},
		"admin":      {},
		"welcome":    {},
		"password1":  {},
		"iloveyou":   {},
		"cypra":      {},
		"changeme":   {},
		"monkey":     {},
		"dragon":     {},
		"football":   {},
		"baseball":   {},
		"trustno1":   {},
		"000000":     {},
		"111111":     {},
		"abc123":     {},
		"passw0rd":   {},
		"superadmin": {},
	}
	if _, ok := common[strings.ToLower(strings.TrimSpace(password))]; ok {
		return ErrPasswordRejected
	}
	return nil
}

func HashPassword(password string, denylist PasswordDenylist) (string, error) {
	if denylist != nil {
		if err := denylist(password); err != nil {
			return "", err
		}
	}
	salt := make([]byte, Argon2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate argon2id salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, Argon2Time, Argon2MemoryKiB, Argon2Threads, Argon2KeyBytes)
	return encodeArgon2id(salt, hash), nil
}

func VerifyPassword(encoded, password string) error {
	params, salt, want, err := decodeArgon2id(encoded)
	if err != nil {
		return err
	}
	if len(want) > int(^uint32(0)) {
		return ErrInvalidHash
	}
	got := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(want))) // #nosec G115 -- guarded above.
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}

type argon2Params struct {
	memory  uint32
	time    uint32
	threads uint8
}

func encodeArgon2id(salt, hash []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		Argon2MemoryKiB,
		Argon2Time,
		Argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

func decodeArgon2id(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	params, err := parseArgon2Params(parts[3])
	if err != nil {
		return argon2Params{}, nil, nil, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("decode argon2id salt: %w", ErrInvalidHash)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, fmt.Errorf("decode argon2id hash: %w", ErrInvalidHash)
	}
	if len(salt) == 0 || len(hash) == 0 {
		return argon2Params{}, nil, nil, ErrInvalidHash
	}
	return params, salt, hash, nil
}

func parseArgon2Params(raw string) (argon2Params, error) {
	values := map[string]string{}
	for _, part := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			return argon2Params{}, ErrInvalidHash
		}
		values[key] = value
	}
	memory, err := parseUint32(values["m"])
	if err != nil {
		return argon2Params{}, ErrInvalidHash
	}
	time, err := parseUint32(values["t"])
	if err != nil {
		return argon2Params{}, ErrInvalidHash
	}
	threads64, err := strconv.ParseUint(values["p"], 10, 8)
	if err != nil || threads64 == 0 {
		return argon2Params{}, ErrInvalidHash
	}
	return argon2Params{memory: memory, time: time, threads: uint8(threads64)}, nil
}

func parseUint32(raw string) (uint32, error) {
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 {
		return 0, ErrInvalidHash
	}
	return uint32(value), nil
}
