DROP TABLE IF EXISTS antecipacao_auditoria;
DROP TABLE IF EXISTS ofertas_antecipacao;
DROP TABLE IF EXISTS rodadas_antecipacao;
DROP INDEX IF EXISTS idx_agendamentos_antecipacao_candidatos;
DROP TRIGGER IF EXISTS trg_manter_aceita_adiantar_em ON agendamentos;
DROP FUNCTION IF EXISTS manter_aceita_adiantar_em();
ALTER TABLE agendamentos DROP COLUMN IF EXISTS aceita_adiantar_em;
