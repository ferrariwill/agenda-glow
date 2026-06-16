-- AgendaGlow: identidade visual do estabelecimento (logo na página pública).

ALTER TABLE estabelecimentos
    ADD COLUMN logo_url TEXT;
