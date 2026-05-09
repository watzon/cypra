# Cypra Next.js Example

Reference Next.js 15 + Auth.js OIDC consumer. There is no TypeScript SDK in Cypra v0.1; use Auth.js' generic OIDC provider with the tenant issuer URL.

## Environment

```sh
CYPRA_ISSUER=https://acme.cypra.localhost
CYPRA_CLIENT_ID=client_cypra_acme_console
CYPRA_CLIENT_SECRET=...
AUTH_SECRET=dev-secret-change-me
AUTH_URL=http://localhost:3000
AUTH_TRUST_HOST=true
```

## Run

```sh
bun install
bun run dev
```

Register `http://localhost:3000/api/auth/callback/cypra` as an allowed redirect URI on the Cypra OIDC client. Visit `http://localhost:3000` and choose "Sign in with Cypra". The example uses Auth.js' generic OIDC provider and exposes Cypra `sub` / `email` claims on the Auth.js session.
