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

For local Cypra, the issuer is typically `https://acme.cypra.localhost` when running through `make dev`. The managed Playwright harness uses a dynamic `http://acme.localhost:<port>` issuer from discovery.

## Configure Next.js

Use the committed example in `examples/nextjs`.

```sh
cd examples/nextjs
CYPRA_ISSUER=https://acme.cypra.localhost \
CYPRA_CLIENT_ID=client_cypra_acme_console \
CYPRA_CLIENT_SECRET=<copy-from-cypra> \
AUTH_SECRET=dev-secret-change-me \
AUTH_URL=http://localhost:3000 \
AUTH_TRUST_HOST=true \
bun run dev
```

Auth.js discovers Cypra through `/.well-known/openid-configuration`, redirects users to the tenant issuer, and verifies returned tokens with the issuer JWKS.

The committed example maps Cypra `sub` and `email` into `session.cypra`. It sets `idToken: false` so Auth.js calls Cypra's `userinfo_endpoint` and verifies the returned Cypra-issued user claims.

For a deployed Next.js app, register `https://<nextjs-app-domain>/api/auth/callback/cypra` on the Cypra project and run with production values:

```sh
CYPRA_ISSUER=https://acme.<install-domain> \
CYPRA_CLIENT_ID=<project-client-id> \
CYPRA_CLIENT_SECRET=<project-client-secret> \
AUTH_SECRET=<32-byte-random-secret> \
AUTH_URL=https://<nextjs-app-domain> \
AUTH_TRUST_HOST=true \
bun run start
```

## Local Smoke Command

From the repository root:

```sh
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/examples-smoke.spec.ts
```

The smoke creates a tenant, creates a project, updates the redirect URI to the dynamic Next.js callback URL, redeems a real Cypra invite for a user session, signs in through Auth.js, and asserts the resulting session contains Cypra claims.

## Provider Setup

Resend free tier:

1. Create a Resend account.
2. Verify a sender domain, or use Resend's sandbox sender for local validation.
3. Create an API key with send permissions.
4. In Cypra, choose Resend on the Email Provider screen, enter `From address`, `From name`, and the API key, then send the provider test.

Google Cloud Console:

1. Create or select a Google Cloud project.
2. Configure the OAuth consent screen.
3. Create a Web application OAuth client.
4. Add `https://<install-domain>/api/v1/auth/google/callback` as an authorized redirect URI.
5. Paste the Google client ID and secret into Cypra's Upstream Provider screen and enable Google.

## Screenshot Checkpoints

- Cypra project detail shows the issuer URL, client ID, client secret, redirect URI editor, and allowed scopes.
- Auth.js sign-in page shows a single "Cypra" provider button.
- Hosted Cypra consent page is titled "Sign in to this application" and lists requested scopes.
- After allowing consent, the browser returns to the Next.js app and `/api/auth/session` includes `cypra.sub` and `cypra.email`.

## Production Notes

- Use HTTPS for the Next.js callback URL.
- Configure a production email provider before inviting real users.
- Configure Google OAuth redirect URIs in Google Cloud Console if you enable upstream Google sign-in.
- Refetch JWKS when your app sees an unknown `kid`.
