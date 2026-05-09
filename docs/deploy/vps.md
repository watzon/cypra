# VPS Deployment

This is the reference single-host deployment shape for v0.1.

## Host

- 2 vCPU / 4 GiB RAM or larger.
- Docker Engine with Compose v2.
- DNS for the install domain and wildcard tenant subdomains.
- Ports 80 and 443 open to Caddy; Cypra itself should stay on the internal compose network.

## Compose Profiles

Default profile:

```sh
docker compose -f deploy/docker-compose.yml up -d postgres cypra
```

TLS profile:

```sh
docker compose -f deploy/docker-compose.yml --profile with-tls up -d
```

Set `PUBLIC_BASE_URL=https://auth.example.com` and point wildcard DNS such as `*.auth.example.com` at the VPS.

## Required Secrets

- `DATABASE_URL` and `MIGRATE_DATABASE_URL` for runtime/migration roles.
- `MASTER_KEY` or `MASTER_KEY_FILE`; use 32 random bytes encoded as base64, hex, or raw 32-byte value.
- Storage config: local disk for small installs, S3-compatible for durable object storage.
- Email provider credentials once terminal email is no longer acceptable.

## First Boot

1. Start the stack.
2. Read the setup token from Cypra logs.
3. Visit `/setup/<token>` on the install domain.
4. Enroll the first instance admin passkey and save backup codes.
5. Create the first tenant and configure email/upstream providers.
