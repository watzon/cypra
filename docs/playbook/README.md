# Cypra Operator Playbook

The operator is the GDPR controller. Cypra ships enabling features; it is not a processor by itself. Use this playbook with the VPS guide in [`../deploy/vps.md`](../deploy/vps.md).

## First 24 Hours

- Configure email and upstream providers.
- Confirm `/healthz`, `/readyz`, `/api/v1/version`, and `/metrics` are reachable from monitoring.
- Load `deploy/alerts.yml` into Prometheus or translate the rules into your monitoring platform.
- Create a second instance admin and verify the last-admin guard.
- Back up Postgres and the storage backend.
- Run a Cypra logical export and restore it into a fresh restore stack.
- Run the deployed smoke checklist from `docs/deploy/vps.md`.
- Record unclear steps as follow-up tasks before inviting another tester.

## Backup And Restore

Use both Cypra logical backups and infrastructure backups.

- Run daily `cypra export` with `--passphrase-file` or `--passphrase-from-stdin`.
- Keep the backup passphrase outside the VPS.
- Snapshot the Docker volumes if using bundled Postgres or local-disk storage.
- Enable bucket versioning or provider backups if using S3-compatible storage.
- Before every upgrade, run an on-demand export and verify a restore into a fresh stack.

Restore expectations:

- `cypra import` restores only into an empty database.
- `cypra import` refuses DSR resurrection unless `--allow-resurrect` is explicitly passed and audited.
- A restore drill is incomplete until `/readyz`, `/api/v1/version`, and `cypra admin list-instance-admins --json` work on the restored stack.

## Upgrade And Rollback

Before an upgrade:

- Confirm `/readyz` is green.
- Record the current `CYPRA_IMAGE` tag.
- Run a fresh `cypra export`.
- Confirm the release notes do not require operator action before migrations.

After changing `CYPRA_IMAGE`, pull and restart only Cypra, then run readiness and deployed smoke checks.

Rollback is image-based unless a release explicitly documents irreversible migrations. If rollback requires data reversal, restore the pre-upgrade backup into a fresh stack instead of editing production data by hand.

## Data Inventory

- Users: email, metadata, profile picture object references, enrolled auth methods.
- Audit entries: actor, action, resource, before/after JSON, IP and user-agent when available.
- OIDC data: clients, consents, authorization codes, refresh tokens, signing keys.
- Operational data: sessions, pending invitations, email outbox, rate-limit buckets, GDPR deletion ledger.
- Storage objects: profile pictures and other object references stored through local disk or S3-compatible storage.

## DSR Runbook

- Export with `GET /api/v1/users/{id}/export` and provide the NDJSON to the requester.
- Delete with `DELETE /api/v1/users/{id}`.
- Verify the `gdpr_deletions` ledger row is `done`.
- Verify user PII is scrubbed and prior audit entries contain the GDPR redaction sentinel.
- Verify profile-picture storage objects are marked deleted.
- If restoring an old backup would resurrect a deleted user, do not pass `--allow-resurrect` unless the operator has a documented legal basis and incident note.

## Tenant Deletion Runbook

- Use the tenant Danger tab and typed-slug confirmation.
- New sign-ins stop immediately when suspended or deletion is scheduled.
- Sessions and refresh tokens are revoked during cascade.
- OIDC clients are soft-deleted.
- Signing keys enter `sunsetting` and JWKS remains available for 30 days.

## Breach Notification Runbook

- Preserve audit entries; do not delete rows.
- Export relevant audit ranges as NDJSON.
- Rotate affected client secrets, signing keys, provider keys, and the master key as appropriate.
- Notify downstream OIDC clients to refetch JWKS on `kid` miss.
- Save the incident timeline, impacted tenants, notifications, and recovery evidence outside the Cypra database.

## Sub-Processor Template

| Provider       | Purpose          | Data categories                 | Region          | DPA link |
| -------------- | ---------------- | ------------------------------- | --------------- | -------- |
| Resend         | Email delivery   | Email address, template payload | Operator-chosen | TBD      |
| Object storage | Profile pictures | Binary objects, metadata        | Operator-chosen | TBD      |

## Monitoring Checklist

- HTTP errors and latency.
- Auth failures and refresh-token reuse.
- Signing-key age.
- Pending migrations and master-key rotation phase.
- Storage write failures.
- Email outbox pending count.
- DB pool saturation.
- Bot-mitigation outcomes and rate-limit spikes.
