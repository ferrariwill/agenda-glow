-- DEV-85: o link já entregue continua valendo depois de uma retentativa.
-- A retentativa precisa reemitir o token porque só o hash é persistido; sem
-- guardar o hash anterior, a cliente que clica na primeira mensagem recebe
-- offer_not_found dentro dos 5 minutos exclusivos dela.

ALTER TABLE ofertas_antecipacao
    ADD COLUMN IF NOT EXISTS token_hash_anterior BYTEA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_oferta_antecipacao_token_anterior
    ON ofertas_antecipacao (token_hash_anterior)
    WHERE token_hash_anterior IS NOT NULL;
