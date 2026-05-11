# Third-party browser dependency policy

Hosted-login pages must run without depending on external CDNs at request time. Any browser script or stylesheet not authored in this repository is vendored under `internal/hostedlogin/static/` and served from the Cypra origin. The Go embed `//go:embed templates/*.html static/*.js` in `internal/hostedlogin/render.go` ships these assets with the binary.

## Why vendor

- A CDN outage must not block sign-in.
- A CDN compromise must not be able to inject arbitrary JS into a tenant's hosted-login.
- Operators on air-gapped or egress-restricted networks should not have to whitelist third-party origins.

## Currently vendored

| Asset    | Version | File                                      | Source                                            |
| -------- | ------- | ----------------------------------------- | ------------------------------------------------- |
| htmx.org | 2.0.4   | `internal/hostedlogin/static/htmx.min.js` | https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js |

Subresource Integrity (SRI) hashes are also pinned in `internal/hostedlogin/templates/base.html`. The SRI hash is defense-in-depth: the file is already trusted because it ships in the binary, but the SRI annotation makes accidental drift between the file on disk and the `<script integrity="…">` attribute caught at first page load.

## Update procedure

1. Decide which version you want, with a security/CVE rationale.
2. Download the minified artifact from the upstream registry (npm tarball or `unpkg.com/<pkg>@<version>/dist/<file>`):

   ```sh
   curl -sL --fail https://unpkg.com/htmx.org@<version>/dist/htmx.min.js -o internal/hostedlogin/static/htmx.min.js
   ```

3. Compute the SHA-384 SRI hash and update `base.html`:

   ```sh
   openssl dgst -sha384 -binary internal/hostedlogin/static/htmx.min.js | base64
   ```

4. Run `./bin/agent-ci run --quiet --all`. The hosted-login render tests assert the SRI string is present and the embedded file is served under `/static/hostedlogin/`.
5. Commit the binary asset, the SRI hash change, and a `CHANGELOG.md` entry under `Changed`.

## What we do not vendor

Browser-native APIs (WebAuthn, `fetch`, `URLSearchParams`) are used directly without polyfills. Cypra hosted-login supports current versions of Chromium, Firefox, and Safari; older browsers see a graceful unsupported-browser message from `passkey.js`.
