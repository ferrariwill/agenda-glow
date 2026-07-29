ALTER TABLE estabelecimentos
    DROP CONSTRAINT IF EXISTS chk_estabelecimentos_whatsapp_status;

DROP INDEX IF EXISTS idx_estabelecimentos_whatsapp_status;

ALTER TABLE estabelecimentos
    DROP COLUMN IF EXISTS whatsapp_connected_at,
    DROP COLUMN IF EXISTS whatsapp_phone_number_id,
    DROP COLUMN IF EXISTS whatsapp_waba_id,
    DROP COLUMN IF EXISTS whatsapp_status;
