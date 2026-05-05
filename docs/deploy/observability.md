# Observability

## Metrics

Cypra exposes Prometheus metrics at `/metrics`. Reference alert rules live in `deploy/alerts.yml` and cover HTTP 5xx rate, auth failures, refresh-token reuse detection, signing-key age, storage failures, DB pool saturation, migrations pending, master-key rotation state, and email outbox backlog.

## Logs

Cypra writes structured `slog` JSON to stdout. Operators should ship stdout to their logging platform and redact fields tagged with `redacted-on-export`, including setup tokens and one-time recovery links.

## Tracing

OpenTelemetry is enabled when `OTEL_EXPORTER_OTLP_ENDPOINT` is set. The expected path to inspect during deployment validation is `/oidc/authorize` through hosted-login/email dispatch/audit emission and back to OIDC token issuance.

## Error Tracking

Cypra does not bundle Sentry, Datadog, or another error tracker. Ship stdout logs to the operator's platform and create alert rules on error-level records and request IDs. If using Sentry or Datadog, collect Cypra logs from Docker stdout rather than adding SDKs to the binary.
