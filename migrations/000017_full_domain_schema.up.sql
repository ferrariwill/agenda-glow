-- AgendaGlow: schema completo do domínio (paridade com mockDb / REGRAS_NEGOCIO).
-- Complementa 000001–000016 com tabelas e colunas ainda ausentes.

-- =============================================================================
-- Estabelecimentos (tenant / salão)
-- =============================================================================
ALTER TABLE estabelecimentos
    ADD COLUMN IF NOT EXISTS bio TEXT,
    ADD COLUMN IF NOT EXISTS email_contato VARCHAR(255),
    ADD COLUMN IF NOT EXISTS cep VARCHAR(10),
    ADD COLUMN IF NOT EXISTS logradouro VARCHAR(255),
    ADD COLUMN IF NOT EXISTS cidade VARCHAR(100),
    ADD COLUMN IF NOT EXISTS uf CHAR(2),
    ADD COLUMN IF NOT EXISTS dona_nome VARCHAR(255),
    ADD COLUMN IF NOT EXISTS dona_email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS dona_atua_como_profissional BOOLEAN NOT NULL DEFAULT FALSE;

-- =============================================================================
-- Usuários do sistema (staff)
-- =============================================================================
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS nome VARCHAR(255),
    ADD COLUMN IF NOT EXISTS telefone VARCHAR(20);

UPDATE users
SET nome = COALESCE(NULLIF(TRIM(nome), ''), SPLIT_PART(email, '@', 1))
WHERE nome IS NULL OR TRIM(nome) = '';

-- =============================================================================
-- Clientes do salão
-- =============================================================================
ALTER TABLE clientes
    ADD COLUMN IF NOT EXISTS ativo BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS data_nascimento DATE,
    ADD COLUMN IF NOT EXISTS foto_url TEXT,
    ADD COLUMN IF NOT EXISTS alergias TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS observacoes_medicas TEXT,
    ADD COLUMN IF NOT EXISTS preferencias TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_clientes_estabelecimento_ativo
    ON clientes (estabelecimento_id)
    WHERE ativo = TRUE;

-- =============================================================================
-- Profissionais (perfil estendido)
-- =============================================================================
ALTER TABLE profissionais
    ADD COLUMN IF NOT EXISTS foto_url TEXT,
    ADD COLUMN IF NOT EXISTS telefone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS biografia TEXT,
    ADD COLUMN IF NOT EXISTS portfolio_url TEXT,
    ADD COLUMN IF NOT EXISTS data_contratacao DATE,
    ADD COLUMN IF NOT EXISTS data_nascimento DATE,
    ADD COLUMN IF NOT EXISTS modelo_pagamento VARCHAR(20) NOT NULL DEFAULT 'PERCENTUAL'
        CHECK (modelo_pagamento IN ('PERCENTUAL', 'FIXO')),
    ADD COLUMN IF NOT EXISTS valor_fixo_atendimento NUMERIC(10, 2)
        CHECK (valor_fixo_atendimento IS NULL OR valor_fixo_atendimento >= 0),
    ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS eh_dona BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_profissionais_user_id
    ON profissionais (user_id)
    WHERE user_id IS NOT NULL;

-- =============================================================================
-- Categorias de serviço (catálogo)
-- =============================================================================
CREATE TABLE IF NOT EXISTS categorias_servicos (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    nome               VARCHAR(100) NOT NULL,
    icone              VARCHAR(50),
    UNIQUE (estabelecimento_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_categorias_servicos_estab_nome
    ON categorias_servicos (estabelecimento_id, LOWER(TRIM(nome)));

CREATE INDEX IF NOT EXISTS idx_categorias_servicos_estabelecimento
    ON categorias_servicos (estabelecimento_id);

-- =============================================================================
-- Serviços (catálogo estendido)
-- =============================================================================
ALTER TABLE servicos
    ADD COLUMN IF NOT EXISTS imagem_url TEXT,
    ADD COLUMN IF NOT EXISTS descricao TEXT,
    ADD COLUMN IF NOT EXISTS categoria_id UUID,
    ADD COLUMN IF NOT EXISTS exibir_catalogo_publico BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS permitir_agendamento_online BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE servicos DROP CONSTRAINT IF EXISTS fk_servicos_categoria_tenant;
ALTER TABLE servicos
    ADD CONSTRAINT fk_servicos_categoria_tenant
        FOREIGN KEY (estabelecimento_id, categoria_id)
        REFERENCES categorias_servicos (estabelecimento_id, id)
        ON DELETE SET NULL;

-- =============================================================================
-- Serviço ↔ Profissional (quem executa cada procedimento)
-- =============================================================================
CREATE TABLE IF NOT EXISTS servico_profissionais (
    estabelecimento_id UUID NOT NULL,
    servico_id         UUID NOT NULL,
    profissional_id    UUID NOT NULL,
    PRIMARY KEY (servico_id, profissional_id),
    CONSTRAINT fk_servico_prof_servico
        FOREIGN KEY (estabelecimento_id, servico_id)
        REFERENCES servicos (estabelecimento_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_servico_prof_profissional
        FOREIGN KEY (estabelecimento_id, profissional_id)
        REFERENCES profissionais (estabelecimento_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_servico_profissionais_prof
    ON servico_profissionais (profissional_id);

-- =============================================================================
-- Serviço ↔ Insumo (ficha técnica / BOM)
-- =============================================================================
-- insumos precisa de unique composto para FK (criado em 000016 sem unique)
ALTER TABLE insumos DROP CONSTRAINT IF EXISTS insumos_estabelecimento_id_id_key;
ALTER TABLE insumos
    ADD CONSTRAINT insumos_estabelecimento_id_id_key UNIQUE (estabelecimento_id, id);

CREATE TABLE IF NOT EXISTS servico_insumos (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL,
    servico_id         UUID NOT NULL,
    insumo_id          UUID NOT NULL,
    quantidade_uso     NUMERIC(12, 3) NOT NULL DEFAULT 1 CHECK (quantidade_uso > 0),
    UNIQUE (servico_id, insumo_id),
    CONSTRAINT fk_servico_insumos_servico
        FOREIGN KEY (estabelecimento_id, servico_id)
        REFERENCES servicos (estabelecimento_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_servico_insumos_insumo
        FOREIGN KEY (estabelecimento_id, insumo_id)
        REFERENCES insumos (estabelecimento_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_servico_insumos_servico
    ON servico_insumos (servico_id);

-- =============================================================================
-- Agendamentos (observações + multi-serviço)
-- =============================================================================
ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS observacoes TEXT;

CREATE TABLE IF NOT EXISTS agendamento_servicos (
    agendamento_id UUID NOT NULL REFERENCES agendamentos (id) ON DELETE CASCADE,
    servico_id     UUID NOT NULL,
    ordem          SMALLINT NOT NULL DEFAULT 0 CHECK (ordem >= 0),
    PRIMARY KEY (agendamento_id, servico_id)
);

CREATE INDEX IF NOT EXISTS idx_agendamento_servicos_servico
    ON agendamento_servicos (servico_id);

INSERT INTO agendamento_servicos (agendamento_id, servico_id, ordem)
SELECT a.id, a.servico_id, 0
FROM agendamentos a
WHERE NOT EXISTS (
    SELECT 1 FROM agendamento_servicos s WHERE s.agendamento_id = a.id
);

-- =============================================================================
-- Fluxo de caixa (metadados do financeiro React)
-- =============================================================================
ALTER TABLE fluxo_caixa
    ADD COLUMN IF NOT EXISTS metodo_pagamento VARCHAR(30)
        CHECK (metodo_pagamento IS NULL OR metodo_pagamento IN (
            'PIX', 'CARTAO_DEBITO', 'CARTAO_CREDITO', 'DINHEIRO'
        )),
    ADD COLUMN IF NOT EXISTS categoria VARCHAR(30)
        CHECK (categoria IS NULL OR categoria IN (
            'SERVICO', 'PRODUTO', 'COMISSAO', 'ALUGUEL', 'INSUMO', 'MARKETING', 'OUTROS'
        )),
    ADD COLUMN IF NOT EXISTS subtitulo VARCHAR(255),
    ADD COLUMN IF NOT EXISTS status_transacao VARCHAR(20)
        CHECK (status_transacao IS NULL OR status_transacao IN ('PAGO', 'PENDENTE', 'VENCIDO')),
    ADD COLUMN IF NOT EXISTS vencimento DATE,
    ADD COLUMN IF NOT EXISTS fornecedor VARCHAR(255),
    ADD COLUMN IF NOT EXISTS natureza VARCHAR(20)
        CHECK (natureza IS NULL OR natureza IN ('FIXO', 'VARIAVEL')),
    ADD COLUMN IF NOT EXISTS recorrente BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS frequencia_recorrencia VARCHAR(20)
        CHECK (frequencia_recorrencia IS NULL OR frequencia_recorrencia IN ('SEMANAL', 'MENSAL', 'ANUAL'));

CREATE INDEX IF NOT EXISTS idx_fluxo_caixa_categoria
    ON fluxo_caixa (estabelecimento_id, categoria);

-- =============================================================================
-- Contas globais de clientes (autoatendimento / login público)
-- =============================================================================
CREATE TABLE IF NOT EXISTS contas_cliente (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telefone      VARCHAR(20) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nome          VARCHAR(255) NOT NULL,
    criado_em     TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT contas_cliente_telefone_unique UNIQUE (telefone)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_contas_cliente_email_lower
    ON contas_cliente (LOWER(email));

CREATE INDEX IF NOT EXISTS idx_contas_cliente_telefone
    ON contas_cliente (telefone);

-- =============================================================================
-- Preferências do cliente por salão
-- =============================================================================
CREATE TABLE IF NOT EXISTS cliente_preferencias_salao (
    conta_cliente_id        UUID NOT NULL REFERENCES contas_cliente (id) ON DELETE CASCADE,
    estabelecimento_id      UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    profissional_favorito_id UUID REFERENCES profissionais (id) ON DELETE SET NULL,
    PRIMARY KEY (conta_cliente_id, estabelecimento_id)
);

CREATE INDEX IF NOT EXISTS idx_cliente_pref_estabelecimento
    ON cliente_preferencias_salao (estabelecimento_id);

-- =============================================================================
-- CRM: notas internas e galeria do cliente
-- =============================================================================
CREATE TABLE IF NOT EXISTS cliente_notas_internas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id  UUID NOT NULL,
    estabelecimento_id UUID NOT NULL,
    autor       VARCHAR(255) NOT NULL,
    texto       TEXT NOT NULL,
    criado_em   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cliente_notas_cliente
        FOREIGN KEY (estabelecimento_id, cliente_id)
        REFERENCES clientes (estabelecimento_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cliente_notas_cliente
    ON cliente_notas_internas (cliente_id);

CREATE TABLE IF NOT EXISTS cliente_galeria (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_id  UUID NOT NULL,
    estabelecimento_id UUID NOT NULL,
    url         TEXT NOT NULL,
    legenda     VARCHAR(500) NOT NULL DEFAULT '',
    data        DATE NOT NULL DEFAULT CURRENT_DATE,
    criado_em   TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_cliente_galeria_cliente
        FOREIGN KEY (estabelecimento_id, cliente_id)
        REFERENCES clientes (estabelecimento_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cliente_galeria_cliente
    ON cliente_galeria (cliente_id);

-- =============================================================================
-- Faturas SaaS (histórico de cobrança dos salões — Super Admin)
-- =============================================================================
CREATE TABLE IF NOT EXISTS faturas_saas (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    valor              NUMERIC(10, 2) NOT NULL CHECK (valor >= 0),
    data               DATE NOT NULL,
    status             VARCHAR(20) NOT NULL
        CHECK (status IN ('SUCESSO', 'PENDENTE', 'FALHA')),
    referencia_mes     CHAR(7) NOT NULL CHECK (referencia_mes ~ '^\d{4}-\d{2}$'),
    criado_em          TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_faturas_saas_estabelecimento_data
    ON faturas_saas (estabelecimento_id, data DESC);

CREATE INDEX IF NOT EXISTS idx_faturas_saas_referencia
    ON faturas_saas (referencia_mes);

-- =============================================================================
-- Fechamento do dia (nota da dona)
-- =============================================================================
CREATE TABLE IF NOT EXISTS fechamento_dia_notas (
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    data               DATE NOT NULL,
    texto              TEXT NOT NULL DEFAULT '',
    atualizado_em      TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (estabelecimento_id, data)
);

-- =============================================================================
-- Fila de espera: FK composta com tenant (integridade multi-tenant)
-- =============================================================================
ALTER TABLE fila_espera DROP CONSTRAINT IF EXISTS fk_fila_espera_prof_tenant;
ALTER TABLE fila_espera
    ADD CONSTRAINT fk_fila_espera_prof_tenant
        FOREIGN KEY (estabelecimento_id, profissional_id)
        REFERENCES profissionais (estabelecimento_id, id) ON DELETE CASCADE;
