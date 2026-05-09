# Cypra Go Server Example

Minimal HTTP server protected by a Cypra tenant OIDC issuer. It demonstrates a standard server-side OIDC flow: redirect to Cypra, handle the authorization-code callback, exchange the code, verify the ID token, store a local cookie session, and protect routes with `requireAuth` middleware.

## Run

```sh
CYPRA_ISSUER=https://acme.cypra.localhost \
CYPRA_CLIENT_ID=client_cypra_acme_console \
CYPRA_CLIENT_SECRET=... \
CYPRA_REDIRECT_URI=http://localhost:9090/callback \
go run .
```

Register `http://localhost:9090/callback` as an allowed redirect URI on the Cypra OIDC client. Open `http://localhost:9090`. Without a local example session, the server redirects to `/login`, then to Cypra authorization. After Cypra redirects back to `/callback`, the example exchanges the code, verifies the ID token, stores a local session cookie, and serves the protected home page through `requireAuth`.
