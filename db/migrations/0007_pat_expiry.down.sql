DROP INDEX IF EXISTS personal_access_tokens_live_idx;

ALTER TABLE personal_access_tokens
    DROP COLUMN IF EXISTS expires_at;
