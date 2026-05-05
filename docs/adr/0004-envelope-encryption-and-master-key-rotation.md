# ADR-0004: Envelope Encryption and Master Key Rotation

Status: proposed

Cypra will encrypt row secrets under per-row DEKs wrapped by an in-memory master KEK, with online resumable rewrap for master-key rotation.
