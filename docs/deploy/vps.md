# VPS Deployment

This is the reference single-host deployment shape for the first external tester release. It assumes Docker Engine with Compose v2 and Caddy terminating public TLS on the same host.

## Host Requirements

- 2 vCPU / 4 GiB RAM or larger.
- Docker Engine with Compose v2.
- DNS for one install domain, for example `auth.example.com`.
- Wildcard DNS for tenant subdomains, for example `*.auth.example.com`.
- Public firewall ingress only on TCP `80` and `443` to Caddy.

Do not expose Cypra or Postgres directly. In the production compose file they stay on the internal compose network.

## DNS And TLS

Create DNS records before starting the TLS profile:

- `A` / `AAAA` for `auth.example.com` pointing at the VPS.
- `A` / `AAAA` wildcard for `*.auth.example.com` pointing at the VPS.

`deploy/Caddyfile.example` is the production template. It serves `CYPRA_INSTALL_DOMAIN` and `*.CYPRA_INSTALL_DOMAIN`, terminates public TLS, and forwards traffic to `cypra:8080` inside the compose network.

Wildcard certificates require ACME DNS-01 with a Caddy build that includes your DNS provider module. If your Caddy image does not include a DNS provider module, set `CADDY_IMAGE` to a Caddy build that does and add the provider-specific `tls { dns ... }` block to `deploy/Caddyfile.example` before external testers use tenant-hosted login.

## Environment

Copy the production template and replace every `CHANGE_ME` value:

```sh
cp .env.production.example .env.production
```

Set at least:

- `CYPRA_IMAGE=ghcr.io/watzon/cypra:<version>` using a concrete release or prerelease tag.
- `CYPRA_INSTALL_DOMAIN=auth.example.com`.
- `CADDY_IMAGE=caddy:2-alpine`, or a Caddy image with your DNS provider module when using wildcard ACME DNS-01.
- `PUBLIC_BASE_URL=https://auth.example.com`.
- `CADDY_ACME_EMAIL=ops@example.com`.
- `POSTGRES_PASSWORD` and matching `DATABASE_URL` / `MIGRATE_DATABASE_URL` values.
- `MASTER_KEY` or `MASTER_KEY_FILE` with a generated 32-byte key.
- `TRUSTED_PROXY_HEADERS=x-forwarded` for the Caddy path.

Use `.env.example` only for local development. It contains development defaults and is not production-safe.

## Start Production TLS Profile

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml --profile with-tls up -d
```

Expected public listeners:

- `0.0.0.0:80` -> Caddy ACME HTTP challenge and redirects.
- `0.0.0.0:443` -> Caddy HTTPS.

Expected private-only services:

- `cypra:8080` inside the compose network only.
- `postgres:5432` inside the compose network only.

## Health And Readiness

Cypra's container healthcheck runs `cypra healthcheck`, which calls `http://127.0.0.1:8080/readyz` from inside the container. `/readyz` reflects database connectivity, migration state, storage readiness, and master-key readiness.

Useful checks:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml ps
curl -fsS https://auth.example.com/healthz
curl -fsS https://auth.example.com/readyz
```

If `/readyz` fails, check Cypra logs first:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml logs cypra
```

## First Boot

1. Start the TLS profile.
2. Read the setup token from Cypra logs.
3. Visit `https://auth.example.com/setup/<token>`.
4. Enroll the first instance admin passkey and save backup codes.
5. Create the first tenant and configure email/upstream providers.

## Local Development Is Separate

For local development, use `make dev` or layer `deploy/docker-compose.dev.yml` explicitly. The dev override publishes host ports for Postgres and Cypra and uses `deploy/Caddyfile.local` with `tls internal` for `.localhost` only. Do not use the dev override on a public VPS.
