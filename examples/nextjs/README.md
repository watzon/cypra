# Cypra Next.js Example

Reference Next.js 15 + Auth.js OIDC consumer. There is no TypeScript SDK in Cypra v0.1; use Auth.js' generic OIDC provider with the tenant issuer URL.

## Environment

```sh
CYPRA_ISSUER=https://acme.cypra.localhost
CYPRA_CLIENT_ID=client_cypra_acme_console
CYPRA_CLIENT_SECRET=...
AUTH_SECRET=dev-secret-change-me
```

## Run

```sh
bun install
bun run dev
```

Visit `http://localhost:3000` and choose "Sign in with Cypra".
