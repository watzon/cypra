# Cypra

[![Build](https://github.com/watzon/cypra/actions/workflows/build.yml/badge.svg)](https://github.com/watzon/cypra/actions/workflows/build.yml)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

Cypra is a self-hosted, multi-tenant authentication server for technical operators who want a small, inspectable alternative to managed auth platforms and heavyweight identity stacks.

It ships as one Go server with an embedded React dashboard, Postgres persistence, hosted login pages, passkeys, password and magic-link flows, tenant-scoped OIDC issuers, audit logs, backup/export tooling, and a Go SDK.

> [!IMPORTANT]
> Cypra is not ready for external production testers yet. The codebase is locally runnable and heavily tested, but the active readiness plan is still closing security, deployment, release, and UX gates before the first VPS tester rollout. Track that work in [`PLAN.md`](./PLAN.md) and [`TASKS.md`](./TASKS.md).

## Why Cypra Exists

Cypra is for teams that want to own their auth stack without inheriting a cluster-sized control plane.

- **Self-hosted by default:** run Cypra on a small VPS with Docker Compose and Postgres.
- **Tenant-aware from the core:** each tenant has its own issuer, users, clients, WebAuthn RP ID, signing keys, sessions, and branding surface.
- **Operator-friendly:** setup wizard, dashboard, recovery commands, health/readiness checks, metrics, audit export, and backup/import flows are part of the product.
- **OIDC-compatible:** downstream apps use normal OIDC clients; examples include Next.js/Auth.js and a Go server.
- **Security-oriented:** tenant isolation uses host resolution, a tenant-scoped DB boundary, and Postgres RLS; secrets use envelope encryption; sessions and refresh tokens have reuse detection.

## Current Status

Cypra is in an external-tester readiness push.

What is already strong locally:

- `./bin/agent-ci run --quiet --all` passes.
- Coverage floors pass for security-critical packages.
- Managed e2e, accessibility, performance, image-size, and cold-start gates exist.
- The Dockerfile is compact and production-oriented: multi-stage build, distroless non-root runtime, migrations included.
- The dashboard and hosted login have a coherent design system documented in [`DESIGN.md`](./DESIGN.md).

What still blocks the first external tester wave:

- Security hardening around factor management, consent, proxy trust, cookies, PATs, and master-key validation.
- Production deployment hardening for the VPS Compose/Caddy path.
- Release automation, GHCR publishing, GitHub Release artifacts, and deployed smoke evidence.
- Dashboard and hosted-login polish to remove no-op actions, placeholder copy, and prototype-grade interactions.

## Architecture At A Glance

```text
Browser / app
  -> hosted login / dashboard / OIDC / API
  -> single Cypra Go server
  -> Postgres
  -> local-disk or S3-compatible object storage
```

The same server process exposes:

- Hosted login routes (`/login`, `/signup`, `/reset`, `/2fa`, `/oidc/consent`)
- Per-tenant OIDC discovery, JWKS, authorize, token, userinfo, and revoke endpoints
- Dashboard and admin REST APIs
- Setup and recovery flows
- `/healthz`, `/readyz`, `/metrics`, and storage proxy routes

Tenant isolation is enforced in layers: host-based tenant resolution, explicit tenant-scoped database APIs, Postgres RLS policies, per-tenant issuer URLs, and per-tenant signing-key namespaces.

## Features

| Area           | Included                                                                                                    |
| -------------- | ----------------------------------------------------------------------------------------------------------- |
| Authentication | Password, magic link, passkeys/WebAuthn, TOTP, WebAuthn 2FA, backup codes, Google upstream OAuth            |
| OIDC           | Per-tenant issuer, discovery, JWKS, auth code + PKCE, refresh-token rotation, userinfo, revoke, consent     |
| Admin          | Setup token, instance admins, tenant/project/user management, PATs, signing keys, provider configuration    |
| Operations     | Health/readiness, metrics, OTEL hooks, audit export, GDPR export/delete, backup/export/import, recovery CLI |
| UI             | Embedded dashboard, server-rendered hosted login, tenant accent/logo/display-name branding                  |
| SDKs/examples  | Go admin/OIDC SDK, Next.js/Auth.js example, Go server example                                               |

Known non-goals for the current readiness push: SAML, SMS, embeddable widgets, TypeScript SDK, tenant CNAME automation, Kubernetes/Helm, multi-region SaaS deployment, and compliance certification claims.

## Prerequisites

- Go via the pinned toolchain in [`go.mod`](./go.mod)
- Bun `1.3.11`
- Docker Engine with Compose v2
- `golangci-lint`, `gofumpt`, and `goimports`
- For full local HTTPS/wildcard development: `portless`
- For live-reloading local dev: `air`

## Quick Start: Run The Checks

```sh
git clone https://github.com/watzon/cypra.git
cd cypra
bun install --frozen-lockfile
go mod download
./bin/agent-ci run --quiet --all
```

Equivalent project target:

```sh
make ci
```

The CI pipeline installs dependencies, builds the dashboard and Go server, runs lint/format/typecheck, Go tests, dashboard tests, coverage floors, p99 performance checks, compressed image-size checks, and cold-start checks.

## Local Development

Start the full local developer stack:

```sh
make dev
```

`make dev` creates `.env` from `.env.example` if needed, starts local Postgres, starts `portless` wildcard routing, registers `https://cypra.localhost`, runs migrations, starts Vite for the dashboard, and runs the Go server through `air`.

Useful local routes:

- `https://cypra.localhost`
- `https://<tenant>.cypra.localhost`

Manage only the local dependency stack:

```sh
make dev-up
make dev-down
```

Reset local development data:

```sh
make dev-reset
```

> [!WARNING]
> `make dev-reset` deletes the local Cypra Postgres volume. It is intended only for local development.

## First Run Path

The local canonical demo walks from a fresh install to a downstream Next.js sign-in:

1. Start Cypra with `make dev`.
2. Open the setup URL printed in the server logs.
3. Redeem the setup token, enroll a passkey, and save backup codes.
4. Create tenant `acme` and project `console`.
5. Configure local providers.
6. Start `examples/nextjs` with the project issuer/client values.
7. Sign in through Cypra from the Next.js app.

Full walkthrough: [`docs/firstrun.md`](./docs/firstrun.md).

Run the managed browser smoke:

```sh
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/examples-smoke.spec.ts tests/e2e/canonical-demo/canonical-demo.spec.ts
```

## Common Commands

| Command                            | Purpose                                                                |
| ---------------------------------- | ---------------------------------------------------------------------- |
| `make install`                     | Install Bun and Go dependencies                                        |
| `make build`                       | Build dashboard assets and the `bin/cypra` server binary               |
| `make test`                        | Run Go tests, coverage floors, and dashboard tests                     |
| `make lint`                        | Run Go, dashboard, no-op UI, audit-write, and escape-hatch lint checks |
| `make typecheck`                   | Run Go vet and TypeScript checks                                       |
| `make ci`                          | Run the full local CI pipeline                                         |
| `./bin/agent-ci run --quiet --all` | Agent-friendly wrapper around the same CI pipeline                     |

## Deployment Notes

Deployment docs exist, but they are still being hardened before external testers should rely on them.

- VPS reference path: [`docs/deploy/vps.md`](./docs/deploy/vps.md)
- Compose and Caddy reference: [`deploy/README.md`](./deploy/README.md)
- Observability: [`docs/deploy/observability.md`](./docs/deploy/observability.md)
- Railway template notes: [`docs/deploy/railway-template.md`](./docs/deploy/railway-template.md)

The first external tester release requires a versioned GHCR image, verified release artifacts, a fresh VPS trial, deployed smoke evidence, metrics/tracing evidence, and owner approval. See [`TASKS.md`](./TASKS.md) for the active gate list.

## Repository Map

| Path             | Purpose                                                                               |
| ---------------- | ------------------------------------------------------------------------------------- |
| `cmd/cypra/`     | CLI entrypoint and backup/import commands                                             |
| `internal/`      | Go server, auth, OIDC, storage, DB, sessions, observability, and HTTP handlers        |
| `dashboard/`     | Embedded React dashboard                                                              |
| `db/migrations/` | SQL migrations                                                                        |
| `deploy/`        | Compose, Caddy, and alerting references                                               |
| `docs/`          | Architecture, security, deployment, playbooks, tutorials, ADRs, and readiness notes   |
| `examples/`      | Downstream app examples                                                               |
| `sdk/go/`        | Go SDK packages                                                                       |
| `tests/e2e/`     | Playwright e2e, accessibility, canonical demo, backup/import, and example smoke tests |

## Documentation

- Product/readiness plan: [`PLAN.md`](./PLAN.md)
- Design system and UI behavior: [`DESIGN.md`](./DESIGN.md)
- Execution tracker: [`TASKS.md`](./TASKS.md)
- Architecture overview: [`docs/architecture/overview.md`](./docs/architecture/overview.md)
- Threat model: [`docs/security/threat-model.md`](./docs/security/threat-model.md)
- Testing guide: [`docs/contributing/testing.md`](./docs/contributing/testing.md)
- Extension guide: [`docs/contributing/extending.md`](./docs/contributing/extending.md)

The original broader product plan and historical implementation log are archived under [`docs/archive/`](./docs/archive/).
