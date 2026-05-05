# Cypra Operator Playbook

The operator is the GDPR controller. Cypra ships enabling features; it is not a processor by itself.

## First 24 Hours

- Configure email and upstream providers.
- Confirm `/readyz` and `/metrics` are reachable from monitoring.
- Load `deploy/alerts.yml` into Prometheus.
- Create a second instance admin and verify the last-admin guard.
- Back up Postgres and the storage backend.

## Data Inventory

- Users: email, metadata, profile picture object references, enrolled auth methods.
- Audit entries: actor, action, resource, before/after JSON, IP and user-agent when available.
- OIDC data: clients, consents, authorization codes, refresh tokens, signing keys.
- Operational data: sessions, pending invitations, email outbox, rate-limit buckets, GDPR deletion ledger.

## DSR Runbook

- Export with `GET /api/v1/users/{id}/export` and provide the NDJSON to the requester.
- Delete with `DELETE /api/v1/users/{id}`.
- Verify the `gdpr_deletions` ledger row is `done`.
- Verify user PII is scrubbed and prior audit entries contain the GDPR redaction sentinel.
- Verify profile-picture storage objects are marked deleted.

## Tenant Deletion Runbook

- Use the tenant Danger tab and typed-slug confirmation.
- New sign-ins stop immediately when suspended or deletion is scheduled.
- Sessions and refresh tokens are revoked during cascade.
- OIDC clients are soft-deleted.
- Signing keys enter `sunsetting` and JWKS remains available for 30 days.

## Breach Notification Runbook

- Preserve audit entries; do not delete rows.
- Export relevant audit ranges as NDJSON.
- Rotate affected client secrets, signing keys, and the master key as appropriate.
- Notify downstream OIDC clients to refetch JWKS on `kid` miss.

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
