# Cypra OIDC SDK

Thin wrapper around `golang.org/x/oauth2` and `github.com/coreos/go-oidc/v3/oidc` for Cypra tenant issuers.

## Usage

```go
client, err := oidc.New(ctx, "https://acme.cypra.example.com", clientID, clientSecret, redirectURI)
if err != nil {
    log.Fatal(err)
}

http.Redirect(w, r, client.AuthCodeURL(state), http.StatusFound)
```

Use `Exchange` for the callback code, `UserInfo` for claims from the userinfo endpoint, `RefreshToken` for refresh grants, and `Verify` for raw ID tokens.
