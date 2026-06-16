-- AgendaGlow: planos corporativos SaaS — mensalidades cobradas dos estabelecimentos.

CREATE TABLE planos_saas (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome                 VARCHAR(255) NOT NULL,
    preco_mensal         NUMERIC(10, 2) NOT NULL CHECK (preco_mensal >= 0),
    limite_profissionais INTEGER NOT NULL CHECK (limite_profissionais > 0),
    ativo                BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE assinaturas_estabelecimentos (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL UNIQUE REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    plano_id           UUID NOT NULL REFERENCES planos_saas (id),
    status             VARCHAR(50) NOT NULL DEFAULT 'ATIVO'
        CHECK (status IN ('ATIVO', 'PAGAMENTO_PENDENTE', 'SUSPENSO')),
    data_vencimento    DATE NOT NULL,
    atualizado_em      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_planos_saas_ativo ON planos_saas (ativo) WHERE ativo = TRUE;
CREATE INDEX idx_assinaturas_estabelecimentos_plano ON assinaturas_estabelecimentos (plano_id);
CREATE INDEX idx_assinaturas_estabelecimentos_vencimento ON assinaturas_estabelecimentos (data_vencimento);
