-- AgendaGlow: cadastro de clientes e auditoria de origem do agendamento.

CREATE TABLE clientes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome           VARCHAR(255) NOT NULL,
    telefone       VARCHAR(20) NOT NULL UNIQUE,
    email          VARCHAR(255),
    data_cadastro  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_clientes_telefone ON clientes (telefone);

-- Migra clientes existentes a partir dos agendamentos legados (nome/telefone em texto).
INSERT INTO clientes (nome, telefone)
SELECT DISTINCT ON (cliente_telefone) cliente_nome, cliente_telefone
FROM agendamentos
ORDER BY cliente_telefone, data_hora_inicio ASC
ON CONFLICT (telefone) DO NOTHING;

ALTER TABLE agendamentos
    ADD COLUMN cliente_id UUID REFERENCES clientes (id),
    ADD COLUMN origem_agendamento VARCHAR(50);

UPDATE agendamentos a
SET cliente_id = c.id
FROM clientes c
WHERE c.telefone = a.cliente_telefone;

UPDATE agendamentos
SET origem_agendamento = 'INTERNO'
WHERE origem_agendamento IS NULL;

ALTER TABLE agendamentos
    ALTER COLUMN cliente_id SET NOT NULL,
    ALTER COLUMN origem_agendamento SET NOT NULL,
    ADD CONSTRAINT chk_origem_agendamento
        CHECK (origem_agendamento IN ('INTERNO', 'EXTERNO')),
    DROP COLUMN cliente_nome,
    DROP COLUMN cliente_telefone;

DROP INDEX IF EXISTS idx_agendamentos_cliente_telefone;

CREATE INDEX idx_agendamentos_cliente_id ON agendamentos (cliente_id);
CREATE INDEX idx_agendamentos_origem ON agendamentos (origem_agendamento);
