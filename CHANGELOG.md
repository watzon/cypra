# Changelog

## v0.1.0 - Unreleased

Cypra v0.1 is the first controlled external-tester release candidate of the self-hosted, multi-tenant auth platform. It is intended for technical operators who can run Docker Compose on a VPS and follow the deployment playbook.

### Added

- Go server with embedded React dashboard, Postgres migrations, health/readiness checks, metrics, and local storage.
- Tenant isolation through host resolver, `TenantScopedDB`, and Postgres RLS.
- Hosted login, passkeys, password, magic-link, TOTP, backup codes, and Google upstream OAuth primitives.
- Per-tenant OIDC provider with discovery, JWKS, authorize, token, userinfo, revoke, consent, refresh-token rotation, and reuse detection.
- Instance-admin bootstrap, recovery, dashboard diagnostics, provider config, audit log, GDPR export/delete, backup export/import, PATs, and Go SDK.
- Local canonical demo, Lighthouse-style baseline, critical coverage floor, VPS deployment guide, backup/restore procedure, upgrade/rollback guidance, and operator playbooks.

### Changed

- Hosted login no longer depends on an external CDN. HTMX 2.0.4 is vendored under `internal/hostedlogin/static/htmx.min.js`, served from the Cypra origin, and pinned with a SHA-384 SRI hash in `templates/base.html`. The dependency-update procedure lives in `docs/playbook/dependencies.md`.
- Hosted-login passkey flows now surface specific user-facing messages for cancellation, unsupported browser/authenticator, network failure, and origin/RP rejection without leaking server detail.
- The 2FA challenge UI splits TOTP, security-key (WebAuthn), and backup-code into separate panels with factor-appropriate inputs and copy instead of a single shared code field.
- Invite onboarding clarifies which auth methods are required versus optional and previews the post-redemption passkey enrollment step.
- Dark-mode `--text-tertiary` was raised to `#8B8B93` and the consent "+ new" badge was restyled so hosted-login passes axe-core with zero violations in both color schemes and on overridden tenant accents.
- Removed the production-visible `noop` bot-verifier placeholder markup from login and signup; bot-verifier integration remains a v1.1 limitation.

### Known Limitations

- No SAML.
- No embeddable widget.
- No TypeScript SDK.
- No built-in CNAME management.
- No SMS/Twilio.
- No bundled Turnstile/hCaptcha provider; the verifier seam exists for v1.1.
- Dashboard is desktop-optimized for technical operators in this tester wave.
- Railway one-click template publication is approved as a post-tester deferral.
- Fresh-VPS release-candidate trial evidence, including deployed smoke, metrics, tracing, and human stopwatch timing, is tracked in the readiness plan before external tester rollout.
