# Breach Notification

Use audit NDJSON exports to determine impact. Preserve append-only audit rows, rotate affected secrets, notify affected tenants and downstream OIDC clients, and document all actions in the operator incident log.

## Immediate Containment

1. Preserve Cypra, Caddy, Postgres, and platform logs.
2. Export relevant audit ranges as NDJSON.
3. Capture `/api/v1/version`, deployed image tag, and current configuration values excluding secrets.
4. Disable or rotate affected upstream provider credentials, OIDC client secrets, signing keys, and email API keys.
5. If instance-admin access is affected, use `cypra admin invite <email>` from the host to establish a clean recovery admin.

## Impact Review

- Review audit entries by actor, tenant, resource, IP, and user-agent.
- Check refresh-token reuse metrics and auth-failure spikes.
- Confirm whether storage objects or backup files were accessible.
- Confirm whether any setup token or recovery invite appeared in logs that left the host.

## Notification Notes

- The operator owns notification timing and content.
- Notify downstream OIDC clients to refetch JWKS on `kid` miss after signing-key rotation.
- Record tenant notifications, regulator notifications if required, and the final remediation summary outside the Cypra database.
