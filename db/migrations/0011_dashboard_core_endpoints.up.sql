ALTER TABLE users ADD COLUMN display_name TEXT NOT NULL DEFAULT '';

ALTER TABLE sessions ADD COLUMN auth_kind TEXT NOT NULL DEFAULT 'cookie';
ALTER TABLE sessions ADD CONSTRAINT sessions_auth_kind_check CHECK (auth_kind IN ('cookie', 'pat'));

CREATE INDEX sessions_subject_active_idx
    ON sessions (subject_id, last_seen_at DESC)
    WHERE revoked_at IS NULL;
