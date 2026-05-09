package storage_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/storage"
	"github.com/watzon/cypra/internal/storage/localdisk"
)

// pngBytes is a real 1x1 PNG so http.DetectContentType returns image/png.
var pngBytes = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

func TestUploaderHappyPath(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	store := localdisk.Store{Root: root, Secret: dbtest.TestMasterKey()}
	uploader := storage.Uploader{DB: harness.SQL, Store: store, Backend: "local-disk"}
	result, err := uploader.Upload(context.Background(), storage.UploadParams{
		TenantID:     tenantID,
		Reader:       bytes.NewReader(pngBytes),
		AllowedTypes: []string{"image/png"},
		MaxSize:      1024 * 1024,
		KeyPrefix:    "tenants/" + tenantID.String() + "/branding",
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if result.ContentType != "image/png" {
		t.Fatalf("content type = %q", result.ContentType)
	}
	if result.ByteSize != int64(len(pngBytes)) {
		t.Fatalf("byte size = %d", result.ByteSize)
	}
	if got, err := os.ReadFile(filepath.Join(root, result.Key)); err != nil { // #nosec G304 -- test reads the uploaded key under a temp root.
		t.Fatalf("read written file: %v", err)
	} else if !bytes.Equal(got, pngBytes) {
		t.Fatalf("file contents mismatch")
	}
	var rows int
	if err := harness.SQL.QueryRow(`SELECT count(*) FROM storage_objects WHERE id = $1 AND tenant_id = $2 AND backend = 'local-disk' AND deleted_at IS NULL`, result.ID, tenantID).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 storage_objects row, got %d", rows)
	}
}

func TestUploaderRejectsOversizedUpload(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	store := localdisk.Store{Root: t.TempDir(), Secret: dbtest.TestMasterKey()}
	uploader := storage.Uploader{DB: harness.SQL, Store: store, Backend: "local-disk"}
	_, err := uploader.Upload(context.Background(), storage.UploadParams{
		TenantID:     tenantID,
		Reader:       bytes.NewReader(pngBytes),
		AllowedTypes: []string{"image/png"},
		MaxSize:      int64(len(pngBytes) - 1),
	})
	if !errors.Is(err, storage.ErrUploadTooLarge) {
		t.Fatalf("expected ErrUploadTooLarge, got %v", err)
	}
}

func TestUploaderRejectsDisallowedType(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	store := localdisk.Store{Root: t.TempDir(), Secret: dbtest.TestMasterKey()}
	uploader := storage.Uploader{DB: harness.SQL, Store: store, Backend: "local-disk"}
	_, err := uploader.Upload(context.Background(), storage.UploadParams{
		TenantID:     tenantID,
		Reader:       bytes.NewReader([]byte("not an image")),
		AllowedTypes: []string{"image/png"},
		MaxSize:      1024,
	})
	if !errors.Is(err, storage.ErrUploadTypeNotAllowed) {
		t.Fatalf("expected ErrUploadTypeNotAllowed, got %v", err)
	}
}
