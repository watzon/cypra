# Changelog

## v0.1.0 - Unreleased

Cypra v0.1 is the first locally runnable release candidate of the self-hosted, multi-tenant auth platform.

### Added

- Go server with embedded React dashboard, Postgres migrations, health/readiness checks, metrics, and local storage.
- Tenant isolation through host resolver, `TenantScopedDB`, and Postgres RLS.
- Hosted login, passkeys, password, magic-link, TOTP, backup codes, and Google upstream OAuth primitives.
- Per-tenant OIDC provider with discovery, JWKS, authorize, token, userinfo, revoke, consent, refresh-token rotation, and reuse detection.
- Instance-admin bootstrap, recovery, dashboard diagnostics, provider config, audit log, GDPR export/delete, backup export/import, PATs, and Go SDK.
- Local canonical demo, Lighthouse-style baseline, critical coverage floor, and operator docs.

### Known Limitations

- No SAML.
- No embeddable widget.
- No TypeScript SDK.
- No built-in CNAME management.
- No SMS/Twilio.
- No bundled Turnstile/hCaptcha provider; the verifier seam exists for v1.1.
- Published multi-arch image, Railway template, release tag, and deployed smoke tests are deployment-phase work.
