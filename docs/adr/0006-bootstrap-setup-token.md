# ADR-0006: Bootstrap Setup Token

Status: accepted

Cypra will mint a single-use first-boot setup token that is log-printed with redaction metadata and recoverable through reset-bootstrap if plaintext is lost.

## Context

A fresh self-hosted install needs one safe path to create the first instance admin. There is no admin account yet, so the bootstrap mechanism must be self-contained, single-use, and recoverable if the operator loses the plaintext token after it is stored hashed.

## Decision

On first boot, Cypra mints one 32-byte random setup token, stores only its hash in `bootstrap_tokens`, and logs the plaintext token with `redacted-on-export=true`. The token expires after 24 hours and is redeemed atomically by setting `consumed_at` when it is live and unrevoked.

After redemption, the caller may create the first `instance_admin`. Once any instance admin exists, minting and reset-bootstrap are refused. `RevokeSetupToken` revokes live setup tokens and mints a fresh token only while no instance admins exist, covering the crash case where the DB commit succeeded but plaintext logging did not reach the operator.

## Consequences

- Bootstrap secrets are not stored in plaintext.
- Operators can recover from a lost first token before admin creation.
- After first admin creation, bootstrap is permanently unavailable until later explicit admin reset tooling is added.
