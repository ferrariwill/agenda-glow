-- AgendaGlow: multi-tenancy por estabelecimento (slug na URL pública).

CREATE TABLE estabelecimentos (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome_comercial VARCHAR(255) NOT NULL,
    slug           VARCHAR(100) NOT NULL UNIQUE,
    ativo          BOOLEAN NOT NULL DEFAULT TRUE,
    data_cadastro  TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_estabelecimentos_slug_formato
        CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$')
);

CREATE INDEX idx_estabelecimentos_slug_ativo
    ON estabelecimentos (slug)
    WHERE ativo = TRUE;

-- Estabelecimento padrão para backfill de dados legados.
INSERT INTO estabelecimentos (nome_comercial, slug)
VALUES ('Estabelecimento Padrão', 'estabelecimento-padrao');

-- Profissionais
ALTER TABLE profissionais
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE profissionais
SET estabelecimento_id = (SELECT id FROM estabelecimentos WHERE slug = 'estabelecimento-padrao');

ALTER TABLE profissionais
    ALTER COLUMN estabelecimento_id SET NOT NULL,
    ADD UNIQUE (estabelecimento_id, id);

CREATE INDEX idx_profissionais_estabelecimento ON profissionais (estabelecimento_id);

-- Serviços
ALTER TABLE servicos
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE servicos
SET estabelecimento_id = (SELECT id FROM estabelecimentos WHERE slug = 'estabelecimento-padrao');

ALTER TABLE servicos
    ALTER COLUMN estabelecimento_id SET NOT NULL,
    ADD UNIQUE (estabelecimento_id, id);

CREATE INDEX idx_servicos_estabelecimento ON servicos (estabelecimento_id);

-- Clientes (telefone único por estabelecimento)
ALTER TABLE clientes
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE clientes
SET estabelecimento_id = (SELECT id FROM estabelecimentos WHERE slug = 'estabelecimento-padrao');

ALTER TABLE clientes
    DROP CONSTRAINT IF EXISTS clientes_telefone_key,
    ALTER COLUMN estabelecimento_id SET NOT NULL,
    ADD CONSTRAINT clientes_estabelecimento_telefone_unique
        UNIQUE (estabelecimento_id, telefone),
    ADD UNIQUE (estabelecimento_id, id);

CREATE INDEX idx_clientes_estabelecimento ON clientes (estabelecimento_id);

-- Planos de assinatura
ALTER TABLE planos_assinatura
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE planos_assinatura
SET estabelecimento_id = (SELECT id FROM estabelecimentos WHERE slug = 'estabelecimento-padrao');

ALTER TABLE planos_assinatura
    ALTER COLUMN estabelecimento_id SET NOT NULL,
    ADD UNIQUE (estabelecimento_id, id);

CREATE INDEX idx_planos_assinatura_estabelecimento ON planos_assinatura (estabelecimento_id);

-- Agendamentos
ALTER TABLE agendamentos
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE agendamentos a
SET estabelecimento_id = p.estabelecimento_id
FROM profissionais p
WHERE p.id = a.profissional_id;

ALTER TABLE agendamentos
    ALTER COLUMN estabelecimento_id SET NOT NULL,
    ADD CONSTRAINT fk_agendamentos_profissional_tenant
        FOREIGN KEY (estabelecimento_id, profissional_id)
        REFERENCES profissionais (estabelecimento_id, id),
    ADD CONSTRAINT fk_agendamentos_servico_tenant
        FOREIGN KEY (estabelecimento_id, servico_id)
        REFERENCES servicos (estabelecimento_id, id),
    ADD CONSTRAINT fk_agendamentos_cliente_tenant
        FOREIGN KEY (estabelecimento_id, cliente_id)
        REFERENCES clientes (estabelecimento_id, id);

CREATE INDEX idx_agendamentos_estabelecimento ON agendamentos (estabelecimento_id);

-- Fluxo de caixa
ALTER TABLE fluxo_caixa
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE fluxo_caixa fc
SET estabelecimento_id = p.estabelecimento_id
FROM profissionais p
WHERE fc.profissional_id = p.id;

UPDATE fluxo_caixa
SET estabelecimento_id = (SELECT id FROM estabelecimentos WHERE slug = 'estabelecimento-padrao')
WHERE estabelecimento_id IS NULL;

ALTER TABLE fluxo_caixa
    ALTER COLUMN estabelecimento_id SET NOT NULL;

CREATE INDEX idx_fluxo_caixa_estabelecimento ON fluxo_caixa (estabelecimento_id);

-- Assinantes do clube
ALTER TABLE clientes_assinantes
    ADD COLUMN estabelecimento_id UUID REFERENCES estabelecimentos (id) ON DELETE CASCADE;

UPDATE clientes_assinantes ca
SET estabelecimento_id = pa.estabelecimento_id
FROM planos_assinatura pa
WHERE pa.id = ca.plano_id;

ALTER TABLE clientes_assinantes
    DROP CONSTRAINT IF EXISTS clientes_assinantes_cliente_telefone_key,
    ALTER COLUMN estabelecimento_id SET NOT NULL,
    ADD CONSTRAINT clientes_assinantes_estabelecimento_telefone_unique
        UNIQUE (estabelecimento_id, cliente_telefone),
    ADD CONSTRAINT fk_assinantes_plano_tenant
        FOREIGN KEY (estabelecimento_id, plano_id)
        REFERENCES planos_assinatura (estabelecimento_id, id);

CREATE INDEX idx_clientes_assinantes_estabelecimento ON clientes_assinantes (estabelecimento_id);
