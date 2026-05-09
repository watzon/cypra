DROP INDEX IF EXISTS tenants_logo_object_idx;
ALTER TABLE tenants DROP COLUMN IF EXISTS logo_object_id;
