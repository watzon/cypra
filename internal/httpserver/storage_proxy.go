package httpserver

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/watzon/cypra/internal/observability"
	"github.com/watzon/cypra/internal/storage/localdisk"
	"github.com/watzon/cypra/internal/storage/openstore"
	"go.opentelemetry.io/otel/attribute"
)

func (s *Server) storageProxy(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	defer func() { recordStorageOperation("local-disk", "get", time.Since(started)) }()
	ctx, span := observability.StartSpan(r.Context(), "storage.get", attribute.String("storage.backend", "local-disk"))
	defer span.End()
	r = r.WithContext(ctx)
	if s.StorageBackend != openstore.BackendLocalDisk {
		writeError(w, http.StatusNotFound, "storage.not_found")
		return
	}
	store, ok := s.Storage.(localdisk.Store)
	if !ok {
		writeError(w, http.StatusNotFound, "storage.not_found")
		return
	}
	key, err := store.VerifySignedPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusForbidden, "storage.signature_invalid")
		return
	}
	object, err := store.Get(r.Context(), key)
	if err != nil {
		observability.RecordError(span, err)
		if errors.Is(err, os.ErrNotExist) {
			writeError(w, http.StatusNotFound, "storage.not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "storage.read_failed")
		return
	}
	contentType := object.ContentType
	if contentType == "" {
		_ = s.DB.QueryRowContext(r.Context(), `SELECT content_type FROM storage_objects WHERE key = $1 AND deleted_at IS NULL LIMIT 1`, key).Scan(&contentType)
	}
	if contentType == "" {
		contentType = http.DetectContentType(object.Bytes)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(object.Bytes)
}
