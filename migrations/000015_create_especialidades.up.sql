-- Especialidades / cargos cadastrados por estabelecimento (selecionados na equipe).

CREATE TABLE especialidades (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    nome               VARCHAR(100) NOT NULL,
    ativo              BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (estabelecimento_id, id)
);

CREATE UNIQUE INDEX idx_especialidades_estab_nome_unique
    ON especialidades (estabelecimento_id, LOWER(TRIM(nome)));

CREATE INDEX idx_especialidades_estabelecimento ON especialidades (estabelecimento_id);

INSERT INTO especialidades (estabelecimento_id, nome)
SELECT DISTINCT ON (p.estabelecimento_id, LOWER(TRIM(p.especialidade)))
    p.estabelecimento_id,
    TRIM(p.especialidade)
FROM profissionais p
WHERE TRIM(p.especialidade) <> ''
ORDER BY p.estabelecimento_id, LOWER(TRIM(p.especialidade)), TRIM(p.especialidade);

ALTER TABLE profissionais
    ADD COLUMN especialidade_id UUID REFERENCES especialidades (id);

UPDATE profissionais p
SET especialidade_id = e.id
FROM especialidades e
WHERE e.estabelecimento_id = p.estabelecimento_id
  AND LOWER(TRIM(e.nome)) = LOWER(TRIM(p.especialidade));

INSERT INTO especialidades (estabelecimento_id, nome)
SELECT e.id, 'Geral'
FROM estabelecimentos e
WHERE EXISTS (
    SELECT 1 FROM profissionais p
    WHERE p.estabelecimento_id = e.id AND p.especialidade_id IS NULL
);

UPDATE profissionais p
SET especialidade_id = e.id
FROM especialidades e
WHERE p.especialidade_id IS NULL
  AND e.estabelecimento_id = p.estabelecimento_id
  AND e.nome = 'Geral';

ALTER TABLE profissionais
    ALTER COLUMN especialidade_id SET NOT NULL,
    DROP COLUMN especialidade,
    ADD CONSTRAINT fk_profissionais_especialidade_tenant
        FOREIGN KEY (estabelecimento_id, especialidade_id)
        REFERENCES especialidades (estabelecimento_id, id);

CREATE INDEX idx_profissionais_especialidade ON profissionais (especialidade_id);
