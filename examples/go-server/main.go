package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/watzon/cypra/sdk/go/oidc"
)

func main() {
	issuer := getenv("CYPRA_ISSUER", "https://acme.cypra.localhost")
	clientID := getenv("CYPRA_CLIENT_ID", "client_cypra_acme_console")
	clientSecret := os.Getenv("CYPRA_CLIENT_SECRET")
	redirectURI := getenv("CYPRA_REDIRECT_URI", "http://localhost:9090/callback")

	client, err := oidc.New(context.Background(), issuer, clientID, clientSecret, redirectURI)
	if err != nil {
		log.Fatalf("discover cypra issuer: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Redirect(w, r, client.AuthCodeURL("example-state"), http.StatusFound)
			return
		}
		idToken, err := client.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		_, _ = fmt.Fprintf(w, "signed in as %s\n", idToken.Subject)
	})

	log.Printf("go example listening on http://localhost:9090")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
