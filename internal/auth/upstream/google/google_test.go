package google_test

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/watzon/cypra/internal/auth/upstream/google"
)

func TestTamperedStateRejected(t *testing.T) {
	client := google.Client{ClientID: "client", RedirectURI: "https://cypra.localhost/callback", Secret: []byte("secret")}
	_, state, err := client.AuthCodeURL(uuid.New(), "/dashboard", time.Minute)
	if err != nil {
		t.Fatalf("auth url: %v", err)
	}
	raw, err := client.SignState(state)
	if err != nil {
		t.Fatalf("sign state: %v", err)
	}
	tampered := strings.Replace(raw, ".", "A.", 1)
	if _, err := client.ValidateState(tampered, time.Now()); !errors.Is(err, google.ErrStateMismatch) {
		t.Fatalf("tampered state error = %v", err)
	}
	got, err := client.ValidateState(raw, time.Now())
	if err != nil || got.Nonce != state.Nonce {
		t.Fatalf("valid state got=%+v err=%v", got, err)
	}
}

func TestIDTokenNonceRequired(t *testing.T) {
	if err := google.ValidateIDTokenNonce(jwtWithNonce("nonce-a"), "nonce-a"); err != nil {
		t.Fatalf("valid nonce: %v", err)
	}
	if err := google.ValidateIDTokenNonce(jwtWithNonce("nonce-b"), "nonce-a"); !errors.Is(err, google.ErrNonceMismatch) {
		t.Fatalf("wrong nonce error = %v", err)
	}
	if err := google.ValidateIDTokenNonce(jwtWithPayload(`{"sub":"user"}`), "nonce-a"); !errors.Is(err, google.ErrNonceMismatch) {
		t.Fatalf("missing nonce error = %v", err)
	}
}

func jwtWithNonce(nonce string) string {
	return jwtWithPayload(`{"nonce":"` + nonce + `"}`)
}

func jwtWithPayload(payload string) string {
	return "header." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".signature"
}
