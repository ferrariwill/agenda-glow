DROP INDEX IF EXISTS idx_oferta_antecipacao_reenvio;

ALTER TABLE ofertas_antecipacao
    DROP COLUMN IF EXISTS proxima_tentativa_em;
