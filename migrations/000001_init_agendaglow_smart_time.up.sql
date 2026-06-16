-- AgendaGlow: migração inicial com tempo dinâmico por variações de serviço.
-- Suporta bloqueio preciso de agenda (ex.: cabelo longo, gel, nail art somam minutos ao serviço base).

-- Profissionais do estúdio.
CREATE TABLE profissionais (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome                 VARCHAR(255) NOT NULL,
    especialidade        VARCHAR(100) NOT NULL,
    comissao_porcentagem NUMERIC(5, 2) NOT NULL DEFAULT 40.00
        CHECK (comissao_porcentagem >= 0 AND comissao_porcentagem <= 100),
    ativo                BOOLEAN NOT NULL DEFAULT TRUE
);

-- Serviços base (procedimento principal com preço e duração padrão).
CREATE TABLE servicos (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome                 VARCHAR(255) NOT NULL,
    preco_base           NUMERIC(10, 2) NOT NULL CHECK (preco_base >= 0),
    duracao_base_minutos INTEGER NOT NULL CHECK (duracao_base_minutos > 0),
    ativo                BOOLEAN NOT NULL DEFAULT TRUE
);

-- Variações/adicionais que incrementam tempo e valor do serviço base.
CREATE TABLE servico_adicionais (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    servico_id                 UUID NOT NULL REFERENCES servicos (id) ON DELETE CASCADE,
    nome                       VARCHAR(255) NOT NULL,
    preco_adicional            NUMERIC(10, 2) NOT NULL DEFAULT 0.00 CHECK (preco_adicional >= 0),
    duracao_adicional_minutos  INTEGER NOT NULL DEFAULT 0 CHECK (duracao_adicional_minutos >= 0),
    UNIQUE (servico_id, id)
);

-- Agendamentos com bloqueio de horário calculado dinamicamente (base + adicionais).
-- data_hora_fim é preenchido na aplicação Go: início + duracao_base + Σ duracao_adicional.
CREATE TABLE agendamentos (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_nome         VARCHAR(255) NOT NULL,
    cliente_telefone     VARCHAR(20) NOT NULL,
    profissional_id      UUID NOT NULL REFERENCES profissionais (id),
    servico_id           UUID NOT NULL REFERENCES servicos (id),
    data_hora_inicio     TIMESTAMP NOT NULL,
    data_hora_fim        TIMESTAMP NOT NULL,
    status               VARCHAR(50) NOT NULL DEFAULT 'AGENDADO'
        CHECK (status IN ('AGENDADO', 'CONFIRMADO', 'CONCLUIDO', 'CANCELADO')),
    via_clube_assinatura BOOLEAN NOT NULL DEFAULT FALSE,
    CHECK (data_hora_fim > data_hora_inicio)
);

-- Adicionais escolhidos pela cliente no momento do agendamento (N:N).
CREATE TABLE agendamento_adicionais (
    agendamento_id UUID NOT NULL REFERENCES agendamentos (id) ON DELETE CASCADE,
    adicional_id   UUID NOT NULL REFERENCES servico_adicionais (id),
    PRIMARY KEY (agendamento_id, adicional_id)
);

-- Índices para consulta de agenda e detecção de colisão de horários.
CREATE INDEX idx_servico_adicionais_servico_id
    ON servico_adicionais (servico_id);

CREATE INDEX idx_agendamentos_profissional_inicio
    ON agendamentos (profissional_id, data_hora_inicio);

CREATE INDEX idx_agendamentos_profissional_intervalo
    ON agendamentos (profissional_id, data_hora_inicio, data_hora_fim)
    WHERE status NOT IN ('CANCELADO', 'CONCLUIDO');

CREATE INDEX idx_agendamentos_cliente_telefone
    ON agendamentos (cliente_telefone);

-- Garante que cada adicional vinculado pertence ao mesmo serviço do agendamento.
CREATE OR REPLACE FUNCTION validar_adicional_do_agendamento()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM agendamentos a
          JOIN servico_adicionais sa ON sa.id = NEW.adicional_id
         WHERE a.id = NEW.agendamento_id
           AND sa.servico_id = a.servico_id
    ) THEN
        RAISE EXCEPTION
            'O adicional % não pertence ao serviço do agendamento %',
            NEW.adicional_id,
            NEW.agendamento_id;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_validar_adicional_do_agendamento
    BEFORE INSERT OR UPDATE OF agendamento_id, adicional_id
    ON agendamento_adicionais
    FOR EACH ROW
    EXECUTE FUNCTION validar_adicional_do_agendamento();
