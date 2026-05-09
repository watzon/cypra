DROP TABLE IF EXISTS invite_continuations;

DROP TABLE IF EXISTS webauthn_challenges;

DROP INDEX IF EXISTS passkey_credentials_tenant_purpose_idx;
ALTER TABLE passkey_credentials
    DROP COLUMN IF EXISTS credential,
    DROP COLUMN IF EXISTS purpose;
