DROP TRIGGER IF EXISTS tenant_memberships_revoke_pats_on_delete ON tenant_memberships;
DROP TRIGGER IF EXISTS tenant_memberships_revoke_pats_on_update ON tenant_memberships;
DROP FUNCTION IF EXISTS revoke_pats_on_membership_change();
