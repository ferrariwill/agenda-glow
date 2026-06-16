-- AgendaGlow: Clube de Assinatura Recorrente — planos mensais e saldo de visitas.

CREATE TABLE planos_assinatura (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome                        VARCHAR(255) NOT NULL,
    preco_mensal                NUMERIC(10, 2) NOT NULL CHECK (preco_mensal >= 0),
    total_visitas_mes           INTEGER NOT NULL CHECK (total_visitas_mes > 0),
    valor_repasse_profissional  NUMERIC(10, 2) NOT NULL DEFAULT 0.00
        CHECK (valor_repasse_profissional >= 0),
    ativo                       BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE clientes_assinantes (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cliente_nome          VARCHAR(255) NOT NULL,
    cliente_telefone      VARCHAR(20) NOT NULL UNIQUE,
    plano_id              UUID NOT NULL REFERENCES planos_assinatura (id),
    status                VARCHAR(50) NOT NULL DEFAULT 'ATIVO'
        CHECK (status IN ('ATIVO', 'INADIMPLENTE', 'CANCELADO')),
    visitas_restantes     INTEGER NOT NULL CHECK (visitas_restantes >= 0),
    data_proxima_cobranca DATE NOT NULL
);

CREATE INDEX idx_planos_assinatura_ativo
    ON planos_assinatura (ativo)
    WHERE ativo = TRUE;

CREATE INDEX idx_clientes_assinantes_telefone_ativo
    ON clientes_assinantes (cliente_telefone)
    WHERE status = 'ATIVO';

CREATE INDEX idx_clientes_assinantes_plano
    ON clientes_assinantes (plano_id);
