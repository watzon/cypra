package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/dbtest"
	"github.com/watzon/cypra/internal/httpserver"
	"github.com/watzon/cypra/internal/storage"
	"github.com/watzon/cypra/internal/storage/localdisk"
)

func TestStorageProxyValidSignedLocalDiskURL(t *testing.T) {
	harness := dbtest.New(t)
	tenantID := dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	t.Setenv("STORAGE_BACKEND", "local-disk")
	t.Setenv("STORAGE_LOCAL_PATH", root)
	store := localdisk.Store{Root: root, Host: "https://acme.cypra.localhost", Secret: dbtest.TestMasterKey()}
	if err := store.Put(context.Background(), storage.Object{Key: "profiles/user.txt", ContentType: "text/plain", Bytes: []byte("hello profile")}); err != nil {
		t.Fatalf("put object: %v", err)
	}
	if _, err := harness.SQL.Exec(`INSERT INTO storage_objects (id, tenant_id, backend, key, content_type, byte_size) VALUES ($1, $2, 'local-disk', 'profiles/user.txt', 'text/plain', 13)`, uuid.New(), tenantID); err != nil {
		t.Fatalf("seed storage object: %v", err)
	}
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, MasterKey: dbtest.TestMasterKey(), TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	signed, err := store.SignedURL("profiles/user.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("sign url: %v", err)
	}
	path := signedPath(t, signed)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = "acme.cypra.localhost"
	resp := httptest.NewRecorder()
	server.Router().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "hello profile" || resp.Header().Get("Content-Type") != "text/plain" {
		t.Fatalf("storage proxy = %d body=%q content-type=%q", resp.Code, resp.Body.String(), resp.Header().Get("Content-Type"))
	}
}

func TestStorageProxyRejectsInvalidSignedURLs(t *testing.T) {
	harness := dbtest.New(t)
	dbtest.SeedTenant(t, harness.SQL, "acme")
	root := t.TempDir()
	t.Setenv("STORAGE_BACKEND", "local-disk")
	t.Setenv("STORAGE_LOCAL_PATH", root)
	store := localdisk.Store{Root: root, Host: "https://acme.cypra.localhost", Secret: dbtest.TestMasterKey()}
	server, err := httpserver.New(httpserver.Options{DB: harness.SQL, TenantDB: harness.TenantDB, PublicBaseURL: "https://cypra.localhost", Version: "test", Commit: "test", KEKLoaded: true, MasterKey: dbtest.TestMasterKey(), TrustDevHeaders: true})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	expired, err := store.SignedURL("profiles/user.txt", -time.Minute)
	if err != nil {
		t.Fatalf("sign expired url: %v", err)
	}
	valid, err := store.SignedURL("profiles/user.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("sign valid url: %v", err)
	}
	paths := []string{
		"/storage/not-a-valid-token",
		signedPath(t, expired),
		signedPath(t, valid) + "x",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "acme.cypra.localhost"
		resp := httptest.NewRecorder()
		server.Router().ServeHTTP(resp, req)
		if resp.Code != http.StatusForbidden || !strings.Contains(resp.Body.String(), "storage.signature_invalid") {
			t.Fatalf("%s = %d %s", path, resp.Code, resp.Body.String())
		}
	}
}

func signedPath(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse signed url: %v", err)
	}
	return parsed.Path
}
