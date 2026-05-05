-- Postgres role separation for Cypra.
-- Operators run DDL through MIGRATE_DATABASE_URL (`cypra_migrate`) and the
-- application runtime through DATABASE_URL (`cypra_runtime`). If the migrate
-- URL falls back to the runtime URL later, `cypra serve` must warn loudly.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cypra_migrate') THEN
        CREATE ROLE cypra_migrate NOLOGIN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'cypra_runtime') THEN
        CREATE ROLE cypra_runtime NOLOGIN;
    END IF;
END
$$;

GRANT USAGE, CREATE ON SCHEMA public TO cypra_migrate;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO cypra_migrate;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO cypra_migrate;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO cypra_migrate;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO cypra_migrate;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO cypra_migrate;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO cypra_migrate;

GRANT USAGE ON SCHEMA public TO cypra_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cypra_runtime;
REVOKE UPDATE, DELETE ON audit_entries FROM cypra_runtime;
GRANT SELECT, INSERT ON audit_entries TO cypra_runtime;
GRANT UPDATE (redacted_at, state_before, state_after, metadata, ip, user_agent) ON audit_entries TO cypra_runtime;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO cypra_runtime;
REVOKE DELETE ON audit_entries FROM cypra_runtime;
REVOKE UPDATE ON audit_entries FROM cypra_runtime;
