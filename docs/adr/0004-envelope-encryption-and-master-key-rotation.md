# ADR-0004: Envelope Encryption and Master Key Rotation

Status: accepted

Cypra will encrypt row secrets under per-row DEKs wrapped by an in-memory master KEK, with online resumable rewrap for master-key rotation.

## Context

Cypra stores secrets such as OIDC client secrets, upstream provider credentials, TOTP secrets, email-provider configuration, and signing private keys. Database compromise should not expose those plaintext values without the operator-held master key.

## Decision

Secret rows use envelope encryption. Each write generates a random 256-bit DEK and encrypts plaintext with AES-256-GCM. The DEK is wrapped by the process master KEK, which is loaded once from `MASTER_KEY` or `MASTER_KEY_FILE` and kept in memory only.

Master-key rotation is an online rewrap: `master_key_rotations` tracks `phase`, `rows_total`, `rows_done`, and errors. Rotation code iterates registered encrypted columns, unwraps each DEK with the old KEK, wraps it under the new KEK, and updates `rows_done` after each row. Resume starts from `rows_done`; cutover marks `phase='done'`.

Encrypted columns use two persisted shapes. Most row secrets store a binary prefix envelope: `encrypted_dek || ciphertext`. OIDC signing private keys store a JSON envelope with separate `encrypted_dek` and `ciphertext` fields. Rotation targets declare their shape explicitly so only the wrapped DEK changes; ciphertext bytes are preserved.

Each row rewrap and its `rows_done` update commit in the same database transaction. A crash before commit leaves both unchanged; a crash after commit leaves both advanced. Resume can therefore restart from `rows_done` without trying to unwrap an already-rewrapped row with the old KEK.

## Consequences

- Losing the KEK is irrecoverable for encrypted rows.
- Rotation can recover after process interruption without restarting from the first row and without corrupting mixed envelope shapes.
- New encrypted columns must be added to the rotation target list and covered by tests.
