-- Integração WhatsApp Oficial (Embedded Signup via Gateway).

ALTER TABLE estabelecimentos
    ADD COLUMN IF NOT EXISTS whatsapp_status VARCHAR(20) NOT NULL DEFAULT 'DESCONECTADO',
    ADD COLUMN IF NOT EXISTS whatsapp_waba_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS whatsapp_phone_number_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS whatsapp_connected_at TIMESTAMP;

ALTER TABLE estabelecimentos
    DROP CONSTRAINT IF EXISTS chk_estabelecimentos_whatsapp_status;

ALTER TABLE estabelecimentos
    ADD CONSTRAINT chk_estabelecimentos_whatsapp_status
        CHECK (whatsapp_status IN ('DESCONECTADO', 'PENDENTE', 'CONECTADO'));

CREATE INDEX IF NOT EXISTS idx_estabelecimentos_whatsapp_status
    ON estabelecimentos (whatsapp_status);
