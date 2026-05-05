# Cypra E2E Harness

Playwright harness introduced in Phase 11 and extended by the Phase 12 canonical demo.

The harness expects a local Cypra stack plus Postgres, MailHog or another SMTP stub, and a stub Google upstream. Phase 12 wires the full Docker Compose runner; this phase provides reusable helpers for:

- Redeeming the bootstrap token.
- Creating a tenant.
- Configuring email and upstream providers.
- Starting passkey and Google-upstream sign-in flows.

Run manually once a local stack is running:

```sh
bunx playwright test tests/e2e/examples-smoke.spec.ts
```
