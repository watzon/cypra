ALTER TABLE storage_objects
    ADD COLUMN owner_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX storage_objects_owner_user_idx ON storage_objects (tenant_id, owner_user_id)
    WHERE owner_user_id IS NOT NULL;
