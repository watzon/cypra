package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

//revive:disable:exported

var (
	ErrUploadTooLarge       = errors.New("storage: upload exceeds size limit")
	ErrUploadTypeNotAllowed = errors.New("storage: upload content type not allowed")
)

type UploadParams struct {
	TenantID     uuid.UUID
	OwnerUserID  *uuid.UUID
	Reader       io.Reader
	AllowedTypes []string
	MaxSize      int64
	KeyPrefix    string
}

type UploadResult struct {
	ID          uuid.UUID
	Key         string
	ContentType string
	ByteSize    int64
	Backend     string
}

type Uploader struct {
	DB      *sql.DB
	Store   Store
	Backend string
}

func (u Uploader) Upload(ctx context.Context, params UploadParams) (*UploadResult, error) {
	if u.Store == nil || u.DB == nil || u.Backend == "" {
		return nil, errors.New("storage: uploader not configured")
	}
	if params.MaxSize <= 0 {
		return nil, errors.New("storage: MaxSize must be > 0")
	}
	limited := io.LimitReader(params.Reader, params.MaxSize+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	if int64(len(body)) > params.MaxSize {
		return nil, ErrUploadTooLarge
	}
	contentType := http.DetectContentType(body)
	if len(params.AllowedTypes) > 0 && !contains(params.AllowedTypes, contentType) {
		return nil, fmt.Errorf("%w: %s", ErrUploadTypeNotAllowed, contentType)
	}
	id := uuid.New()
	key := buildKey(params.KeyPrefix, id, contentType)
	if err := u.Store.Put(ctx, Object{Key: key, ContentType: contentType, Bytes: body}); err != nil {
		return nil, fmt.Errorf("put object: %w", err)
	}
	if _, err := u.DB.ExecContext(ctx,
		`INSERT INTO storage_objects (id, tenant_id, owner_user_id, backend, key, content_type, byte_size)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, params.TenantID, ownerArg(params.OwnerUserID), u.Backend, key, contentType, len(body),
	); err != nil {
		// Best-effort cleanup of the orphaned blob.
		_ = u.Store.Delete(ctx, key)
		return nil, fmt.Errorf("record storage object: %w", err)
	}
	return &UploadResult{
		ID:          id,
		Key:         key,
		ContentType: contentType,
		ByteSize:    int64(len(body)),
		Backend:     u.Backend,
	}, nil
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func ownerArg(owner *uuid.UUID) any {
	if owner == nil {
		return nil
	}
	return *owner
}

func buildKey(prefix string, id uuid.UUID, contentType string) string {
	cleaned := prefix
	if cleaned != "" && cleaned[len(cleaned)-1] != '/' {
		cleaned += "/"
	}
	return cleaned + id.String() + ExtensionForContentType(contentType)
}

// ExtensionForContentType returns a file extension (with leading dot) for a
// known image MIME type, or "" if the type is unknown.
func ExtensionForContentType(contentType string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}

// SniffImageType returns the detected MIME type if the bytes look like one of
// the supported raster images, or "" otherwise.
func SniffImageType(b []byte) string {
	switch t := http.DetectContentType(b); t {
	case "image/png", "image/jpeg", "image/webp", "image/gif":
		return t
	default:
		return ""
	}
}

