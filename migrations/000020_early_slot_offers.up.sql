-- DEV-85: fila automática e exclusiva de antecipação.

ALTER TABLE agendamentos
    ADD COLUMN IF NOT EXISTS aceita_adiantar_em TIMESTAMPTZ;

UPDATE agendamentos
SET aceita_adiantar_em = NOW()
WHERE aceita_adiantar = TRUE
  AND aceita_adiantar_em IS NULL;

CREATE OR REPLACE FUNCTION manter_aceita_adiantar_em()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.aceita_adiantar = TRUE
       AND (TG_OP = 'INSERT' OR OLD.aceita_adiantar = FALSE OR OLD.aceita_adiantar IS NULL) THEN
        NEW.aceita_adiantar_em = NOW();
    ELSIF NEW.aceita_adiantar = FALSE THEN
        NEW.aceita_adiantar_em = NULL;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_manter_aceita_adiantar_em ON agendamentos;
CREATE TRIGGER trg_manter_aceita_adiantar_em
    BEFORE INSERT OR UPDATE OF aceita_adiantar ON agendamentos
    FOR EACH ROW EXECUTE FUNCTION manter_aceita_adiantar_em();

CREATE INDEX IF NOT EXISTS idx_agendamentos_antecipacao_candidatos
    ON agendamentos (estabelecimento_id, profissional_id, data_hora_inicio)
    WHERE aceita_adiantar = TRUE
      AND status IN ('AGENDADO', 'CONFIRMADO');

CREATE TABLE rodadas_antecipacao (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    profissional_id UUID NOT NULL REFERENCES profissionais(id),
    slot_inicio TIMESTAMPTZ NOT NULL,
    slot_fim TIMESTAMPTZ NOT NULL,
    agendamento_cancelado_id UUID NOT NULL REFERENCES agendamentos(id),
    status VARCHAR(16) NOT NULL DEFAULT 'ATIVA'
        CHECK (status IN ('ATIVA', 'PREENCHIDA', 'ESGOTADA', 'CANCELADA')),
    motivo_encerramento VARCHAR(40)
        CHECK (motivo_encerramento IS NULL OR motivo_encerramento IN ('whatsapp_indisponivel')),
    candidato_atual_agendamento_id UUID REFERENCES agendamentos(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    UNIQUE (estabelecimento_id, agendamento_cancelado_id),
    CHECK (slot_fim > slot_inicio)
);

CREATE UNIQUE INDEX idx_rodada_antecipacao_slot_ativa
    ON rodadas_antecipacao (estabelecimento_id, profissional_id, slot_inicio)
    WHERE status = 'ATIVA';

CREATE TABLE ofertas_antecipacao (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    rodada_id UUID NOT NULL REFERENCES rodadas_antecipacao(id) ON DELETE CASCADE,
    agendamento_candidato_id UUID NOT NULL REFERENCES agendamentos(id),
    cliente_id UUID REFERENCES clientes(id),
    posicao INTEGER NOT NULL CHECK (posicao > 0),
    status VARCHAR(16) NOT NULL DEFAULT 'PENDENTE'
        CHECK (status IN ('PENDENTE', 'ACEITA', 'RECUSADA', 'EXPIRADA', 'INVALIDADA', 'FALHA_ENVIO')),
    token_hash BYTEA NOT NULL,
    enviada_em TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expira_em TIMESTAMPTZ NOT NULL,
    respondida_em TIMESTAMPTZ,
    origem_resposta VARCHAR(16) CHECK (origem_resposta IN ('WHATSAPP', 'WEB')),
    external_message_id TEXT,
    tentativas_envio INTEGER NOT NULL DEFAULT 0,
    UNIQUE (rodada_id, agendamento_candidato_id),
    UNIQUE (token_hash)
);

CREATE UNIQUE INDEX idx_oferta_antecipacao_pendente_rodada
    ON ofertas_antecipacao (rodada_id)
    WHERE status = 'PENDENTE';

CREATE INDEX idx_oferta_antecipacao_expiracao
    ON ofertas_antecipacao (expira_em)
    WHERE status = 'PENDENTE';

CREATE INDEX idx_oferta_antecipacao_candidato_pendente
    ON ofertas_antecipacao (estabelecimento_id, agendamento_candidato_id)
    WHERE status = 'PENDENTE';

CREATE TABLE antecipacao_auditoria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos(id),
    rodada_id UUID NOT NULL REFERENCES rodadas_antecipacao(id) ON DELETE CASCADE,
    oferta_id UUID REFERENCES ofertas_antecipacao(id),
    evento VARCHAR(40) NOT NULL,
    horario_anterior_inicio TIMESTAMPTZ,
    horario_anterior_fim TIMESTAMPTZ,
    horario_novo_inicio TIMESTAMPTZ,
    horario_novo_fim TIMESTAMPTZ,
    origem VARCHAR(16),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
