-- AgendaGlow: controle de quitação de comissões de profissionais parceiras.

ALTER TABLE fluxo_caixa
    ADD COLUMN status_pagamento VARCHAR(50) DEFAULT NULL
        CHECK (status_pagamento IS NULL OR status_pagamento IN ('PENDENTE', 'PAGO'));

UPDATE fluxo_caixa
SET status_pagamento = 'PENDENTE'
WHERE tipo = 'CUSTO_VARIAVEL';

CREATE INDEX idx_fluxo_caixa_status_pagamento
    ON fluxo_caixa (estabelecimento_id, profissional_id, status_pagamento)
    WHERE tipo = 'CUSTO_VARIAVEL';
