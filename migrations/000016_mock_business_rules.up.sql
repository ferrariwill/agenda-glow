-- Regras de negócio do protótipo React (mockDb) → schema PostgreSQL.

-- 1) Perfil SECRETARIA (mesmo escopo da dona: estabelecimento_id, sem profissional_id)
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('SUPER_ADMIN', 'DONA', 'PROFISSIONAL', 'SECRETARIA'));

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_escopo;
ALTER TABLE users ADD CONSTRAINT users_role_escopo CHECK (
    (role = 'SUPER_ADMIN' AND estabelecimento_id IS NULL AND profissional_id IS NULL)
    OR (role = 'DONA' AND estabelecimento_id IS NOT NULL AND profissional_id IS NULL)
    OR (role = 'SECRETARIA' AND estabelecimento_id IS NOT NULL AND profissional_id IS NULL)
    OR (role = 'PROFISSIONAL' AND estabelecimento_id IS NOT NULL AND profissional_id IS NOT NULL)
);

-- 2) Agendamento: encaixe, fila de espera e cobrança
ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS minutos_invadidos INTEGER NOT NULL DEFAULT 0
        CHECK (minutos_invadidos >= 0);

ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS aceita_adiantar BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS valor_cobrado NUMERIC(10, 2)
        CHECK (valor_cobrado IS NULL OR valor_cobrado >= 0);

ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS metodo_pagamento VARCHAR(30)
        CHECK (metodo_pagamento IS NULL OR metodo_pagamento IN (
            'PIX', 'CARTAO_DEBITO', 'CARTAO_CREDITO', 'DINHEIRO'
        ));

ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS cobrado_em DATE;

-- 3) Profissional pendente de aprovação (não conta no limite do plano)
ALTER TABLE profissionais
    ADD COLUMN IF NOT EXISTS pendente_aprovacao BOOLEAN NOT NULL DEFAULT FALSE;

-- 4) Fila de espera (aceitar adiantar horário)
CREATE TABLE IF NOT EXISTS fila_espera (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    profissional_id    UUID NOT NULL REFERENCES profissionais (id) ON DELETE CASCADE,
    cliente_nome       VARCHAR(255) NOT NULL,
    cliente_telefone   VARCHAR(20) NOT NULL,
    status             VARCHAR(20) NOT NULL DEFAULT 'AGUARDANDO'
        CHECK (status IN ('AGUARDANDO', 'NOTIFICADO', 'ATENDIDO')),
    agendamento_id     UUID REFERENCES agendamentos (id) ON DELETE SET NULL,
    data_desejada      DATE,
    hora_desejada      TIME,
    criado_em          TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fila_espera_prof_status
    ON fila_espera (profissional_id, status)
    WHERE status = 'AGUARDANDO';

CREATE UNIQUE INDEX IF NOT EXISTS idx_fila_espera_prof_tel_aguardando
    ON fila_espera (profissional_id, cliente_telefone)
    WHERE status = 'AGUARDANDO';

-- 5) Insumos / estoque
CREATE TABLE IF NOT EXISTS insumos (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    nome               VARCHAR(255) NOT NULL,
    marca              VARCHAR(255),
    categoria          VARCHAR(50) NOT NULL DEFAULT 'GERAL',
    quantidade         NUMERIC(12, 3) NOT NULL DEFAULT 0 CHECK (quantidade >= 0),
    estoque_minimo     NUMERIC(12, 3) NOT NULL DEFAULT 0 CHECK (estoque_minimo >= 0),
    estoque_ideal      NUMERIC(12, 3) NOT NULL DEFAULT 0 CHECK (estoque_ideal >= 0),
    valor_unitario     NUMERIC(10, 2) NOT NULL DEFAULT 0 CHECK (valor_unitario >= 0),
    unidade            VARCHAR(20) NOT NULL DEFAULT 'un',
    imagem_url         TEXT,
    instrucoes_uso     TEXT,
    ativo              BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em          TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_insumos_estabelecimento
    ON insumos (estabelecimento_id);

-- 6) Usuário secretaria dev (mesmo tenant do seed 000013)
INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, ativo)
VALUES (
    'secretaria@glow.local',
    '$2a$10$nCM70BH8Af1d9SXwe4zms.kmxpM9FIr/VkwT5ZbkhUfcKa.z88ZDu',
    'SECRETARIA',
    'a1000002-0002-4002-8002-000000000002',
    NULL,
    TRUE
)
ON CONFLICT (email) DO UPDATE SET
    role = 'SECRETARIA',
    estabelecimento_id = EXCLUDED.estabelecimento_id,
    ativo = TRUE;
