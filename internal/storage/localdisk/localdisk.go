// Package localdisk stores objects on local disk and signs proxy URLs.
package localdisk

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/watzon/cypra/internal/storage"
)

//revive:disable:exported

var ErrInvalidSignature = errors.New("invalid storage signature")

type Store struct {
	Root   string
	Host   string
	Secret []byte
}

func (s Store) Put(_ context.Context, object storage.Object) error {
	path, err := s.path(object.Key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, object.Bytes, 0o600)
}

func (s Store) Get(_ context.Context, key string) (storage.Object, error) {
	path, err := s.path(key)
	if err != nil {
		return storage.Object{}, err
	}
	content, err := os.ReadFile(path) // #nosec G304 -- path is constrained under Store.Root by path().
	if err != nil {
		return storage.Object{}, err
	}
	return storage.Object{Key: key, Bytes: content}, nil
}

func (s Store) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (s Store) SignedURL(key string, ttl time.Duration) (string, error) {
	if ttl > time.Hour {
		ttl = time.Hour
	}
	payload, err := json.Marshal(map[string]any{"key": key, "exp": time.Now().Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	sig := s.sign(encoded)
	return strings.TrimRight(s.Host, "/") + "/storage/" + encoded + "." + sig, nil
}

func (s Store) VerifySignedPath(raw string) (string, error) {
	encoded, sig, ok := strings.Cut(strings.TrimPrefix(raw, "/storage/"), ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.sign(encoded))) {
		return "", ErrInvalidSignature
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidSignature
	}
	var claims struct {
		Key string `json:"key"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || time.Now().Unix() > claims.Exp {
		return "", ErrInvalidSignature
	}
	return claims.Key, nil
}

func (s Store) path(key string) (string, error) {
	cleaned := filepath.Clean("/" + key)
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, strings.TrimPrefix(cleaned, "/"))
	if !strings.HasPrefix(path, root) {
		return "", fmt.Errorf("invalid storage key")
	}
	return path, nil
}

func (s Store) sign(encodedPayload string) string {
	mac := hmac.New(sha256.New, s.Secret)
	_, _ = mac.Write([]byte(encodedPayload))
	return url.QueryEscape(base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))
}
