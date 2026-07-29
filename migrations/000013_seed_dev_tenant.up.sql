-- Tenant de desenvolvimento: Estúdio Glow Salto + usuários DONA e PROFISSIONAL.
-- Senha de todos os usuários dev: AgendaGlow@2026
-- Hash bcrypt: $2a$10$nCM70BH8Af1d9SXwe4zms.kmxpM9FIr/VkwT5ZbkhUfcKa.z88ZDu

INSERT INTO planos_saas (id, nome, preco_mensal, limite_profissionais, ativo)
VALUES (
    'a1000001-0001-4001-8001-000000000001',
    'Plano Solo Dev',
    97.00,
    1,
    TRUE
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO estabelecimentos (id, nome_comercial, slug, ativo)
VALUES (
    'a1000002-0002-4002-8002-000000000002',
    'Estúdio Glow Salto',
    'estudio-glow-salto',
    TRUE
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO assinaturas_estabelecimentos (estabelecimento_id, plano_id, status, data_vencimento)
VALUES (
    'a1000002-0002-4002-8002-000000000002',
    'a1000001-0001-4001-8001-000000000001',
    'ATIVO',
    (CURRENT_DATE + INTERVAL '12 months')::DATE
)
ON CONFLICT (estabelecimento_id) DO UPDATE SET
    plano_id = EXCLUDED.plano_id,
    status = 'ATIVO',
    data_vencimento = EXCLUDED.data_vencimento,
    atualizado_em = NOW();

INSERT INTO profissionais (id, estabelecimento_id, nome, especialidade, comissao_porcentagem, ativo, email)
VALUES (
    'a1000003-0003-4003-8003-000000000003',
    'a1000002-0002-4002-8002-000000000002',
    'Cláudia - Manicure',
    'Manicure',
    40.00,
    TRUE,
    'claudia@glow.local'
)
ON CONFLICT (estabelecimento_id, id) DO NOTHING;

INSERT INTO servicos (id, estabelecimento_id, nome, preco_base, duracao_base_minutos, ativo)
VALUES (
    'a1000004-0004-4004-8004-000000000004',
    'a1000002-0002-4002-8002-000000000002',
    'Fazer Unhas',
    50.00,
    45,
    TRUE
)
ON CONFLICT (estabelecimento_id, id) DO NOTHING;

INSERT INTO servico_adicionais (id, servico_id, nome, preco_adicional, duracao_adicional_minutos)
VALUES (
    'a1000005-0005-4005-8005-000000000005',
    'a1000004-0004-4004-8004-000000000004',
    'Alongamento em Gel',
    80.00,
    60
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, ativo)
VALUES
    (
        'dona@glow.local',
        '$2a$10$nCM70BH8Af1d9SXwe4zms.kmxpM9FIr/VkwT5ZbkhUfcKa.z88ZDu',
        'DONA',
        'a1000002-0002-4002-8002-000000000002',
        NULL,
        TRUE
    ),
    (
        'claudia@glow.local',
        '$2a$10$nCM70BH8Af1d9SXwe4zms.kmxpM9FIr/VkwT5ZbkhUfcKa.z88ZDu',
        'PROFISSIONAL',
        'a1000002-0002-4002-8002-000000000002',
        'a1000003-0003-4003-8003-000000000003',
        TRUE
    )
ON CONFLICT (email) DO NOTHING;
