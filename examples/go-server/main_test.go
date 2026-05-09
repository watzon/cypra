package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreosoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

type fakeOIDCClient struct{}

func (fakeOIDCClient) AuthCodeURL(state string, _ ...oauth2.AuthCodeOption) string {
	return "https://auth.example/oidc/authorize?state=" + state
}

func (fakeOIDCClient) Exchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "raw-id-token"}), nil
}

func (fakeOIDCClient) Verify(context.Context, string) (*coreosoidc.IDToken, error) {
	return &coreosoidc.IDToken{Subject: "tenant-user-1"}, nil
}

func TestLoginRedirectsWithStateCookie(t *testing.T) {
	server := httptest.NewServer(newApp(fakeOIDCClient{}).routes())
	defer server.Close()

	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Get(server.URL + "/login")
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFound || !strings.HasPrefix(resp.Header.Get("Location"), "https://auth.example/oidc/authorize?state=") {
		t.Fatalf("status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if len(resp.Cookies()) != 1 || resp.Cookies()[0].Name != stateCookieName || resp.Cookies()[0].Value == "" {
		t.Fatalf("state cookies = %#v", resp.Cookies())
	}
}

func TestCallbackCreatesSessionAndRequireAuthServesHome(t *testing.T) {
	app := newApp(fakeOIDCClient{})
	server := httptest.NewServer(app.routes())
	defer server.Close()

	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequest(http.MethodGet, server.URL+"/callback?code=abc&state=state-1", nil)
	if err != nil {
		t.Fatalf("new callback request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "state-1"})
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/" {
		t.Fatalf("callback status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
	var sessionCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("session cookie missing: %#v", resp.Cookies())
	}

	req, err = http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("new home request: %v", err)
	}
	req.AddCookie(sessionCookie)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("home request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("home status = %d", resp.StatusCode)
	}
}

func TestRequireAuthRedirectsMissingAndExpiredSessions(t *testing.T) {
	app := newApp(fakeOIDCClient{})
	server := httptest.NewServer(app.routes())
	defer server.Close()

	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("home request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/login" {
		t.Fatalf("missing session status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	app.sessions["expired"] = session{Subject: "old", Expiry: time.Now().Add(-time.Minute)}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	if err != nil {
		t.Fatalf("new home request: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "expired"})
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("expired home request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/login" {
		t.Fatalf("expired session status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
}
