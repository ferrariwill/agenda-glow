-- DEV-199: conteúdo por embalagem para estoque em unidade base (volume/peso/contagem total).
-- quantidade permanece o estoque canônico na unidade base; conteudo_por_embalagem
-- permite cadastro por embalagens físicas (ex.: 10 × 50 ml → quantidade = 500).

ALTER TABLE insumos
    ADD COLUMN IF NOT EXISTS conteudo_por_embalagem NUMERIC(12, 3) NOT NULL DEFAULT 1
        CHECK (conteudo_por_embalagem > 0);

UPDATE insumos
SET conteudo_por_embalagem = 1
WHERE conteudo_por_embalagem IS NULL;
