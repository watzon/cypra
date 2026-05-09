DROP INDEX IF EXISTS storage_objects_owner_user_idx;

ALTER TABLE storage_objects
    DROP COLUMN IF EXISTS owner_user_id;
