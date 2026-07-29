DROP INDEX IF EXISTS idx_estabelecimentos_whatsapp_enabled;

ALTER TABLE estabelecimentos
    DROP COLUMN IF EXISTS whatsapp_enabled;

