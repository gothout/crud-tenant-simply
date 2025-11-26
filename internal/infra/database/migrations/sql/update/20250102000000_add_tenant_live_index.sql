CREATE INDEX IF NOT EXISTS idx_tenant_live ON tenant (live);
CREATE INDEX IF NOT EXISTS idx_users_tenant_uuid ON users(tenant_uuid);