# From Zero To Next.js In An Afternoon

This tutorial connects a Next.js 15 app to a local Cypra tenant using Auth.js' generic OIDC provider.

## Create The Tenant

Start Cypra locally, redeem the bootstrap token, and create tenant `acme`.

## Create The Project

Create a project named `Console App` with redirect URI:

```txt
http://localhost:3000/api/auth/callback/cypra
```

Copy the issuer URL, client ID, and client secret from the project detail screen.

## Configure Next.js

Use the committed example in `examples/nextjs`.

```sh
cd examples/nextjs
CYPRA_ISSUER=https://acme.cypra.localhost \
CYPRA_CLIENT_ID=client_cypra_acme_console \
CYPRA_CLIENT_SECRET=<copy-from-cypra> \
AUTH_SECRET=dev-secret-change-me \
bun run dev
```

Auth.js discovers Cypra through `/.well-known/openid-configuration`, redirects users to the tenant issuer, and verifies returned tokens with the issuer JWKS.

## Production Notes

- Use HTTPS for the Next.js callback URL.
- Configure a production email provider before inviting real users.
- Configure Google OAuth redirect URIs in Google Cloud Console if you enable upstream Google sign-in.
- Refetch JWKS when your app sees an unknown `kid`.
