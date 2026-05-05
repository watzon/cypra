# First Run: Local Cypra To Next.js

This is the local, deployment-free canonical demo path for Cypra v0.1.

## Prerequisites

- Go, Bun, Docker, and local `.localhost` wildcard support.
- A Cypra checkout with dependencies installed.
- The local Postgres service from `deploy/docker-compose.yml`.

## 1. Start The Local Stack

```sh
docker compose -f deploy/docker-compose.yml up -d postgres
PUBLIC_BASE_URL=https://cypra.localhost \
DATABASE_URL=postgres://cypra:cypra@localhost:54320/cypra?sslmode=disable \
MIGRATE_DATABASE_URL=postgres://cypra:cypra@localhost:54320/cypra?sslmode=disable \
MASTER_KEY=dev-only-change-me-dev-only-change-me-32b \
STORAGE_BACKEND=local-disk \
STORAGE_LOCAL_PATH=.data/storage \
go run ./cmd/cypra serve
```

## 2. Redeem Bootstrap

Open `https://cypra.localhost/setup/<setup-token>`, redeem the token, enroll a passkey, and save the backup codes.

## 3. Create Tenant And Project

In the dashboard:

- Create tenant `acme`.
- Create project `console`.
- Copy the issuer URL, client ID, and client secret.

## 4. Configure Providers

For a local run, use terminal email and the Google upstream stub values from the dashboard's Upstream Provider screen.

## 5. Configure The Next.js Example

```sh
cd examples/nextjs
CYPRA_ISSUER=https://acme.cypra.localhost \
CYPRA_CLIENT_ID=client_cypra_acme_console \
CYPRA_CLIENT_SECRET=<copy-from-project-detail> \
AUTH_SECRET=dev-secret-change-me \
bun run dev
```

Open `http://localhost:3000` and choose "Sign in with Cypra".

## Timing

The unattended Playwright canonical demo has an 8-minute budget. The human-read target is 30 minutes from a clean local stack. Deployment-specific timing, Railway, live Google Console, and live Resend verification are outside the local objective and belong to deployment validation.
