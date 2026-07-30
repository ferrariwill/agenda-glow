-- Trava administrativa: só o SUPER_ADMIN libera o recurso WhatsApp por salão.

ALTER TABLE estabelecimentos
    ADD COLUMN IF NOT EXISTS whatsapp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Backfill: salão já conectado não pode perder o recurso ao subir a migration.
UPDATE estabelecimentos
SET whatsapp_enabled = TRUE
WHERE whatsapp_status = 'CONECTADO';

CREATE INDEX IF NOT EXISTS idx_estabelecimentos_whatsapp_enabled
    ON estabelecimentos (whatsapp_enabled)
    WHERE whatsapp_enabled = TRUE;

