ALTER TABLE profissionais
    DROP CONSTRAINT IF EXISTS fk_profissionais_especialidade_tenant;

ALTER TABLE profissionais
    ADD COLUMN especialidade VARCHAR(100);

UPDATE profissionais p
SET especialidade = e.nome
FROM especialidades e
WHERE e.id = p.especialidade_id;

ALTER TABLE profissionais
    ALTER COLUMN especialidade SET NOT NULL,
    DROP COLUMN IF EXISTS especialidade_id;

DROP TABLE IF EXISTS especialidades;
