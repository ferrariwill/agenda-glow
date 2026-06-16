-- AgendaGlow: jornada de trabalho personalizada por profissional e dia da semana.

CREATE TABLE expedientes_profissionais (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profissional_id  UUID NOT NULL REFERENCES profissionais (id) ON DELETE CASCADE,
    dia_semana       INTEGER NOT NULL CHECK (dia_semana >= 0 AND dia_semana <= 6),
    horario_entrada  TIME NOT NULL,
    inicio_almoco    TIME,
    fim_almoco       TIME,
    horario_saida    TIME NOT NULL,
    CONSTRAINT expedientes_profissionais_dia_unico UNIQUE (profissional_id, dia_semana),
    CONSTRAINT chk_expedientes_horario_valido
        CHECK (horario_saida > horario_entrada),
    CONSTRAINT chk_expedientes_almoco_par
        CHECK (
            (inicio_almoco IS NULL AND fim_almoco IS NULL)
            OR (inicio_almoco IS NOT NULL AND fim_almoco IS NOT NULL AND fim_almoco > inicio_almoco)
        )
);

CREATE INDEX idx_expedientes_profissional_dia
    ON expedientes_profissionais (profissional_id, dia_semana);
