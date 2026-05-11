# Deploy

This directory contains the reference single-host deployment files.

For local development, prefer the repository root `make dev` target. It layers `docker-compose.dev.yml` on top of the production-safe compose file for host port mappings, loads values from `.env`, and registers the local HTTPS route with portless.

## Production Profile

Runs Cypra and Postgres on the internal compose network. It does not publish Cypra or Postgres to the host:

```sh
CYPRA_IMAGE=ghcr.io/watzon/cypra:0.1.0-rc.2 \
  docker compose --env-file .env.production -f deploy/docker-compose.yml up -d postgres cypra
```

Use a release tag for `CYPRA_IMAGE`. To test unpublished changes locally, build a local image and set `CYPRA_IMAGE` explicitly rather than editing the production compose file:

```sh
docker build -t cypra:local .
CYPRA_IMAGE=cypra:local docker compose --env-file .env -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up -d cypra
```

Set at least:

- `PUBLIC_BASE_URL=https://auth.example.com`
- `MASTER_KEY=<32-byte key as base64, hex, or raw 32-byte value>`
- `DATABASE_URL` and `MIGRATE_DATABASE_URL` if not using the bundled Postgres service
- `TRUSTED_PROXY_HEADERS=x-forwarded` when Cypra sits behind the production Caddy service

## with-tls Profile

Runs Cypra, Postgres, and Caddy. In this profile Caddy is the only public entrypoint, and it publishes ports `80` and `443`:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml --profile with-tls up -d
```

Copy `.env.production.example` to `.env.production`, set `CYPRA_INSTALL_DOMAIN`, `CADDY_ACME_EMAIL`, `PUBLIC_BASE_URL`, `TRUSTED_PROXY_HEADERS=x-forwarded`, and DNS records before using this outside local testing. Create both the install host record, for example `auth.example.com`, and wildcard tenant DNS, for example `*.auth.example.com`.

For wildcard tenant certificates, use `CADDY_IMAGE` with a DNS-provider-enabled Caddy build and add that provider's DNS-01 `tls` block to `Caddyfile.example`. Caddy terminates TLS, forwards `Host` plus `X-Forwarded-*` headers to Cypra on the internal compose network, and should be the only service receiving public traffic. Do not add client-controlled trust headers or publish Cypra directly to make proxy behavior work.

Never expose these services directly to the internet:

- `postgres:5432`
- `cypra:8080`

Only Caddy should receive public traffic in the production TLS profile.

## Local Compose Override

`docker-compose.dev.yml` is for local development and CI harnesses only. It publishes Postgres to `${POSTGRES_HOST_PORT:-54320}` and Cypra to `${CYPRA_HOST_PORT:-8080}` so host-run tools can connect. Do not use it for a public VPS.

## Storage

Local disk storage is suitable for local testing and small single-host installs when the `cypra-storage` volume is backed up. Use S3-compatible storage for PaaS, hosts where local files are ephemeral, or operators who already rely on bucket versioning and lifecycle policies.

The full day-one deployment, backup, restore, upgrade, rollback, monitoring, and deployed smoke checklist lives in [`../docs/deploy/vps.md`](../docs/deploy/vps.md).
