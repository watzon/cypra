# Testing

## Local Gates

Run the same pipeline used by CI before committing:

```sh
make ci-pipeline
./bin/agent-ci run --quiet --all
```

Phase 13 adds a dashboard baseline check for the canonical local routes:

```sh
make lighthouse-baseline
```

The baseline runner starts the dashboard locally, visits `/setup/cypra_setup_test` and `/dashboard/tenants?state=demo`, runs axe, records `test-results/lighthouse/latest.json`, and compares against `tests/e2e/lighthouse-baseline.json`.

## Coverage Floors

Security-critical packages are tracked individually:

```sh
make test-go-coverage
```

The CI floor is at least 85% line coverage for each package: `internal/crypto`, `internal/oidc`, `internal/auth`, `internal/db`, `internal/sessions`, and `internal/bootstrap`. The assertion runs per package so one high-coverage package cannot mask a drop in another security-critical package.
