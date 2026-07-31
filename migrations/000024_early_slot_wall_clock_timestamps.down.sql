-- Reverte DEV-105: volta os horários de agenda da antecipação para TIMESTAMPTZ.
-- Interpreta os dígitos de parede como UTC (o mesmo eixo usado no UP).

ALTER TABLE rodadas_antecipacao
    ALTER COLUMN slot_inicio TYPE TIMESTAMPTZ USING slot_inicio AT TIME ZONE 'UTC',
    ALTER COLUMN slot_fim TYPE TIMESTAMPTZ USING slot_fim AT TIME ZONE 'UTC';

ALTER TABLE ofertas_antecipacao
    ALTER COLUMN inicio_snapshot TYPE TIMESTAMPTZ USING inicio_snapshot AT TIME ZONE 'UTC',
    ALTER COLUMN fim_snapshot TYPE TIMESTAMPTZ USING fim_snapshot AT TIME ZONE 'UTC';
