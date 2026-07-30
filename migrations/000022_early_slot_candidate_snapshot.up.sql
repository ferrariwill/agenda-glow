-- DEV-85: horário do candidato no instante da oferta.
-- Sem este retrato não há como distinguir "candidato intacto" de "candidato
-- reagendado para outro horário que por acaso ainda é elegível", e a oferta
-- antiga sobrescreveria o reagendamento novo.

ALTER TABLE ofertas_antecipacao
    ADD COLUMN IF NOT EXISTS agendamento_inicio_original TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS agendamento_fim_original TIMESTAMPTZ;
