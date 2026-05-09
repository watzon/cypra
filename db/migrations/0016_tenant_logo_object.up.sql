ALTER TABLE tenants
    ADD COLUMN logo_object_id UUID NULL REFERENCES storage_objects(id) ON DELETE SET NULL;

CREATE INDEX tenants_logo_object_idx ON tenants (logo_object_id) WHERE logo_object_id IS NOT NULL;
