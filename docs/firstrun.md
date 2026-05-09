# First Run: Local Cypra To Next.js

This is the local, deployment-free canonical demo path for Cypra v0.1.

## Prerequisites

- Go, Bun, Docker, and local `.localhost` wildcard support.
- A Cypra checkout with dependencies installed.
- The local Postgres service from `deploy/docker-compose.yml`.
- Optional for the browser-validated smoke: Playwright browsers installed through `bunx playwright install chromium`.

## 1. Start The Local Stack

```sh
make dev
```

`make dev` creates `.env` from `.env.example` when needed, starts local compose dependencies, starts portless with wildcard routing, registers `https://cypra.localhost` to Cypra, runs migrations, starts the dashboard Vite server, and starts Cypra. Stop it with `Ctrl-C`; the dev compose services are stopped automatically.

## 2. Redeem Bootstrap

Open `https://cypra.localhost/setup/<setup-token>`, redeem the token, enroll a passkey, and save the backup codes.

## 3. Create Tenant And Project

In the dashboard:

- Create tenant `acme`.
- Create project `console`.
- Copy the issuer URL, client ID, and client secret.

## 4. Configure Providers

For a local run, use terminal email and the Google upstream stub values from the dashboard's Upstream Provider screen.

For a deployed or public demo, configure real providers:

- Resend free tier: create a Resend account, verify a sender domain or the sandbox sender, create an API key, then choose the Resend provider in Cypra with `From address`, `From name`, and the API key. Send a provider test before issuing invites.
- Google Cloud Console: create an OAuth consent screen, create a Web application OAuth client, and add `https://<install-domain>/api/v1/auth/google/callback` as an authorized redirect URI. Copy the Google client ID and secret into the Cypra Upstream Provider screen and enable Google.

## 5. Configure The Next.js Example

```sh
cd examples/nextjs
CYPRA_ISSUER=https://acme.cypra.localhost \
CYPRA_CLIENT_ID=client_cypra_acme_console \
CYPRA_CLIENT_SECRET=<copy-from-project-detail> \
AUTH_SECRET=dev-secret-change-me \
AUTH_URL=http://localhost:3000 \
AUTH_TRUST_HOST=true \
bun run dev
```

Register `http://localhost:3000/api/auth/callback/cypra` on the project before starting the flow.

Open `http://localhost:3000` and choose "Sign in with Cypra".

For a deployed instance, use the same command shape with production values:

```sh
CYPRA_ISSUER=https://acme.<install-domain> \
CYPRA_CLIENT_ID=<project-client-id> \
CYPRA_CLIENT_SECRET=<project-client-secret> \
AUTH_SECRET=<32-byte-random-secret> \
AUTH_URL=https://<nextjs-app-domain> \
AUTH_TRUST_HOST=true \
bun run start
```

## 6. Verify The Browser Smoke

The managed smoke starts Postgres, Cypra, and the Next.js example for you:

```sh
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/examples-smoke.spec.ts tests/e2e/canonical-demo/canonical-demo.spec.ts
```

Expected visual checkpoints:

- Setup screen titled "Set up Cypra".
- Dashboard prompt "Next step: create your first tenant" after backup codes are saved.
- Tenant project detail screen showing issuer, client ID, and client secret controls.
- Hosted consent screen titled "Sign in to this application".
- Next.js example returns to `http://localhost:<port>` with Auth.js session claims populated from Cypra.

## Timing

The unattended Playwright canonical demo has an 8-minute budget and currently records per-step timings in the `canonical-demo-timings` log event. The human-read target is 30 minutes from a clean local stack. Deployment-specific timing, Railway, live Google Console, and live Resend verification are outside the local objective and belong to deployment validation.
