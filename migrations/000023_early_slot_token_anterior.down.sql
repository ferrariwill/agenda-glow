DROP INDEX IF EXISTS idx_oferta_antecipacao_token_anterior;

ALTER TABLE ofertas_antecipacao
    DROP COLUMN IF EXISTS token_hash_anterior;
