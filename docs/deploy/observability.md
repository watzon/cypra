# Observability

Use this with the first-24-hours checklist in [`vps.md`](./vps.md) and the operator playbook in [`../playbook/README.md`](../playbook/README.md).

## Metrics

Cypra exposes Prometheus metrics at `/metrics`.

```sh
curl -fsS https://auth.example.com/metrics | head
```

Reference alert rules live in `deploy/alerts.yml` and cover HTTP 5xx rate, auth failures, refresh-token reuse detection, signing-key age, storage failures, DB pool saturation, migrations pending, master-key rotation state, and email outbox backlog.

Minimum first-tester dashboard:

- `cypra_http_request_duration_seconds`
- `cypra_auth_attempts_total`
- `cypra_oidc_refresh_reuse_detected_total`
- `cypra_signing_key_age_seconds`
- `cypra_storage_operation_duration_seconds`
- `cypra_email_outbox_pending`
- `cypra_db_pool_*`
- `cypra_rate_limit_exceeded_total`
- `cypra_migrations_pending`
- `cypra_master_key_rotation_phase`
- `cypra_botmitigation_check_total`

## Logs

Cypra writes structured `slog` JSON to stdout. Operators should ship stdout to their logging platform and redact fields tagged with `redacted-on-export`, including setup tokens and one-time recovery links.

Useful commands:

```sh
docker compose --env-file .env.production -f deploy/docker-compose.yml logs --tail=200 cypra
docker compose --env-file .env.production -f deploy/docker-compose.yml logs -f cypra caddy
```

Retain Cypra, Caddy, and Postgres logs outside the container lifecycle before inviting external testers.

## Tracing

OpenTelemetry is enabled when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

The expected path to inspect during deployment validation is `/oidc/authorize` through hosted-login/email dispatch/audit emission and back to OIDC token issuance.

Minimal collector smoke:

1. Set `OTEL_EXPORTER_OTLP_ENDPOINT` in `.env.production`.
2. Restart Cypra with `docker compose --env-file .env.production -f deploy/docker-compose.yml up -d cypra`.
3. Complete a hosted-login sign-in through a tenant issuer.
4. Confirm the collector shows spans for authorize, login/consent, token, email dispatch if used, and audit emission.

## Error Tracking

Cypra does not bundle Sentry, Datadog, or another error tracker. Ship stdout logs to the operator's platform and create alert rules on error-level records and request IDs. If using Sentry or Datadog, collect Cypra logs from Docker stdout rather than adding SDKs to the binary.
