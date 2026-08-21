-- DEV-212: avisos globais da plataforma (inbox do sino / Super Admin broadcast).

CREATE TABLE avisos_globais (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo VARCHAR(200) NOT NULL,
    corpo TEXT,
    severidade VARCHAR(20) NOT NULL DEFAULT 'INFO'
        CHECK (severidade IN ('INFO', 'WARNING', 'CRITICAL')),
    audience_tipo VARCHAR(30) NOT NULL
        CHECK (audience_tipo IN ('SUPER_ADMINS', 'ALL_TENANTS', 'ESTABELECIMENTOS')),
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users (id),
    criado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP
);

CREATE INDEX idx_avisos_globais_inbox
    ON avisos_globais (ativo, expires_at, criado_em DESC);

CREATE TABLE aviso_estabelecimentos (
    aviso_id UUID NOT NULL REFERENCES avisos_globais (id) ON DELETE CASCADE,
    estabelecimento_id UUID NOT NULL REFERENCES estabelecimentos (id) ON DELETE CASCADE,
    PRIMARY KEY (aviso_id, estabelecimento_id)
);

CREATE TABLE aviso_leituras (
    aviso_id UUID NOT NULL REFERENCES avisos_globais (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    read_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (aviso_id, user_id)
);

CREATE INDEX idx_aviso_leituras_user ON aviso_leituras (user_id);
