package httpserver

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/storage"
)

const (
	logoMaxBytes      = 5 * 1024 * 1024
	logoMultipartMem  = 1 * 1024 * 1024
	logoFormFieldName = "file"
)

var allowedLogoTypes = []string{"image/png", "image/jpeg", "image/webp"}

// putTenantLogo accepts a multipart upload from instance admins and replaces
// the tenant's stored logo, soft-deleting any prior object.
func (s *Server) putTenantLogo(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, logoMaxBytes+1024)
	if err := r.ParseMultipartForm(logoMultipartMem); err != nil {
		writeError(w, http.StatusBadRequest, "tenant.logo_too_large")
		return
	}
	file, header, err := r.FormFile(logoFormFieldName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.logo_missing")
		return
	}
	defer func() { _ = file.Close() }()
	if header.Size > logoMaxBytes {
		writeError(w, http.StatusBadRequest, "tenant.logo_too_large")
		return
	}
	uploader := storage.Uploader{DB: s.DB, Store: s.Storage, Backend: s.StorageBackend}
	result, err := uploader.Upload(r.Context(), storage.UploadParams{
		TenantID:     tenantID,
		Reader:       file,
		AllowedTypes: allowedLogoTypes,
		MaxSize:      logoMaxBytes,
		KeyPrefix:    "tenants/" + tenantID.String() + "/branding",
	})
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUploadTooLarge):
			writeError(w, http.StatusBadRequest, "tenant.logo_too_large")
		case errors.Is(err, storage.ErrUploadTypeNotAllowed):
			writeError(w, http.StatusBadRequest, "tenant.logo_type_invalid")
		default:
			writeError(w, http.StatusInternalServerError, "tenant.logo_failed")
		}
		return
	}

	prevID, prevKey, prevBackend, prevErr := s.fetchTenantLogo(r.Context(), tenantID)
	if prevErr != nil && !errors.Is(prevErr, sql.ErrNoRows) {
		_ = s.Storage.Delete(r.Context(), result.Key)
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(),
		`UPDATE tenants SET logo_object_id = $1, updated_at = now() WHERE id = $2 AND deleted_at IS NULL`,
		result.ID, tenantID,
	); err != nil {
		_ = s.Storage.Delete(r.Context(), result.Key)
		_, _ = s.DB.ExecContext(r.Context(), `UPDATE storage_objects SET deleted_at = now() WHERE id = $1`, result.ID)
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if prevID != uuid.Nil {
		_, _ = s.DB.ExecContext(r.Context(), `UPDATE storage_objects SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, prevID)
		if prevBackend == s.StorageBackend && prevKey != "" {
			_ = s.Storage.Delete(r.Context(), prevKey)
		}
	}

	slug := s.tenantSlug(r.Context(), tenantID)
	writeJSON(w, http.StatusOK, map[string]any{
		"object_id":    result.ID,
		"content_type": result.ContentType,
		"byte_size":    result.ByteSize,
		"logo_url":     tenantLogoURL(slug),
	})
}

// deleteTenantLogo clears the tenant logo, soft-deleting the storage row and
// removing the underlying object best-effort.
func (s *Server) deleteTenantLogo(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tenant.id_invalid")
		return
	}
	prevID, prevKey, prevBackend, err := s.fetchTenantLogo(r.Context(), tenantID)
	if errors.Is(err, sql.ErrNoRows) || prevID == uuid.Nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if _, err := s.DB.ExecContext(r.Context(),
		`UPDATE tenants SET logo_object_id = NULL, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		tenantID,
	); err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	_, _ = s.DB.ExecContext(r.Context(), `UPDATE storage_objects SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, prevID)
	if prevBackend == s.StorageBackend && prevKey != "" {
		_ = s.Storage.Delete(r.Context(), prevKey)
	}
	w.WriteHeader(http.StatusNoContent)
}

// getBrandingLogo serves the current tenant logo to anonymous clients. The
// path is intentionally stable so hosted-login pages and emails can embed it.
func (s *Server) getBrandingLogo(w http.ResponseWriter, r *http.Request) {
	slug := strings.ToLower(chi.URLParam(r, "slug"))
	if slug == "" {
		writeError(w, http.StatusNotFound, "tenant.logo_not_found")
		return
	}
	var (
		key         string
		contentType string
		objectID    uuid.UUID
		backend     string
	)
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT so.id, so.key, so.content_type, so.backend
		   FROM tenants t
		   JOIN storage_objects so ON so.id = t.logo_object_id
		  WHERE t.slug = $1 AND t.deleted_at IS NULL AND so.deleted_at IS NULL`,
		slug,
	).Scan(&objectID, &key, &contentType, &backend)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "tenant.logo_not_found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db.error")
		return
	}
	if backend != s.StorageBackend {
		// The configured backend changed since this object was uploaded; the
		// bytes live elsewhere and aren't reachable from this server.
		writeError(w, http.StatusNotFound, "tenant.logo_not_found")
		return
	}
	if match := r.Header.Get("If-None-Match"); match != "" && match == `"`+objectID.String()+`"` {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	object, err := s.Storage.Get(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusNotFound, "tenant.logo_not_found")
		return
	}
	if contentType == "" {
		contentType = http.DetectContentType(object.Bytes)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", `"`+objectID.String()+`"`)
	_, _ = w.Write(object.Bytes)
}

func (s *Server) fetchTenantLogo(ctx context.Context, tenantID uuid.UUID) (uuid.UUID, string, string, error) {
	var (
		objectID uuid.NullUUID
		key      sql.NullString
		backend  sql.NullString
	)
	err := s.DB.QueryRowContext(ctx,
		`SELECT t.logo_object_id, so.key, so.backend::text
		   FROM tenants t
		   LEFT JOIN storage_objects so ON so.id = t.logo_object_id
		  WHERE t.id = $1 AND t.deleted_at IS NULL`,
		tenantID,
	).Scan(&objectID, &key, &backend)
	if err != nil {
		return uuid.Nil, "", "", err
	}
	if !objectID.Valid {
		return uuid.Nil, "", "", nil
	}
	return objectID.UUID, key.String, backend.String, nil
}

func (s *Server) tenantSlug(ctx context.Context, tenantID uuid.UUID) string {
	var slug string
	_ = s.DB.QueryRowContext(ctx,
		`SELECT slug FROM tenants WHERE id = $1 AND deleted_at IS NULL`,
		tenantID,
	).Scan(&slug)
	return slug
}

func tenantLogoURL(slug string) string {
	if slug == "" {
		return ""
	}
	return "/api/v1/branding/" + slug + "/logo"
}
