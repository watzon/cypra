CREATE OR REPLACE FUNCTION revoke_pats_on_membership_change()
RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        UPDATE personal_access_tokens
        SET revoked_at = COALESCE(revoked_at, now())
        WHERE tenant_id = OLD.tenant_id
          AND user_id = OLD.user_id
          AND revoked_at IS NULL;
        RETURN OLD;
    END IF;

    IF OLD.role IS DISTINCT FROM NEW.role AND NEW.role NOT IN ('owner', 'admin') THEN
        UPDATE personal_access_tokens
        SET revoked_at = COALESCE(revoked_at, now())
        WHERE tenant_id = NEW.tenant_id
          AND user_id = NEW.user_id
          AND revoked_at IS NULL;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tenant_memberships_revoke_pats_on_update
AFTER UPDATE OF role ON tenant_memberships
FOR EACH ROW
EXECUTE FUNCTION revoke_pats_on_membership_change();

CREATE TRIGGER tenant_memberships_revoke_pats_on_delete
AFTER DELETE ON tenant_memberships
FOR EACH ROW
EXECUTE FUNCTION revoke_pats_on_membership_change();
