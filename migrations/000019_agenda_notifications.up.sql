-- DEV-82: ciclo anti-no-show, gestão pública e fila transacional de notificações.

CREATE TABLE configuracoes_notificacoes_agenda (
    estabelecimento_id UUID PRIMARY KEY
        REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    lembretes_ativos BOOLEAN NOT NULL DEFAULT TRUE,
    antecedencia_confirmacao_horas INTEGER NOT NULL DEFAULT 24
        CHECK (antecedencia_confirmacao_horas BETWEEN 1 AND 168),
    antecedencia_lembrete_horas INTEGER NOT NULL DEFAULT 1
        CHECK (antecedencia_lembrete_horas BETWEEN 1 AND 48),
    janela_minima_cancelamento_horas INTEGER NOT NULL DEFAULT 2
        CHECK (janela_minima_cancelamento_horas BETWEEN 0 AND 168),
    motivo_cancelamento_obrigatorio BOOLEAN NOT NULL DEFAULT FALSE,
    template_confirmacao TEXT NOT NULL DEFAULT
        'Olá! Confirme seu horário de {{servico}} em {{nome_salao}}: {{data_hora}}. Gestão: {{link_gestao}}',
    template_lembrete TEXT NOT NULL DEFAULT
        'Lembrete: seu horário de {{servico}} com {{profissional}} é {{data_hora}} em {{nome_salao}}. Gestão: {{link_gestao}}',
    criado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_config_notificacao_template_confirmacao
        CHECK (length(template_confirmacao) BETWEEN 1 AND 2000
               AND template_confirmacao !~ '[<>]'),
    CONSTRAINT chk_config_notificacao_template_lembrete
        CHECK (length(template_lembrete) BETWEEN 1 AND 2000
               AND template_lembrete !~ '[<>]')
);

INSERT INTO configuracoes_notificacoes_agenda (estabelecimento_id)
SELECT id FROM estabelecimentos
ON CONFLICT (estabelecimento_id) DO NOTHING;

ALTER TABLE estabelecimentos
    ADD COLUMN whatsapp_phone_number VARCHAR(30);

ALTER TABLE agendamentos
    ADD COLUMN gestao_token UUID DEFAULT gen_random_uuid(),
    ADD COLUMN gestao_token_expires_at TIMESTAMP,
    ADD COLUMN confirmacao_cliente VARCHAR(30) NOT NULL DEFAULT 'PENDENTE',
    ADD COLUMN motivo_cancelamento_cliente TEXT;

CREATE FUNCTION set_agendamento_gestao_token_expiry()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.gestao_token_expires_at IS NULL THEN
        NEW.gestao_token_expires_at := NEW.data_hora_fim + INTERVAL '24 hours';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_agendamento_gestao_token_expiry
BEFORE INSERT ON agendamentos
FOR EACH ROW EXECUTE FUNCTION set_agendamento_gestao_token_expiry();

UPDATE agendamentos
SET gestao_token = COALESCE(gestao_token, gen_random_uuid()),
    gestao_token_expires_at = COALESCE(
        gestao_token_expires_at,
        data_hora_fim + INTERVAL '24 hours'
    );

ALTER TABLE agendamentos
    ALTER COLUMN gestao_token SET NOT NULL,
    ALTER COLUMN gestao_token SET DEFAULT gen_random_uuid(),
    ALTER COLUMN gestao_token_expires_at SET NOT NULL,
    ADD CONSTRAINT agendamentos_gestao_token_unique UNIQUE (gestao_token),
    ADD CONSTRAINT agendamentos_estabelecimento_id_id_unique
        UNIQUE (estabelecimento_id, id),
    ADD CONSTRAINT chk_agendamentos_confirmacao_cliente
        CHECK (confirmacao_cliente IN (
            'PENDENTE', 'CONFIRMADO_CLIENTE', 'CANCELADO_CLIENTE'
        ));

CREATE INDEX idx_agendamentos_gestao_token
    ON agendamentos (gestao_token);

CREATE TABLE agendamento_notificacoes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL,
    agendamento_id UUID NOT NULL,
    tipo VARCHAR(40) NOT NULL,
    canal VARCHAR(20) NOT NULL DEFAULT 'WHATSAPP',
    status_envio VARCHAR(20) NOT NULL DEFAULT 'PENDENTE',
    tentativas INTEGER NOT NULL DEFAULT 0 CHECK (tentativas >= 0),
    proxima_tentativa_em TIMESTAMP NOT NULL DEFAULT NOW(),
    enviado_em TIMESTAMP,
    external_message_id VARCHAR(255),
    payload_resumido JSONB NOT NULL DEFAULT '{}'::jsonb,
    ultimo_erro TEXT,
    criado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_agendamento_notificacoes_estabelecimento
        FOREIGN KEY (estabelecimento_id)
        REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    CONSTRAINT fk_agendamento_notificacoes_tenant
        FOREIGN KEY (estabelecimento_id, agendamento_id)
        REFERENCES agendamentos (estabelecimento_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_agendamento_notificacoes_tipo
        CHECK (tipo IN (
            'CONFIRMACAO_RESERVA', 'PEDIDO_CONFIRMACAO', 'LEMBRETE',
            'CANCELAMENTO_CLIENTE', 'CANCELAMENTO_CONFIRMADO'
        )),
    CONSTRAINT chk_agendamento_notificacoes_canal
        CHECK (canal = 'WHATSAPP'),
    -- REGISTRADO: linha de auditoria de evento recebido do cliente; nunca é enviada.
    CONSTRAINT chk_agendamento_notificacoes_status
        CHECK (status_envio IN ('PENDENTE', 'ENVIANDO', 'ENVIADO', 'FALHOU', 'REGISTRADO')),
    UNIQUE (estabelecimento_id, agendamento_id, tipo)
);

CREATE INDEX idx_agendamento_notificacoes_pendentes
    ON agendamento_notificacoes (proxima_tentativa_em, criado_em)
    WHERE status_envio IN ('PENDENTE', 'FALHOU');

CREATE INDEX idx_agendamento_notificacoes_agendamento
    ON agendamento_notificacoes (estabelecimento_id, agendamento_id);
