# Contributing

## Prerequisites

- Go 1.26.1 via `.tool-versions` (`go.mod` supports Go 1.23+ with a pinned toolchain).
- Bun 1.3.11 via `.tool-versions` and `package.json` `packageManager`.
- Docker Desktop or another Docker Compose compatible runtime.
- `portless` for HTTPS local development on `https://cypra.localhost` and `https://*.cypra.localhost`.
- `golangci-lint`, `gofumpt`, and `goimports` on your `PATH`.

## Setup

```sh
bun install --frozen-lockfile
go mod download
```

## Local Development

Start Postgres and run the app stack:

```sh
make dev
```

`make dev` creates `.env` from `.env.example` if it does not exist, starts the compose-managed development dependencies, starts portless with wildcard routing, registers `https://cypra.localhost` to Cypra, runs migrations, starts the dashboard Vite server, and stops the dev compose services when you exit. Use `make dev-up` and `make dev-down` when you only need the compose dependencies.

The development URLs are:

- `https://cypra.localhost` for the instance dashboard.
- `https://acme.cypra.localhost` style tenant subdomains.

Install `portless` before using WebAuthn or tenant-subdomain flows locally. `make dev` starts the proxy and registers the Cypra route. The project relies on HTTPS and wildcard `.localhost` routing so dev and production cookie, RP ID, and issuer behavior stay aligned.

Agents configuring or troubleshooting this setup should load the `portless` skill and preserve the `https://cypra.localhost` plus `https://*.cypra.localhost` convention.

## CI

Run the full local pipeline before committing:

```sh
make ci
./bin/agent-ci run --quiet --all
```

`make ci-pipeline` is the shared source of truth for local CI, the project-local `agent-ci` wrapper, and GitHub Actions.

## Common Commands

- `make build` builds the dashboard first, then the Go binary.
- `make test` runs Go and dashboard tests.
- `make lint` runs Go and dashboard linters.
- `make typecheck` runs Go vet and TypeScript checks.
- `make image-size` builds the production Docker image, measures the gzip-compressed image archive, and fails over 80 MB.

## Commit Messages

Use concise imperative subjects. Bitforge phase completion commits use:

```text
Phase N: <title> -- complete
```

Do not commit `.env`, local databases, `node_modules`, dashboard build output, or agent-local state.
