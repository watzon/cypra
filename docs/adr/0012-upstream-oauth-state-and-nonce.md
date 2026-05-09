# ADR-0012: Upstream OAuth State and Nonce

Status: accepted

Cypra will require signed OAuth `state` and validated `nonce` on upstream callbacks to prevent replay and callback tampering.

## Context

Cypra supports upstream Google OAuth as a tenant sign-in method. OAuth callback parameters cross browser and third-party boundaries, so Cypra must bind a callback to the tenant and login continuation it originally issued.

OIDC also requires `nonce` validation for browser-based authentication so an ID token from a different flow cannot be replayed into the current flow.

## Decision

The Google upstream client creates a signed `state` bundle containing `tenant_id`, `return_url`, `nonce`, and an expiry. The bundle is JSON encoded, base64url encoded, and signed with HMAC-SHA256. Callback handling rejects malformed, expired, or tampered state with `auth.upstream_state_mismatch`.

The same nonce is sent in the upstream authorization request. Callback handling parses the upstream ID token claims and requires the echoed `nonce` to match the signed state nonce. Missing or mismatched nonce is rejected with `auth.upstream_nonce_mismatch`.

## Consequences

- OAuth callbacks are pinned to the original tenant and return URL.
- State tampering and callback swapping fail before any session is created.
- Stubbed upstream tests can exercise the security checks without real Google credentials.
- The state signing secret must be operator-held and not logged.
