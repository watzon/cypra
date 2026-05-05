# Extending Cypra

## Add An Auth Method

Auth methods live under `internal/auth`. Keep the verifier small and tenant-aware, then wire it through hosted-login handlers and dashboard state surfaces.

Required steps:

- Store tenant-owned credentials in tables with direct `tenant_id` columns and RLS policies.
- Route all secret operations through `internal/crypto`.
- Emit audit events for enroll, disable, reset, and successful/failed verification as applicable.
- Add rate-limit and bot-mitigation hooks to every interactive verification endpoint.
- Add unit tests plus integration tests against real Postgres via `internal/dbtest` when persistence is involved.

## Add An Email Backend

Email backends implement the sender abstraction used by the in-process email worker. The worker owns retry, outbox locking, and exponential backoff; backends should only translate a message into provider API calls.

Required steps:

- Add provider config persistence with envelope-encrypted secrets.
- Add a health check result for Instance Diagnostics.
- Avoid blocking auth handlers; handlers must enqueue `email_outbox` rows.
- Add metrics for success/failure and preserve request/audit correlation.

## Add A Storage Backend

Storage backends implement object put/delete and signed URL generation. Storage is operator-controlled, not tenant-controlled.

Required steps:

- Keep storage config in bootstrap/operator configuration or instance-admin surfaces only.
- Encrypt provider credentials at rest.
- Return bounded-TTL signed URLs.
- Ensure user-delete purges profile-picture objects and tenant-delete cascades stored objects.
- Add read-only diagnostics so operators can identify the active backend without exposing secrets.
