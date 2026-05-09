// Package openstore wires the storage backends from configuration.
package openstore

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/watzon/cypra/internal/storage"
	"github.com/watzon/cypra/internal/storage/localdisk"
	"github.com/watzon/cypra/internal/storage/s3compat"
)

const (
	BackendLocalDisk    = "local-disk"
	BackendS3Compatible = "s3-compatible"
)

type Config struct {
	Backend        string
	LocalRoot      string
	LocalSecret    []byte
	S3Bucket       string
	S3Region       string
	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3UsePathStyle bool
}

func LoadConfigFromEnv(masterKey []byte) Config {
	backend := strings.TrimSpace(os.Getenv("STORAGE_BACKEND"))
	if backend == "" {
		backend = BackendLocalDisk
	}
	root := strings.TrimSpace(os.Getenv("STORAGE_LOCAL_PATH"))
	if root == "" {
		root = "cypra-storage"
	}
	return Config{
		Backend:        backend,
		LocalRoot:      root,
		LocalSecret:    masterKey,
		S3Bucket:       os.Getenv("STORAGE_S3_BUCKET"),
		S3Region:       os.Getenv("STORAGE_S3_REGION"),
		S3Endpoint:     os.Getenv("STORAGE_S3_ENDPOINT"),
		S3AccessKey:    os.Getenv("STORAGE_S3_ACCESS_KEY_ID"),
		S3SecretKey:    os.Getenv("STORAGE_S3_SECRET_ACCESS_KEY"),
		S3UsePathStyle: os.Getenv("STORAGE_S3_PATH_STYLE") == "true",
	}
}

func New(cfg Config) (storage.Store, string, error) {
	switch cfg.Backend {
	case BackendLocalDisk, "":
		if cfg.LocalRoot == "" {
			return nil, "", errors.New("storage: STORAGE_LOCAL_PATH is required for local-disk backend")
		}
		return localdisk.Store{Root: cfg.LocalRoot, Secret: cfg.LocalSecret}, BackendLocalDisk, nil
	case BackendS3Compatible:
		if cfg.S3Bucket == "" {
			return nil, "", errors.New("storage: STORAGE_S3_BUCKET is required for s3-compatible backend")
		}
		return s3compat.New(s3compat.Config{
			Bucket:          cfg.S3Bucket,
			Region:          cfg.S3Region,
			Endpoint:        cfg.S3Endpoint,
			AccessKeyID:     cfg.S3AccessKey,
			SecretAccessKey: cfg.S3SecretKey,
			PathStyle:       cfg.S3UsePathStyle,
		}), BackendS3Compatible, nil
	default:
		return nil, "", fmt.Errorf("storage: unknown backend %q", cfg.Backend)
	}
}
