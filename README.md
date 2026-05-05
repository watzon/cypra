# Cypra

[![Build](https://github.com/watzon/cypra/actions/workflows/build.yml/badge.svg)](https://github.com/watzon/cypra/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

Cypra is an open-source, self-hosted, multi-tenant authentication platform: a Go server with an embedded React dashboard, Postgres storage, hosted login, and a per-tenant OIDC issuer designed to be bootable by a small team in an afternoon.

The product and implementation source of truth lives in [`PLAN.md`](./PLAN.md), [`DESIGN.md`](./DESIGN.md), and [`BRAINSTORM.md`](./BRAINSTORM.md). Execution is tracked in [`TASKS.md`](./TASKS.md).

## Architecture

Cypra ships as one Go binary with an embedded dashboard SPA. The server exposes hosted login, per-tenant OIDC, REST admin APIs, health/readiness checks, metrics, storage URLs, and the dashboard from the same process. Postgres is the only required external data service; storage can be local disk or S3-compatible.

Tenant isolation is enforced at three layers: host-based tenant resolution, `TenantScopedDB`, and Postgres RLS. Each tenant has its own OIDC issuer (`https://<tenant>.<install-domain>`), WebAuthn RP ID, signing-key namespace, users, sessions, and OIDC clients.

Screenshots and visual walkthroughs are captured during the Phase 12/13 local canonical demo. See [`docs/firstrun.md`](./docs/firstrun.md) for the first-run path and [`tests/e2e/lighthouse-baseline.json`](./tests/e2e/lighthouse-baseline.json) for the current hosted-login/dashboard baseline.

## Prerequisites

- Go 1.26.1, with `go.mod` accepting Go 1.23+.
- Bun 1.3.11.
- Docker Compose.
- `portless` for HTTPS and wildcard tenant-subdomain local development.
- `golangci-lint`, `gofumpt`, and `goimports`.

## From Zero To Tests Pass

```sh
git clone https://github.com/watzon/cypra.git
cd cypra
bun install --frozen-lockfile
go mod download
make ci
```

The agent-friendly equivalent is:

```sh
./bin/agent-ci run --quiet --all
```

## Local Development

Start Postgres:

```sh
docker compose -f deploy/docker-compose.yml up -d postgres
```

Run the development target:

```sh
make dev
```

Local HTTPS routes are expected to be served through `portless`:

- `https://cypra.localhost`
- `https://*.cypra.localhost`

## Quick Start With Docker Compose

The reference compose stack is Cypra plus Postgres:

```sh
docker compose -f deploy/docker-compose.yml up
```

For TLS termination with Caddy, use the `with-tls` profile after configuring DNS and `deploy/Caddyfile.example`:

```sh
docker compose -f deploy/docker-compose.yml --profile with-tls up
```

The Railway one-click recipe is documented in [`docs/deploy/railway-template.md`](./docs/deploy/railway-template.md). Publication of the separate Railway template repository is a deployment task.

## Tests

- `make test` runs Go and dashboard tests.
- `make lint` runs Go and dashboard linters.
- `make typecheck` runs Go vet and TypeScript checks.
- `make ci-pipeline` is the exact pipeline used by `./bin/agent-ci` and GitHub Actions.

## Documentation

- [`docs/architecture/overview.md`](./docs/architecture/overview.md) explains the runtime architecture.
- [`docs/security/threat-model.md`](./docs/security/threat-model.md) covers threats, secret handling, and GDPR posture.
- [`docs/contributing/testing.md`](./docs/contributing/testing.md) documents CI and coverage floors.
- [`docs/contributing/extending.md`](./docs/contributing/extending.md) explains auth, email, and storage extension points.
- [`docs/deploy/vps.md`](./docs/deploy/vps.md) describes the single-host VPS recipe.
- [`docs/deploy/observability.md`](./docs/deploy/observability.md) covers metrics, logs, traces, and error tracking.
