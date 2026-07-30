DROP TABLE IF EXISTS agendamento_notificacoes;

DROP TRIGGER IF EXISTS trg_agendamento_gestao_token_expiry ON agendamentos;
DROP FUNCTION IF EXISTS set_agendamento_gestao_token_expiry();

ALTER TABLE agendamentos
    DROP CONSTRAINT IF EXISTS chk_agendamentos_confirmacao_cliente,
    DROP CONSTRAINT IF EXISTS agendamentos_estabelecimento_id_id_unique,
    DROP CONSTRAINT IF EXISTS agendamentos_gestao_token_unique,
    DROP COLUMN IF EXISTS motivo_cancelamento_cliente,
    DROP COLUMN IF EXISTS confirmacao_cliente,
    DROP COLUMN IF EXISTS gestao_token_expires_at,
    DROP COLUMN IF EXISTS gestao_token;

DROP TABLE IF EXISTS configuracoes_notificacoes_agenda;

ALTER TABLE estabelecimentos
    DROP COLUMN IF EXISTS whatsapp_phone_number;
