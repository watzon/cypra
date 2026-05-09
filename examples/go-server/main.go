package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	coreosoidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/watzon/cypra/sdk/go/oidc"
	"golang.org/x/oauth2"
)

const (
	stateCookieName   = "cypra_example_oauth_state"
	sessionCookieName = "cypra_example_session"
	sessionTTL        = 8 * time.Hour
)

type oidcClient interface {
	AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string
	Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error)
	Verify(ctx context.Context, rawIDToken string) (*coreosoidc.IDToken, error)
}

type session struct {
	Subject string
	Email   string
	Expiry  time.Time
}

type app struct {
	client   oidcClient
	sessions map[string]session
	mu       sync.Mutex
}

func main() {
	issuer := getenv("CYPRA_ISSUER", "https://acme.cypra.localhost")
	clientID := getenv("CYPRA_CLIENT_ID", "client_cypra_acme_console")
	clientSecret := os.Getenv("CYPRA_CLIENT_SECRET")
	redirectURI := getenv("CYPRA_REDIRECT_URI", "http://localhost:9090/callback")

	client, err := oidc.New(context.Background(), issuer, clientID, clientSecret, redirectURI)
	if err != nil {
		log.Fatalf("discover cypra issuer: %v", err)
	}

	server := newApp(client)
	log.Printf("go example listening on http://localhost:9090")
	log.Fatal(http.ListenAndServe(":9090", server.routes()))
}

func newApp(client oidcClient) *app {
	return &app{client: client, sessions: make(map[string]session)}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", a.requireAuth(http.HandlerFunc(a.home)))
	mux.HandleFunc("/login", a.login)
	mux.HandleFunc("/callback", a.callback)
	mux.HandleFunc("/logout", a.logout)
	return mux
}

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	sess := sessionFromContext(r.Context())
	_, _ = fmt.Fprintf(w, "signed in as %s\n", sess.Subject)
}

func (a *app) login(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		http.Error(w, "state generation failed", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: state, Path: "/", MaxAge: 300, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, a.client.AuthCodeURL(state, oauth2.AccessTypeOffline), http.StatusFound)
}

func (a *app) callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" || r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	token, err := a.client.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "code exchange failed", http.StatusBadGateway)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, "missing id token", http.StatusBadGateway)
		return
	}
	idToken, err := a.client.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "invalid id token", http.StatusUnauthorized)
		return
	}
	sessionID, err := randomToken()
	if err != nil {
		http.Error(w, "session generation failed", http.StatusInternalServerError)
		return
	}
	a.mu.Lock()
	a.sessions[sessionID] = session{Subject: idToken.Subject, Expiry: time.Now().Add(sessionTTL)}
	a.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: sessionID, Path: "/", MaxAge: int(sessionTTL.Seconds()), HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *app) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		a.mu.Lock()
		delete(a.sessions, cookie.Value)
		a.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusFound)
}

type sessionKey struct{}

func (a *app) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		a.mu.Lock()
		sess, ok := a.sessions[cookie.Value]
		if ok && time.Now().After(sess.Expiry) {
			delete(a.sessions, cookie.Value)
			ok = false
		}
		a.mu.Unlock()
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), sessionKey{}, sess)))
	})
}

func sessionFromContext(ctx context.Context) session {
	sess, _ := ctx.Value(sessionKey{}).(session)
	return sess
}

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
