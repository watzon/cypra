# ADR-0005: OIDC Signing Key Rotation Schedule

Status: proposed

Cypra will rotate tenant OIDC signing keys on a 90-day cadence with a 30-day overlap and cache-aware JWKS behavior.
