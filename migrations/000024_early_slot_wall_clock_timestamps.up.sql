-- DEV-105: horários de agenda na antecipação alinhados ao TIMESTAMP de parede
-- de agendamentos (000001). Comparar TIMESTAMPTZ com TIMESTAMP usa o GUC
-- TimeZone da sessão; com America/Sao_Paulo o aceite falhava em silêncio
-- (appointment_no_longer_eligible) e EXTRACT/::time liam o slot deslocado.
-- Instantes operacionais (created_at, expira_em, etc.) permanecem TIMESTAMPTZ.
-- AT TIME ZONE 'UTC' porque o app grava via lib/pq com Location UTC.

ALTER TABLE rodadas_antecipacao
    ALTER COLUMN slot_inicio TYPE TIMESTAMP USING (slot_inicio AT TIME ZONE 'UTC'),
    ALTER COLUMN slot_fim TYPE TIMESTAMP USING (slot_fim AT TIME ZONE 'UTC');

ALTER TABLE ofertas_antecipacao
    ALTER COLUMN inicio_snapshot TYPE TIMESTAMP USING (inicio_snapshot AT TIME ZONE 'UTC'),
    ALTER COLUMN fim_snapshot TYPE TIMESTAMP USING (fim_snapshot AT TIME ZONE 'UTC');
