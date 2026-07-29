DELETE FROM users WHERE email = 'secretaria@glow.local';

DROP TABLE IF EXISTS insumos;
DROP TABLE IF EXISTS fila_espera;

ALTER TABLE profissionais DROP COLUMN IF EXISTS pendente_aprovacao;

ALTER TABLE agendamentos DROP COLUMN IF EXISTS cobrado_em;
ALTER TABLE agendamentos DROP COLUMN IF EXISTS metodo_pagamento;
ALTER TABLE agendamentos DROP COLUMN IF EXISTS valor_cobrado;
ALTER TABLE agendamentos DROP COLUMN IF EXISTS aceita_adiantar;
ALTER TABLE agendamentos DROP COLUMN IF EXISTS minutos_invadidos;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_escopo;
ALTER TABLE users ADD CONSTRAINT users_role_escopo CHECK (
    (role = 'SUPER_ADMIN' AND estabelecimento_id IS NULL AND profissional_id IS NULL)
    OR (role = 'DONA' AND estabelecimento_id IS NOT NULL AND profissional_id IS NULL)
    OR (role = 'PROFISSIONAL' AND estabelecimento_id IS NOT NULL AND profissional_id IS NOT NULL)
);

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('SUPER_ADMIN', 'DONA', 'PROFISSIONAL'));
