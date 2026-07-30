-- DEV-85: segunda tentativa de envio da oferta, agendada e durável.
-- Sem esta coluna não há como distinguir uma oferta entregue de uma que falhou
-- no Gateway, e a retentativa se perderia num restart da API.

ALTER TABLE ofertas_antecipacao
    ADD COLUMN IF NOT EXISTS proxima_tentativa_em TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_oferta_antecipacao_reenvio
    ON ofertas_antecipacao (proxima_tentativa_em)
    WHERE status = 'PENDENTE' AND proxima_tentativa_em IS NOT NULL;
