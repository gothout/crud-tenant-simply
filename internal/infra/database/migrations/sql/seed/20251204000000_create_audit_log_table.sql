CREATE TABLE audit_log (
   id SERIAL PRIMARY KEY,

   tenant_uuid UUID,
   user_uuid   UUID,

   identifier TEXT,

   ray_trace_code VARCHAR(100) NOT NULL,

   domain     VARCHAR(100) NOT NULL,
   action     VARCHAR(100) NOT NULL,
   function   VARCHAR(150) NOT NULL,
   success    BOOLEAN NOT NULL,

   input_data  TEXT,
   output_data TEXT,

   created_at TIMESTAMP NOT NULL
);
-- Tenant
CREATE INDEX idx_audit_log_tenant_uuid
    ON audit_log (tenant_uuid);

-- User
CREATE INDEX idx_audit_log_user_uuid
    ON audit_log (user_uuid);

-- RayTraceCode (principal para rastreabilidade)
CREATE INDEX idx_audit_log_ray_trace_code
    ON audit_log (ray_trace_code);

-- Domain
CREATE INDEX idx_audit_log_domain
    ON audit_log (domain);

-- Action
CREATE INDEX idx_audit_log_action
    ON audit_log (action);

-- Consulta combinada Domain + Action
CREATE INDEX idx_audit_log_domain_action
    ON audit_log (domain, action);

-- created_at (para ordenação por logs recentes)
CREATE INDEX idx_audit_log_created_at_desc
    ON audit_log (created_at DESC);
