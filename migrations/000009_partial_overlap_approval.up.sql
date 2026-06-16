-- AgendaGlow: aprovação de encaixe por sobreposição parcial de término.

ALTER TABLE agendamentos DROP CONSTRAINT IF EXISTS agendamentos_status_check;

ALTER TABLE agendamentos
    ADD CONSTRAINT agendamentos_status_check
        CHECK (status IN ('AGENDADO', 'CONFIRMADO', 'CONCLUIDO', 'CANCELADO', 'EM_APROVACAO'));

ALTER TABLE profissionais
    ADD COLUMN IF NOT EXISTS email VARCHAR(255);

CREATE INDEX idx_agendamentos_em_aprovacao
    ON agendamentos (profissional_id, data_hora_inicio)
    WHERE status = 'EM_APROVACAO';
