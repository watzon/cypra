# ADR-0012: Upstream OAuth State and Nonce

Status: proposed

Cypra will require signed OAuth `state` and validated `nonce` on upstream callbacks to prevent replay and callback tampering.
