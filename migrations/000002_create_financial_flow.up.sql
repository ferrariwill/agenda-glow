-- AgendaGlow: fluxo de caixa — receitas, custos fixos e comissões de profissionais.

CREATE TABLE fluxo_caixa (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tipo            VARCHAR(50) NOT NULL
        CHECK (tipo IN ('ENTRADA', 'CUSTO_FIXO', 'CUSTO_VARIAVEL')),
    descricao       VARCHAR(255) NOT NULL,
    valor           NUMERIC(10, 2) NOT NULL CHECK (valor >= 0),
    profissional_id UUID REFERENCES profissionais (id),
    data_transacao  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fluxo_caixa_tipo ON fluxo_caixa (tipo);
CREATE INDEX idx_fluxo_caixa_profissional ON fluxo_caixa (profissional_id);
CREATE INDEX idx_fluxo_caixa_data ON fluxo_caixa (data_transacao);
