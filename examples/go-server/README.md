# Cypra Go Server Example

Minimal HTTP server protected by a Cypra tenant OIDC issuer.

## Run

```sh
CYPRA_ISSUER=https://acme.cypra.localhost \
CYPRA_CLIENT_ID=client_cypra_acme_console \
CYPRA_CLIENT_SECRET=... \
go run .
```

Open `http://localhost:9090`. Without a bearer token, the server redirects to Cypra authorization. With a bearer ID token, it verifies the token through the Go SDK and prints the subject.
