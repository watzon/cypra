# Bot Mitigation

Cypra v1 ships a `Verifier` seam and Prometheus counters. The default verifier is `noop` and records `cypra_botmitigation_check_total{verifier,outcome}`.

For v1.1, replace the verifier with a Turnstile or hCaptcha implementation at server construction, keep the same `Verify(ctx, token)` contract, and preserve the metric labels so existing alerts continue to work.

Operational checklist:

- Watch `cypra_rate_limit_exceeded_total` and `cypra_auth_attempts_total{outcome="fail"}` for stuffing spikes.
- Enable a reverse-proxy blocklist for active attacks.
- Swap in a challenge verifier when legitimate users behind shared NATs see collateral rate limits.
