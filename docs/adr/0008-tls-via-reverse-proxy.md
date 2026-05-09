# ADR-0008: TLS via Reverse Proxy

Status: accepted

Cypra will rely on an operator-managed reverse proxy for TLS, with Caddy as the reference deployment and explicit trusted-header modes.

## Context

Cypra needs HTTPS for passkeys, secure cookies, and OIDC redirect integrity. Shipping a built-in ACME/TLS stack would duplicate mature reverse proxies and complicate self-hosted network topologies.

## Decision

Cypra terminates HTTP behind an operator-managed reverse proxy. Caddy is the reference deployment and `deploy/Caddyfile.example` documents TLS termination plus the trusted host-forwarding headers Cypra accepts.

`TRUSTED_PROXY_HEADERS` remains explicit. Cypra only honors forwarded host headers when the proxy marks the request with `X-Cypra-Trusted-Proxy: true`, avoiding accidental tenant-host spoofing from arbitrary clients.

## Consequences

Operators can use Caddy, nginx, Traefik, or cloud load balancers without Cypra owning certificates. Local development can run the `with-tls` compose profile for a reference HTTPS stack, while production hardening remains a deployment concern.
