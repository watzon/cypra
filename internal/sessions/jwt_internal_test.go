package sessions

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	cypra "github.com/watzon/cypra/internal/crypto"
)

func TestSignJWTRejectsAlgorithmKeyMismatches(t *testing.T) {
	tenantID := uuid.New()
	kek := bytes.Repeat([]byte{4}, cypra.MasterKeyBytes)
	generated, err := cypra.GenerateSigningKey(tenantID, 1, cypra.SigningAlgRS256, kek, time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	claims := AccessTokenClaims{Issuer: "https://acme.cypra.localhost", Subject: tenantID.String() + ":" + uuid.NewString(), IssuedAt: 100, Expires: 200, NotBefore: 100, Audience: "client", Scope: []string{"openid"}}

	if _, err := signJWT(jwtHeader{Algorithm: cypra.SigningAlgES256, Type: "JWT", KID: generated.KID}, claims, generated.PrivateKeyEncrypted, kek); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("es256/rsa mismatch error = %v, want invalid token", err)
	}
	if _, err := signJWT(jwtHeader{Algorithm: "HS256", Type: "JWT", KID: generated.KID}, claims, generated.PrivateKeyEncrypted, kek); !errors.Is(err, cypra.ErrUnsupportedSigningAlgorithm) {
		t.Fatalf("unsupported signing error = %v", err)
	}
}

func TestVerifySignatureRejectsMalformedJWKValues(t *testing.T) {
	if err := verifySignature(jwtHeader{Algorithm: cypra.SigningAlgRS256}, "header.claims", []byte("signature"), []byte(`{"n":"%%%","e":"AQAB"}`)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("bad rsa n error = %v", err)
	}
	if err := verifySignature(jwtHeader{Algorithm: cypra.SigningAlgRS256}, "header.claims", []byte("signature"), []byte(`{"n":"AQAB","e":"%%%"}`)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("bad rsa e error = %v", err)
	}
	if err := verifySignature(jwtHeader{Algorithm: cypra.SigningAlgES256}, "header.claims", make([]byte, 64), []byte(`{"x":"%%%","y":"AQAB"}`)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("bad es256 x error = %v", err)
	}
	if err := verifySignature(jwtHeader{Algorithm: cypra.SigningAlgES256}, "header.claims", make([]byte, 64), []byte(`{"x":"AQAB","y":"%%%"}`)); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("bad es256 y error = %v", err)
	}
}

func TestRefreshServicePropagatesClosedDatabaseErrors(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://invalid")
	if err != nil {
		t.Fatalf("open db handle: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db handle: %v", err)
	}
	service := NewRefreshService(db)
	ctx := context.Background()
	parentID := uuid.New()

	if _, err := service.Mint(ctx, uuid.New(), uuid.New(), uuid.New(), []string{"openid"}, nil, time.Unix(200, 0).UTC()); err == nil {
		t.Fatal("Mint unexpectedly succeeded on closed db")
	}
	if _, err := service.Mint(ctx, uuid.New(), uuid.New(), uuid.New(), []string{"openid"}, &parentID, time.Unix(200, 0).UTC()); err == nil {
		t.Fatal("Mint with parent unexpectedly succeeded on closed db")
	}
	if _, err := service.Consume(ctx, "token", time.Unix(200, 0).UTC()); err == nil {
		t.Fatal("Consume unexpectedly succeeded on closed db")
	}
}
