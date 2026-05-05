// Package storage abstracts binary object storage.
package storage

import (
	"context"
	"time"
)

//revive:disable:exported

type Object struct {
	Key         string
	ContentType string
	Bytes       []byte
}

type Store interface {
	Put(ctx context.Context, object Object) error
	Get(ctx context.Context, key string) (Object, error)
	Delete(ctx context.Context, key string) error
	SignedURL(key string, ttl time.Duration) (string, error)
}
