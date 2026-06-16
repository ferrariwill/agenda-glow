-- AgendaGlow: credenciais unificadas com papéis de acesso (Super Admin, Dona, Profissional).

CREATE TABLE users (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email              VARCHAR(255) NOT NULL UNIQUE,
    password_hash      VARCHAR(255) NOT NULL,
    role               VARCHAR(50) NOT NULL
        CHECK (role IN ('SUPER_ADMIN', 'DONA', 'PROFISSIONAL')),
    estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    profissional_id    UUID REFERENCES profissionais (id) ON DELETE CASCADE,
    ativo              BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em          TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT users_role_escopo CHECK (
        (role = 'SUPER_ADMIN' AND estabelecimento_id IS NULL AND profissional_id IS NULL)
        OR (role = 'DONA' AND estabelecimento_id IS NOT NULL AND profissional_id IS NULL)
        OR (role = 'PROFISSIONAL' AND estabelecimento_id IS NOT NULL AND profissional_id IS NOT NULL)
    )
);

CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_users_estabelecimento ON users (estabelecimento_id) WHERE estabelecimento_id IS NOT NULL;
CREATE INDEX idx_users_profissional ON users (profissional_id) WHERE profissional_id IS NOT NULL;
