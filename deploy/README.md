# Deploy

This directory contains the reference single-host deployment files.

For local development, prefer the repository root `make dev` target. It uses this compose file for dependencies, loads values from `.env`, and registers the local HTTPS route with portless.

## Default Profile

Runs Cypra and Postgres:

```sh
docker compose -f deploy/docker-compose.yml up -d postgres cypra
```

Build the local image first when testing unpublished changes:

```sh
docker build -t ghcr.io/watzon/cypra:dev .
```

Set at least:

- `PUBLIC_BASE_URL=https://auth.example.com`
- `MASTER_KEY=<32-byte key as base64, hex, or raw 32-byte value>`
- `DATABASE_URL` and `MIGRATE_DATABASE_URL` if not using the bundled Postgres service

## with-tls Profile

Runs Cypra, Postgres, and Caddy:

```sh
docker compose -f deploy/docker-compose.yml --profile with-tls up -d
```

Edit `deploy/Caddyfile.example` for the real install domain and wildcard tenant domain before using it outside local testing. Caddy terminates TLS and forwards requests to Cypra on the internal compose network.

## Storage

Local disk storage is suitable for local testing and small single-host installs. Use S3-compatible storage for PaaS or hosts where local files are ephemeral.
