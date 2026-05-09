# Cypra E2E Harness

Playwright harness for live-stack and managed-stack browser coverage.

The harness can run against an existing local Cypra stack or can start a managed local Cypra + Postgres stack for the current test process. The managed path includes a deterministic SMTP stub, Google upstream stub, and downstream Next.js app orchestration.

Reusable helpers cover:

- Redeeming the bootstrap token.
- Enrolling the first instance-admin passkey with a virtual authenticator.
- Creating a tenant and project.
- Configuring email against the SMTP stub and upstream providers against the Google stub.
- Starting the passkey sign-in flow, enrolling/verifying WebAuthn second factor, and completing Google-upstream callback/session creation through the local stub.
- Exporting a managed-stack backup, importing it into a fresh managed stack, and reusing the same virtual authenticator to assert a restored tenant passkey.

Run against an existing local stack:

```sh
CYPRA_E2E_LIVE=1 \
CYPRA_BASE_URL=https://cypra.localhost \
CYPRA_TENANT_URL=https://acme.cypra.localhost \
CYPRA_SETUP_TOKEN=<token> \
bunx playwright test tests/e2e/examples-smoke.spec.ts
```

Run with a managed local stack:

```sh
CYPRA_E2E_MANAGED=1 bunx playwright test tests/e2e/canonical-demo/canonical-demo.spec.ts
```

Run backup/import browser proof:

```sh
CYPRA_E2E_MANAGED=1 bunx playwright test tests/e2e/backup-import.spec.ts
```

Run the accessibility route matrix:

```sh
CYPRA_E2E_MANAGED=1 bunx playwright test tests/e2e/accessibility.spec.ts
```

The managed path starts a fresh Compose-scoped Postgres service, runs migrations, mints a setup token, launches `cypra serve`, runs the browser flow, then tears the stack down with volumes removed.
