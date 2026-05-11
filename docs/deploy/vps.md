# VPS Deployment

This is the reference single-host deployment shape for the first external tester release. It assumes Docker Engine with Compose v2 and Caddy terminating public TLS on the same host.

The goal is an install that a technical tester can operate without reading source code: Caddy is the only public entrypoint, Cypra and Postgres stay private on the compose network, backups are repeatable, and rollback has a known path.

## Host Requirements

- 2 vCPU / 4 GiB RAM or larger.
- Docker Engine with Compose v2.
- DNS control for one install domain, for example `auth.example.com`.
- Wildcard DNS control for tenant subdomains, for example `*.auth.example.com`.
- Public firewall ingress only on TCP `80` and `443` to Caddy.
- Persistent disk for Docker volumes, or S3-compatible object storage for uploaded objects.

Do not expose Cypra or Postgres directly. In the production compose file they stay on the internal compose network.

## 1. Choose Domains And DNS

Pick one install host. The examples below use `auth.example.com`.

Create DNS records before starting the TLS profile:

- `A` / `AAAA` for `auth.example.com` pointing at the VPS.
- `A` / `AAAA` wildcard for `*.auth.example.com` pointing at the VPS.

Verify propagation from your workstation:

```sh
dig +short auth.example.com
dig +short foo.auth.example.com
```

Both commands should return the VPS address. Tenant issuers and WebAuthn RP IDs depend on wildcard tenant hosts, so do not skip the wildcard record.

## 2. Install Runtime Prerequisites

On a fresh Ubuntu-style VPS:

```sh
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg openssl
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker "$USER"
```

Log out and back in so the `docker` group is active, then verify Compose:

```sh
docker version
docker compose version
```

## 3. Put Deployment Files On The Host

Create a deployment directory and copy these files from the release checkout or tarball:

```text
deploy/docker-compose.yml
deploy/Caddyfile.example
.env.production.example
```

Keep the files together so the compose file can mount `./Caddyfile.example` into Caddy.

## 4. Configure Production Environment

Copy the production template and replace every `CHANGE_ME` value:

```sh
cp .env.production.example .env.production
chmod 600 .env.production
```

Set at least:

- `CYPRA_IMAGE=ghcr.io/watzon/cypra:<version>` using a concrete release or prerelease tag.
- `CYPRA_INSTALL_DOMAIN=auth.example.com`.
- `PUBLIC_BASE_URL=https://auth.example.com`.
- `CADDY_ACME_EMAIL=ops@example.com`.
- `POSTGRES_PASSWORD` with a long random value.
- `DATABASE_URL=postgres://cypra:<same-password>@postgres:5432/cypra?sslmode=disable`.
- `MIGRATE_DATABASE_URL=postgres://cypra:<same-password>@postgres:5432/cypra?sslmode=disable`.
- `MASTER_KEY` or `MASTER_KEY_FILE` with one generated 32-byte key.
- `TRUSTED_PROXY_HEADERS=x-forwarded` for the production Caddy path.
- `LOG_LEVEL=info` unless you are actively debugging.

Generate safe secrets on the VPS:

```sh
openssl rand -base64 32
openssl rand -base64 48
```

Use the 32-byte value for `MASTER_KEY`. Use a separate long value for `POSTGRES_PASSWORD`. Store both in a password manager or secret store before starting the service. Losing the master key can make encrypted columns unrecoverable; a Cypra backup can be restored only when you also have its passphrase and a valid destination `MASTER_KEY`.

`MASTER_KEY_FILE` is preferred when your host has a secret-file workflow. If you use it, mount the file into the Cypra container and leave `MASTER_KEY` empty. Do not commit `.env.production` or secret files.

Rotation expectations:

- Rotate `POSTGRES_PASSWORD` during a planned maintenance window and update `DATABASE_URL`, `MIGRATE_DATABASE_URL`, and the Postgres role password together.
- Rotate upstream OAuth client secrets and email provider API keys in the provider console first, then update Cypra immediately.
- Rotate OIDC signing keys from the dashboard or CLI path that preserves overlap; do not delete active keys manually.
- Rotate `MASTER_KEY` only through Cypra's master-key rotation workflow. Never replace `MASTER_KEY` directly in `.env.production` while encrypted rows still depend on the old key.
- Rotate backup passphrases by creating a new export with the new passphrase; old backup files remain protected by the old passphrase.

## 5. Configure TLS And Reverse Proxy

`deploy/Caddyfile.example` is the production template. It serves `CYPRA_INSTALL_DOMAIN` and `*.CYPRA_INSTALL_DOMAIN`, terminates public TLS, and forwards traffic to `cypra:8080` inside the compose network.

The reverse proxy forwards these headers to Cypra:

- `Host`
- `X-Forwarded-Host`
- `X-Forwarded-Proto`
- `X-Forwarded-For`

Cypra trusts forwarded headers only when `TRUSTED_PROXY_HEADERS=x-forwarded` is configured server-side. Do not add any client-controlled trust header. Do not run Cypra directly on a public port to work around proxy issues.

Wildcard certificates require ACME DNS-01 with a Caddy build that includes your DNS provider module. If your Caddy image does not include a DNS provider module, set `CADDY_IMAGE` to a Caddy build that does and add the provider-specific `tls { dns ... }` block to `deploy/Caddyfile.example` before external testers use tenant-hosted login.

If you only have HTTP-01 certificates for the install host, tenant subdomains will not be production-ready. Fix DNS-01 before inviting testers.

## 6. Start The Production TLS Profile

Start Cypra, Postgres, and Caddy:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml --profile with-tls up -d
```

Expected public listeners:

- `0.0.0.0:80` -> Caddy ACME HTTP challenge and redirects.
- `0.0.0.0:443` -> Caddy HTTPS.

Expected private-only services:

- `cypra:8080` inside the compose network only.
- `postgres:5432` inside the compose network only.

Confirm the shape:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml ps
docker compose --env-file .env.production -f deploy/docker-compose.yml port caddy 443
docker compose --env-file .env.production -f deploy/docker-compose.yml port cypra 8080 || true
docker compose --env-file .env.production -f deploy/docker-compose.yml port postgres 5432 || true
```

The Caddy command should show a host binding. The Cypra and Postgres commands should not show public host bindings.

## 7. Health, Readiness, Logs, And Metrics

Cypra's container healthcheck runs `cypra healthcheck`, which calls `http://127.0.0.1:8080/readyz` from inside the container. `/readyz` reflects database connectivity, migration state, storage readiness, and master-key readiness.

Run these checks after boot and after every upgrade:

```sh
curl -fsS https://auth.example.com/healthz
curl -fsS https://auth.example.com/readyz
curl -fsS https://auth.example.com/api/v1/version
curl -fsS https://auth.example.com/metrics | head
```

Inspect logs:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml logs --tail=200 cypra
docker compose --env-file .env.production -f deploy/docker-compose.yml logs --tail=200 caddy
docker compose --env-file .env.production -f deploy/docker-compose.yml logs --tail=200 postgres
```

Follow logs during a smoke test:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml logs -f cypra caddy
```

Cypra writes structured JSON logs to stdout. Treat setup tokens and recovery links as sensitive even though Cypra tags them with `redacted-on-export` for log shippers.

## 8. First Boot

1. Start the TLS profile.
2. Read the setup token from Cypra logs.
3. Visit `https://auth.example.com/setup/<token>`.
4. Enroll the first instance admin passkey and save backup codes.
5. Create a second instance admin before inviting testers.
6. Create the first tenant and project.
7. Configure email and upstream providers.
8. Run the deployed smoke checklist below.

If the setup token is lost before use and no instance admin exists, mint a fresh token from the host:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml run --rm cypra admin reset-bootstrap
```

If at least one instance admin already exists, use the recovery invite path instead of resetting bootstrap:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml run --rm cypra admin invite recovery-admin@example.com
```

## 9. Storage Choices

### Local Disk

`STORAGE_BACKEND=local-disk` stores objects in the `cypra-storage` Docker volume at `STORAGE_LOCAL_PATH`. This is acceptable for a small single-host tester install only if the host volume is included in backups.

Use local disk when:

- The VPS has persistent disk snapshots.
- You can back up Docker volumes regularly.
- You want the simplest first tester setup.

Do not use local disk when the host filesystem is ephemeral or you cannot prove volume backups.

### S3-Compatible Storage

Use `STORAGE_BACKEND=s3-compatible` and set the `STORAGE_S3_*` values when object durability should not depend on the VPS disk.

Use S3-compatible storage when:

- The host may be replaced or rebuilt often.
- You already have bucket backup/versioning policies.
- You need storage lifecycle controls outside Docker.

Back up S3 credentials and bucket policy with the rest of the deployment secrets. Cypra does not manage bucket versioning or cross-region replication for you.

## 10. Database Role Expectations

The bundled Postgres service is for the first single-host tester install. It creates one database named `cypra` and one application role named `cypra`. Cypra migrations and runtime access use the configured `DATABASE_URL` / `MIGRATE_DATABASE_URL` values.

For a managed external Postgres service:

- Create an empty database for Cypra.
- Point `DATABASE_URL` and `MIGRATE_DATABASE_URL` at that database.
- Keep TLS settings appropriate for the provider instead of `sslmode=disable`.
- Remove or ignore the bundled `postgres` service only after confirming Cypra can reach the managed database.

Do not publish Postgres to the internet. If you need emergency SQL access, use SSH to the host and run `docker compose exec postgres psql -U cypra -d cypra`, or use the managed provider's private access path.

## 11. Backup Cadence

Use both Cypra logical backups and infrastructure backups.

Recommended cadence for testers:

- Daily `cypra export` with a passphrase stored outside the VPS.
- Daily VPS or volume snapshot if using local disk storage.
- Before every upgrade, run an on-demand `cypra export` and verify the file is non-empty.
- Keep at least seven daily backups and one pre-upgrade backup until the next upgrade has run for 24 hours.

Create a backup passphrase file with mode `0600`:

```sh
openssl rand -base64 48 > backup.passphrase
chmod 600 backup.passphrase
```

Run an export:

```sh
mkdir -p backups
docker compose --env-file .env.production -f deploy/docker-compose.yml run --rm \
  -v "$PWD/backups:/backups" \
  -v "$PWD/backup.passphrase:/run/secrets/cypra-backup-passphrase:ro" \
  cypra export \
  --out "/backups/cypra-$(date -u +%Y%m%dT%H%M%SZ).json" \
  --passphrase-file /run/secrets/cypra-backup-passphrase
```

Copy backup files and the passphrase to separate secure storage. Do not store the only copy beside the VPS.

## 12. Restore Drill

Run a restore drill before inviting testers and after changing backup automation.

The safest drill is a second fresh stack with an empty database and a different compose project name:

```sh
cp .env.production .env.restore
# Edit .env.restore to use a separate PUBLIC_BASE_URL, POSTGRES_PASSWORD, and any test-only ports/DNS.
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml up -d postgres
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml run --rm cypra migrate
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml run --rm \
  -v "$PWD/backups:/backups:ro" \
  -v "$PWD/backup.passphrase:/run/secrets/cypra-backup-passphrase:ro" \
  cypra import --passphrase-file /run/secrets/cypra-backup-passphrase /backups/<backup-file>.json
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml up -d cypra
```

Verify:

```sh
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml run --rm cypra admin list-instance-admins --json
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml exec cypra /usr/local/bin/cypra healthcheck
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml run --rm cypra version --json
```

`cypra import` refuses to overwrite a non-empty Cypra database. It also refuses to resurrect DSR-deleted users unless you pass `--allow-resurrect`, which should require a documented operator decision.

Clean up the drill only after saving notes:

```sh
docker compose --env-file .env.restore -p cypra-restore -f deploy/docker-compose.yml down -v
```

## 13. Export / Import Expectations And Limits

`cypra export` writes a passphrase-protected Cypra backup envelope. It includes database content, storage object content for local-disk storage, and encrypted columns rewrapped under the backup passphrase.

`cypra import` restores into an empty Cypra database and rewraps secrets under the destination `MASTER_KEY`.

Important limits:

- The backup passphrase is mandatory and is not recoverable.
- Imports refuse non-empty databases.
- DSR-deleted users are protected by the deletion ledger and require explicit `--allow-resurrect` to restore.
- For S3-compatible storage, keep provider-side bucket backups/versioning; do not rely only on the database export.
- A backup is not a substitute for preserving `.env.production`, `MASTER_KEY` or `MASTER_KEY_FILE`, Caddy state, and DNS settings.

## 14. Upgrade Procedure

Before upgrading:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml ps
curl -fsS https://auth.example.com/readyz
docker compose --env-file .env.production -f deploy/docker-compose.yml run --rm cypra version --json
```

Create a pre-upgrade backup using the backup command above.

Edit `.env.production` and set `CYPRA_IMAGE` to the new concrete tag:

```sh
CYPRA_IMAGE=ghcr.io/watzon/cypra:<new-version>
```

Pull and restart:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml pull cypra
docker compose --env-file .env.production -f deploy/docker-compose.yml up -d cypra
```

Verify after upgrade:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml ps
curl -fsS https://auth.example.com/readyz
curl -fsS https://auth.example.com/api/v1/version
docker compose --env-file .env.production -f deploy/docker-compose.yml logs --tail=200 cypra
```

Run the deployed smoke checklist before considering the upgrade complete.

## 15. Rollback Procedure

Rollback is image-based unless a release explicitly documents irreversible migrations. Stop and review release notes before downgrading across a migration boundary.

If the new container is unhealthy and no destructive migration has been applied:

```sh
# Edit .env.production back to the previous CYPRA_IMAGE tag.
docker compose --env-file .env.production -f deploy/docker-compose.yml pull cypra
docker compose --env-file .env.production -f deploy/docker-compose.yml up -d cypra
curl -fsS https://auth.example.com/readyz
```

If data was changed and the old image cannot run safely, restore the pre-upgrade backup into a fresh stack instead of reusing the failed database. Do not run ad-hoc SQL deletes or schema changes as a rollback.

## 16. Certificate Renewal Checks

Caddy renews certificates automatically. Check renewal health weekly and after DNS changes:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml logs --since=24h caddy
openssl s_client -connect auth.example.com:443 -servername auth.example.com </dev/null 2>/dev/null | openssl x509 -noout -dates -subject
openssl s_client -connect foo.auth.example.com:443 -servername foo.auth.example.com </dev/null 2>/dev/null | openssl x509 -noout -dates -subject
```

Investigate any ACME error before inviting new testers. Wildcard renewal failures usually mean the DNS-01 provider module or credentials are missing from the configured Caddy image.

## 17. First 24 Hours Monitoring Checklist

During the first day of a tester install:

- Confirm `/healthz`, `/readyz`, and `/metrics` return successfully.
- Load `deploy/alerts.yml` into Prometheus or translate the same conditions into your monitoring platform.
- Watch HTTP 5xx rate, request latency, auth failures, refresh-token reuse, signing-key age, storage failures, DB pool saturation, migration pending state, master-key rotation state, and email outbox backlog.
- Confirm Cypra, Caddy, and Postgres logs are being retained outside the container lifecycle.
- Create a second instance admin and verify you can list admins from the CLI.
- Run one backup export and one restore drill.
- Confirm email provider diagnostics and upstream OAuth provider diagnostics from the dashboard.
- Record any unclear doc step as a follow-up task before inviting the next tester.

## 18. Deployed Smoke Checklist

Run this after first boot, after upgrades, and before external tester rollout:

1. Open `https://auth.example.com` and confirm the dashboard loads.
2. Redeem setup or sign in as an existing instance admin.
3. Create or verify tenant `acme`.
4. Create or verify project `console`.
5. Copy issuer, client ID, and client secret.
6. Configure the Next.js example with deployed values from `docs/firstrun.md`.
7. Sign in through `https://acme.auth.example.com` from the Next.js example.
8. Confirm the example receives Auth.js session claims.
9. Run `curl -fsS https://auth.example.com/readyz`.
10. Run `curl -fsS https://auth.example.com/metrics | head`.
11. Run a backup export.
12. Import that backup into a fresh restore stack.
13. Verify `cypra admin list-instance-admins --json` works on the restored stack.
14. Mint and redeem a recovery invite with `cypra admin invite <email>`.

If any step fails, keep logs and add a P0/P1 follow-up before tester rollout.

## 19. Known Beta Limits

Cypra v0.1 is for controlled external testers, not general availability. Current non-goals are intentional and should be communicated plainly:

- No SAML.
- No embeddable widget.
- No TypeScript SDK.
- No built-in tenant CNAME management.
- No SMS/Twilio.
- No bundled Turnstile/hCaptcha provider.
- The dashboard is desktop-optimized for technical operators.
- Railway one-click publication is deferred until after the first tester wave.

## 20. Local Development Is Separate

For local development, use `make dev` or layer `deploy/docker-compose.dev.yml` explicitly. The dev override publishes host ports for Postgres and Cypra and uses `deploy/Caddyfile.local` with `tls internal` for `.localhost` only. Do not use the dev override on a public VPS.
