CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS tenant (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    document VARCHAR(100) NOT NULL UNIQUE,
    live BOOLEAN NOT NULL DEFAULT TRUE,
    create_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
    update_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW()
);

-- (Opcional, mas recomendado) Cria um tipo ENUM para os papéis
CREATE TYPE user_role AS ENUM (
    'SYSTEM_ADMIN',  -- Administrador global do sistema (tenant_uuid = NULL)
    'TENANT_ADMIN',  -- Administrador de um tenant específico (tenant_uuid != NULL)
    'TENANT_USER'    -- Usuário comum de um tenant (tenant_uuid != NULL)
    );

CREATE TABLE IF NOT EXISTS users (
     uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Chave estrangeira para o tenant.
    -- É NULÁVEL: se for NULL, é um usuário global/sistema.
    -- Se tiver um valor, pertence ao tenant correspondente.
     tenant_uuid UUID,

     name VARCHAR(255) NOT NULL,
     email VARCHAR(255) NOT NULL UNIQUE,
     password_hash VARCHAR(255) NOT NULL,

     role user_role NOT NULL DEFAULT 'TENANT_USER',
     live BOOLEAN NOT NULL DEFAULT TRUE,
     create_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),
     update_at TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT NOW(),

     CONSTRAINT fk_tenant
         FOREIGN KEY(tenant_uuid)
             REFERENCES tenant(uuid)
             ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS users_acess_tokens (
    user_uuid UUID,
    token TEXT NOT NULL UNIQUE,
    expire_date TIMESTAMP WITHOUT TIME ZONE NOT NULL,
    CONSTRAINT fk_user
        FOREIGN KEY(user_uuid)
            REFERENCES users(uuid)
            ON DELETE SET NULL
)