package localdisk_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/watzon/cypra/internal/storage"
	"github.com/watzon/cypra/internal/storage/localdisk"
)

func TestLocalDiskRoundTripAndSignedURL(t *testing.T) {
	store := localdisk.Store{Root: t.TempDir(), Host: "https://acme.cypra.localhost", Secret: []byte("secret")}
	if err := store.Put(context.Background(), storage.Object{Key: "profiles/user.txt", Bytes: []byte("hello")}); err != nil {
		t.Fatalf("put: %v", err)
	}
	object, err := store.Get(context.Background(), "profiles/user.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(object.Bytes, []byte("hello")) {
		t.Fatalf("object bytes = %q", object.Bytes)
	}
	signed, err := store.SignedURL("profiles/user.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("signed url: %v", err)
	}
	key, err := store.VerifySignedPath(signed[len("https://acme.cypra.localhost"):])
	if err != nil {
		t.Fatalf("verify signed path: %v", err)
	}
	if key != "profiles/user.txt" {
		t.Fatalf("signed key = %q", key)
	}
	if _, err := store.VerifySignedPath("/storage/tampered.bad"); err == nil {
		t.Fatal("tampered signed URL unexpectedly verified")
	}
}
