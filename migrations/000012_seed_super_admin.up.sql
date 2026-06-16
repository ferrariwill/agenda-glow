-- Seed do Super Admin (altere a senha após o primeiro login).
-- Senha inicial documentada: AgendaGlow@2026

INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, ativo)
VALUES (
    'ferrariwill@gmail.com',
    '$2a$10$nCM70BH8Af1d9SXwe4zms.kmxpM9FIr/VkwT5ZbkhUfcKa.z88ZDu',
    'SUPER_ADMIN',
    NULL,
    NULL,
    TRUE
)
ON CONFLICT (email) DO NOTHING;
