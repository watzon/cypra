# Cypra Go SDK

Go clients for Cypra's admin REST API and tenant OIDC issuers.

## Install

```sh
go get github.com/watzon/cypra/sdk/go/admin
go get github.com/watzon/cypra/sdk/go/oidc
```

## Admin Quickstart

```go
client, err := admin.New("https://cypra.example.com", os.Getenv("CYPRA_PAT"))
if err != nil {
    log.Fatal(err)
}

tenant, err := client.CreateTenant(context.Background(), "acme", "Acme")
```

Admin calls use `Authorization: Bearer <pat>` and typed errors from the root `cypra` package. Use `errors.Is(err, cypra.ErrConflict)` and `errors.As(err, *cypra.Error)` for control flow.

## OIDC Quickstart

```go
client, err := oidc.New(ctx, "https://acme.cypra.example.com", clientID, clientSecret, redirectURI)
if err != nil {
    log.Fatal(err)
}

redirect := client.AuthCodeURL("state-value")
```

The OIDC client is intentionally thin: it delegates discovery and verification to `github.com/coreos/go-oidc/v3/oidc` and token exchange to `golang.org/x/oauth2`.
