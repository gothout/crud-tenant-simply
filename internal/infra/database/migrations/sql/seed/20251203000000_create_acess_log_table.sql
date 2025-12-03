CREATE TABLE IF NOT EXISTS access_log (
  id              BIGSERIAL PRIMARY KEY,

  tenant_uuid     UUID,
  user_uuid       UUID,
  identifier      TEXT,                 -- email, documento, username etc.

  ray_trace_code  VARCHAR(100) NOT NULL,

  method          VARCHAR(10)  NOT NULL,
  path            TEXT         NOT NULL,
  host            TEXT         NOT NULL,
  status_code     INTEGER      NOT NULL,

  ip              INET         NOT NULL,
  user_agent      TEXT,
  referer         TEXT,
  content_type    TEXT,
  user_language   TEXT,

  request_time    TIMESTAMP WITHOUT TIME ZONE NOT NULL,
  latency_ms      NUMERIC(10,3) NOT NULL,

  created_at      TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

-- Índice para busca por tenant e ordenação por data (mais usado)
CREATE INDEX IF NOT EXISTS idx_access_log_tenant_date
    ON access_log (tenant_uuid, created_at DESC);

-- Índice para buscar logs por usuário
CREATE INDEX IF NOT EXISTS idx_access_log_user_date
    ON access_log (user_uuid, created_at DESC);

-- Busca rápida pelo identificador humano
CREATE INDEX IF NOT EXISTS idx_access_log_identifier
    ON access_log (identifier);

-- Para rastreamento via RayTrace / X-Request-ID
CREATE INDEX IF NOT EXISTS idx_access_log_ray_trace
    ON access_log (ray_trace_code);

-- Para análises de endpoints mais acessados
CREATE INDEX IF NOT EXISTS idx_access_log_path
    ON access_log (path);

-- Para filtros por status HTTP
CREATE INDEX IF NOT EXISTS idx_access_log_status
    ON access_log (status_code);
