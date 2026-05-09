# ADR-0015: Structural Audit Write Enforcement

Status: accepted

Cypra will enforce audit writes through a narrow HTTP-server wrapper instead of allowing mutating handlers to call the audit package directly.

## Context

The plan requires mutating tenant and instance-admin endpoints to emit audit entries with actor, action, resource, and before/after state where applicable. Direct `audit.Write` calls spread across handlers are easy to omit, hard to review, and hard to lint.

## Decision

HTTP handlers use `Server.writeAudit` as the narrow audit-write path. The Makefile `lint-audit-writes` target fails if direct `audit.Write` calls appear outside the approved wrapper and tests.

The wrapper keeps audit behavior close to the HTTP actor/tenant context while preserving an explicit, searchable action string at each mutating endpoint.

## Consequences

- New mutating handlers must either call the wrapper or intentionally extend the allowlist.
- Audit coverage can be checked structurally in CI instead of relying only on integration tests.
- The wrapper is not a substitute for endpoint-specific before/after state tests; high-risk mutations still need behavioral assertions.
