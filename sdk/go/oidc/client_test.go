package oidc_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	cypra "github.com/watzon/cypra/sdk/go"
	"github.com/watzon/cypra/sdk/go/oidc"
)

func TestAuthCodeURLUsesDiscoveredIssuer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		issuer := requestURL(r)
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_, _ = w.Write([]byte(`{"issuer":"` + issuer + `","authorization_endpoint":"` + issuer + `/oidc/authorize","token_endpoint":"` + issuer + `/oidc/token","jwks_uri":"` + issuer + `/.well-known/jwks.json","userinfo_endpoint":"` + issuer + `/oidc/userinfo"}`))
		case "/.well-known/jwks.json":
			_, _ = w.Write([]byte(`{"keys":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := oidc.New(context.Background(), server.URL, "client", "secret", "https://app.example/callback")
	if err != nil {
		t.Fatalf("new oidc client: %v", err)
	}
	url := client.AuthCodeURL("state-123")
	if !strings.HasPrefix(url, server.URL+"/oidc/authorize") || !strings.Contains(url, "client_id=client") || !strings.Contains(url, "state=state-123") {
		t.Fatalf("auth url = %s", url)
	}
}

func TestTokenErrorsAreTyped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		issuer := requestURL(r)
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_, _ = w.Write([]byte(`{"issuer":"` + issuer + `","authorization_endpoint":"` + issuer + `/oidc/authorize","token_endpoint":"` + issuer + `/oidc/token","jwks_uri":"` + issuer + `/.well-known/jwks.json","userinfo_endpoint":"` + issuer + `/oidc/userinfo"}`))
		case "/.well-known/jwks.json":
			_, _ = w.Write([]byte(`{"keys":[]}`))
		case "/oidc/token":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"bad code"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := oidc.New(context.Background(), server.URL, "client", "secret", "https://app.example/callback")
	if err != nil {
		t.Fatalf("new oidc client: %v", err)
	}
	_, err = client.Exchange(context.Background(), "bad-code")
	if !errors.Is(err, cypra.ErrValidation) {
		t.Fatalf("expected typed validation error, got %v", err)
	}
	var apiErr *cypra.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "invalid_grant" || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected oidc error metadata, got %#v", err)
	}
}

func requestURL(r *http.Request) string {
	if r.TLS != nil {
		return "https://" + r.Host
	}
	return "http://" + r.Host
}
