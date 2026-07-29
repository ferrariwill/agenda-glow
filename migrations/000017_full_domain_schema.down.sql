ALTER TABLE fila_espera DROP CONSTRAINT IF EXISTS fk_fila_espera_prof_tenant;

DROP TABLE IF EXISTS fechamento_dia_notas;
DROP TABLE IF EXISTS faturas_saas;
DROP TABLE IF EXISTS cliente_galeria;
DROP TABLE IF EXISTS cliente_notas_internas;
DROP TABLE IF EXISTS cliente_preferencias_salao;
DROP TABLE IF EXISTS contas_cliente;

ALTER TABLE fluxo_caixa
    DROP COLUMN IF EXISTS frequencia_recorrencia,
    DROP COLUMN IF EXISTS recorrente,
    DROP COLUMN IF EXISTS natureza,
    DROP COLUMN IF EXISTS fornecedor,
    DROP COLUMN IF EXISTS vencimento,
    DROP COLUMN IF EXISTS status_transacao,
    DROP COLUMN IF EXISTS subtitulo,
    DROP COLUMN IF EXISTS categoria,
    DROP COLUMN IF EXISTS metodo_pagamento;

DROP TABLE IF EXISTS agendamento_servicos;

ALTER TABLE agendamentos DROP COLUMN IF EXISTS observacoes;

DROP TABLE IF EXISTS servico_insumos;

ALTER TABLE insumos DROP CONSTRAINT IF EXISTS insumos_estabelecimento_id_id_key;

DROP TABLE IF EXISTS servico_profissionais;

ALTER TABLE servicos DROP CONSTRAINT IF EXISTS fk_servicos_categoria_tenant;
ALTER TABLE servicos
    DROP COLUMN IF EXISTS permitir_agendamento_online,
    DROP COLUMN IF EXISTS exibir_catalogo_publico,
    DROP COLUMN IF EXISTS categoria_id,
    DROP COLUMN IF EXISTS descricao,
    DROP COLUMN IF EXISTS imagem_url;

DROP TABLE IF EXISTS categorias_servicos;

ALTER TABLE profissionais
    DROP COLUMN IF EXISTS eh_dona,
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS valor_fixo_atendimento,
    DROP COLUMN IF EXISTS modelo_pagamento,
    DROP COLUMN IF EXISTS data_nascimento,
    DROP COLUMN IF EXISTS data_contratacao,
    DROP COLUMN IF EXISTS portfolio_url,
    DROP COLUMN IF EXISTS biografia,
    DROP COLUMN IF EXISTS telefone,
    DROP COLUMN IF EXISTS foto_url;

ALTER TABLE clientes
    DROP COLUMN IF EXISTS preferencias,
    DROP COLUMN IF EXISTS observacoes_medicas,
    DROP COLUMN IF EXISTS alergias,
    DROP COLUMN IF EXISTS foto_url,
    DROP COLUMN IF EXISTS data_nascimento,
    DROP COLUMN IF EXISTS ativo;

ALTER TABLE users
    DROP COLUMN IF EXISTS telefone,
    DROP COLUMN IF EXISTS nome;

ALTER TABLE estabelecimentos
    DROP COLUMN IF EXISTS dona_atua_como_profissional,
    DROP COLUMN IF EXISTS dona_email,
    DROP COLUMN IF EXISTS dona_nome,
    DROP COLUMN IF EXISTS uf,
    DROP COLUMN IF EXISTS cidade,
    DROP COLUMN IF EXISTS logradouro,
    DROP COLUMN IF EXISTS cep,
    DROP COLUMN IF EXISTS email_contato,
    DROP COLUMN IF EXISTS bio;
