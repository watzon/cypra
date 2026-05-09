DROP POLICY IF EXISTS tenant_isolation ON personal_access_tokens;
ALTER TABLE personal_access_tokens DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS personal_access_tokens;
