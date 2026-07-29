-- Evita planos SaaS duplicados por nome (case-insensitive).

WITH canonical AS (
    SELECT DISTINCT ON (LOWER(TRIM(nome)))
        id AS keep_id,
        LOWER(TRIM(nome)) AS nome_key
    FROM planos_saas
    ORDER BY LOWER(TRIM(nome)), id
),
dups AS (
    SELECT p.id AS dup_id, c.keep_id
    FROM planos_saas p
    INNER JOIN canonical c
        ON LOWER(TRIM(p.nome)) = c.nome_key
       AND p.id <> c.keep_id
)
UPDATE assinaturas_estabelecimentos ae
SET plano_id = d.keep_id,
    atualizado_em = NOW()
FROM dups d
WHERE ae.plano_id = d.dup_id;

WITH canonical AS (
    SELECT DISTINCT ON (LOWER(TRIM(nome)))
        id AS keep_id,
        LOWER(TRIM(nome)) AS nome_key
    FROM planos_saas
    ORDER BY LOWER(TRIM(nome)), id
)
DELETE FROM planos_saas p
USING canonical c
WHERE LOWER(TRIM(p.nome)) = c.nome_key
  AND p.id <> c.keep_id;

CREATE UNIQUE INDEX idx_planos_saas_nome_unique ON planos_saas (LOWER(TRIM(nome)));
