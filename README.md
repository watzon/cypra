# Cypra

[![Build](https://github.com/watzon/cypra/actions/workflows/build.yml/badge.svg)](https://github.com/watzon/cypra/actions/workflows/build.yml)

Cypra is an open-source, self-hosted, multi-tenant authentication platform: a Go server with an embedded React dashboard, Postgres storage, hosted login, and a per-tenant OIDC issuer designed to be bootable by a small team in an afternoon.

The product and implementation source of truth lives in [`PLAN.md`](./PLAN.md), [`DESIGN.md`](./DESIGN.md), and [`BRAINSTORM.md`](./BRAINSTORM.md). Execution is tracked in [`TASKS.md`](./TASKS.md).

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

## Tests

- `make test` runs Go and dashboard tests.
- `make lint` runs Go and dashboard linters.
- `make typecheck` runs Go vet and TypeScript checks.
- `make ci-pipeline` is the exact pipeline used by `./bin/agent-ci` and GitHub Actions.
