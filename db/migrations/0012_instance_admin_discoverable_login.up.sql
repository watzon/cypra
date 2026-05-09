-- Discoverable login starts a webauthn ceremony before the admin id is known
-- (the browser picks the credential, server identifies the admin afterward).
-- Until now the column was NOT NULL, forcing email-bound flows.

ALTER TABLE instance_admin_webauthn_challenges
    ALTER COLUMN instance_admin_id DROP NOT NULL;
