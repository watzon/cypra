# What To Monitor

Load `deploy/alerts.yml` into Prometheus or translate the same conditions into your monitoring system.

## Required First-Tester Checks

- `curl -fsS https://auth.example.com/healthz`
- `curl -fsS https://auth.example.com/readyz`
- `curl -fsS https://auth.example.com/metrics | head`
- `docker compose --env-file .env.production -f deploy/docker-compose.yml logs --tail=200 cypra`

## Key Metrics

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

## First 24 Hours

- Watch HTTP 5xx and latency after setup, tenant creation, and Next.js sign-in.
- Confirm email outbox drains after provider configuration.
- Confirm migrations pending is zero after each restart.
- Confirm storage failures stay at zero after profile-picture or object flows.
- Confirm auth failures and rate-limit spikes are explainable.
- Confirm logs are retained outside Docker before relying on them for incidents.
