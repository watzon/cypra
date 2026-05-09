DROP INDEX IF EXISTS sessions_subject_active_idx;

ALTER TABLE sessions DROP CONSTRAINT IF EXISTS sessions_auth_kind_check;
ALTER TABLE sessions DROP COLUMN IF EXISTS auth_kind;

ALTER TABLE users DROP COLUMN IF EXISTS display_name;
