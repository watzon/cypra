# Testing

## Local Gates

Run the same pipeline used by CI before committing:

```sh
make ci-pipeline
./bin/agent-ci run --quiet --all
```

The dashboard baseline check runs real Lighthouse against canonical local routes:

```sh
make lighthouse-baseline
```

The baseline runner starts the dashboard Vite server, visits `/setup/cypra_setup_test` and `/dashboard/tenants?state=demo`, records Lighthouse scores and web-vital metrics in `test-results/lighthouse/latest.json`, and compares the route set against `tests/e2e/lighthouse-baseline.json`. It enforces per-route thresholds for performance, accessibility, and best practices.

Managed browser e2e tests boot local Postgres, Cypra, SMTP, Google stub, and example-app dependencies for the current test process:

```sh
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/examples-smoke.spec.ts
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/canonical-demo/canonical-demo.spec.ts
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/backup-import.spec.ts
CYPRA_E2E_MANAGED=1 bun run test:e2e -- tests/e2e/accessibility.spec.ts
```

## Coverage Floors

Security-critical packages are tracked individually:

```sh
make test-go-coverage
```

The CI floor is at least 85% line coverage for each package: `internal/crypto`, `internal/oidc`, `internal/auth`, `internal/db`, `internal/sessions`, and `internal/bootstrap`. The assertion runs per package so one high-coverage package cannot mask a drop in another security-critical package.
