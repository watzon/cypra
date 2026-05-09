package httpserver_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
)

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

func TestPutTenantLogoStoresAndExposesPublicURL(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	t.Setenv("STORAGE_BACKEND", "local-disk")
	t.Setenv("STORAGE_LOCAL_PATH", root)

	server, err := httpserver.New(httpserver.Options{
		DB: harness.SQL, TenantDB: harness.TenantDB,
		PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test",
		DevOpenAPI: true, KEKLoaded: true, MasterKey: dbtest.TestMasterKey(),
		TrustDevHeaders: true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	body, contentType := multipartBody(t, pngBytes)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/"+tenantID.String()+"/logo", body)
	req.Host = "cypra.localhost"
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT logo = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"logo_url":"/api/v1/branding/acme/logo"`) {
		t.Fatalf("response body = %s", rec.Body.String())
	}

	var key string
	if err := harness.SQL.QueryRow(`SELECT so.key FROM tenants t JOIN storage_objects so ON so.id = t.logo_object_id WHERE t.id = $1`, tenantID).Scan(&key); err != nil {
		t.Fatalf("read storage object: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, key)); err != nil { // #nosec G304 -- test reads a key returned by the storage layer under a temp root.
		t.Fatalf("read written logo: %v", err)
	} else if !bytes.Equal(got, pngBytes) {
		t.Fatalf("logo bytes mismatch")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/branding/acme/logo", nil)
	getReq.Host = "cypra.localhost"
	getRec := httptest.NewRecorder()
	server.Router().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("GET logo = %d body = %s", getRec.Code, getRec.Body.String())
	}
	if getRec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("content type = %q", getRec.Header().Get("Content-Type"))
	}
	if !bytes.Equal(getRec.Body.Bytes(), pngBytes) {
		t.Fatalf("served bytes mismatch")
	}
}

func TestPutTenantLogoReplacesPriorObject(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	t.Setenv("STORAGE_BACKEND", "local-disk")
	t.Setenv("STORAGE_LOCAL_PATH", root)

	server, err := httpserver.New(httpserver.Options{
		DB: harness.SQL, TenantDB: harness.TenantDB,
		PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test",
		DevOpenAPI: true, KEKLoaded: true, MasterKey: dbtest.TestMasterKey(),
		TrustDevHeaders: true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	uploadOnce := func(t *testing.T) (uuid.UUID, string) {
		t.Helper()
		body, contentType := multipartBody(t, pngBytes)
		req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/"+tenantID.String()+"/logo", body)
		req.Host = "cypra.localhost"
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("X-Cypra-Instance-Admin", "true")
		rec := httptest.NewRecorder()
		server.Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("upload = %d body = %s", rec.Code, rec.Body.String())
		}
		var (
			id  uuid.UUID
			key string
		)
		if err := harness.SQL.QueryRow(`SELECT so.id, so.key FROM tenants t JOIN storage_objects so ON so.id = t.logo_object_id WHERE t.id = $1`, tenantID).Scan(&id, &key); err != nil {
			t.Fatalf("read storage object: %v", err)
		}
		return id, key
	}

	firstID, firstKey := uploadOnce(t)
	secondID, secondKey := uploadOnce(t)
	if firstID == secondID {
		t.Fatalf("expected new object id on replacement")
	}
	if _, err := os.Stat(filepath.Join(root, firstKey)); !os.IsNotExist(err) {
		t.Fatalf("expected previous key gone, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, secondKey)); err != nil {
		t.Fatalf("expected new key present, stat err = %v", err)
	}
	var deletedAt *string
	if err := harness.SQL.QueryRow(`SELECT deleted_at::text FROM storage_objects WHERE id = $1`, firstID).Scan(&deletedAt); err != nil {
		t.Fatalf("read deleted_at: %v", err)
	}
	if deletedAt == nil {
		t.Fatalf("expected previous storage_objects row to be soft-deleted")
	}
}

func TestDeleteTenantLogoClears(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	t.Setenv("STORAGE_BACKEND", "local-disk")
	t.Setenv("STORAGE_LOCAL_PATH", root)

	server, err := httpserver.New(httpserver.Options{
		DB: harness.SQL, TenantDB: harness.TenantDB,
		PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test",
		DevOpenAPI: true, KEKLoaded: true, MasterKey: dbtest.TestMasterKey(),
		TrustDevHeaders: true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	body, contentType := multipartBody(t, pngBytes)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/"+tenantID.String()+"/logo", body)
	req.Host = "cypra.localhost"
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	if rec := httptest.NewRecorder(); true {
		server.Router().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("put = %d body = %s", rec.Code, rec.Body.String())
		}
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/"+tenantID.String()+"/logo", nil)
	delReq.Host = "cypra.localhost"
	delReq.Header.Set("X-Cypra-Instance-Admin", "true")
	delRec := httptest.NewRecorder()
	server.Router().ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d body = %s", delRec.Code, delRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/branding/acme/logo", nil)
	getReq.Host = "cypra.localhost"
	getRec := httptest.NewRecorder()
	server.Router().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRec.Code)
	}
}

func TestPutTenantLogoRejectsNonImageType(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	t.Setenv("STORAGE_BACKEND", "local-disk")
	t.Setenv("STORAGE_LOCAL_PATH", root)

	server, err := httpserver.New(httpserver.Options{
		DB: harness.SQL, TenantDB: harness.TenantDB,
		PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test",
		DevOpenAPI: true, KEKLoaded: true, MasterKey: dbtest.TestMasterKey(),
		TrustDevHeaders: true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	body, contentType := multipartBody(t, []byte("just plain text"))
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/"+tenantID.String()+"/logo", body)
	req.Host = "cypra.localhost"
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Cypra-Instance-Admin", "true")
	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "tenant.logo_type_invalid") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func multipartBody(t *testing.T, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := multipart.NewWriter(buffer)
	part, err := writer.CreateFormFile("file", "logo.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return buffer, writer.FormDataContentType()
}
