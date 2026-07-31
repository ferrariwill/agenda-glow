//go:build integration

package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type postgresWorkerSender struct {
	sent []WhatsAppNotificationInput
}

func (s *postgresWorkerSender) Send(_ context.Context, input WhatsAppNotificationInput) (string, error) {
	s.sent = append(s.sent, input)
	return fmt.Sprintf("message-%d", len(s.sent)), nil
}

func TestAgendaNotificationWorkerPostgresEndToEnd(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		t.Skip("defina TEST_DATABASE_URL para executar a integração real do worker")
	}

	admin, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatalf("conectar ao Postgres: %v", err)
	}
	defer admin.Close()

	schema := fmt.Sprintf("agenda_worker_%d", time.Now().UnixNano())
	if _, err := admin.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatalf("criar schema isolado: %v", err)
	}
	defer func() {
		if _, err := admin.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); err != nil {
			t.Errorf("remover schema isolado: %v", err)
		}
	}()

	parsedDSN, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("interpretar TEST_DATABASE_URL: %v", err)
	}
	query := parsedDSN.Query()
	query.Set("search_path", schema+",public")
	parsedDSN.RawQuery = query.Encode()

	db, err := sqlx.Connect("postgres", parsedDSN.String())
	if err != nil {
		t.Fatalf("conectar ao schema isolado: %v", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	const ddl = `
CREATE TABLE estabelecimentos (
    id UUID PRIMARY KEY,
    nome_comercial TEXT NOT NULL,
    ativo BOOLEAN NOT NULL,
    whatsapp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    whatsapp_status TEXT NOT NULL,
    logradouro TEXT,
    cidade TEXT,
    uf TEXT
);
CREATE TABLE configuracoes_notificacoes_agenda (
    estabelecimento_id UUID PRIMARY KEY,
    lembretes_ativos BOOLEAN NOT NULL,
    antecedencia_confirmacao_horas INTEGER NOT NULL,
    antecedencia_lembrete_horas INTEGER NOT NULL,
    template_confirmacao TEXT NOT NULL,
    template_lembrete TEXT NOT NULL
);
CREATE TABLE clientes (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL,
    telefone TEXT NOT NULL
);
CREATE TABLE servicos (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL,
    nome TEXT NOT NULL
);
CREATE TABLE profissionais (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL,
    nome TEXT NOT NULL
);
CREATE TABLE agendamentos (
    id UUID PRIMARY KEY,
    estabelecimento_id UUID NOT NULL,
    cliente_id UUID NOT NULL,
    servico_id UUID NOT NULL,
    profissional_id UUID NOT NULL,
    data_hora_inicio TIMESTAMP NOT NULL,
    status TEXT NOT NULL,
    confirmacao_cliente TEXT NOT NULL,
    gestao_token UUID NOT NULL
);
CREATE TABLE agendamento_notificacoes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    estabelecimento_id UUID NOT NULL,
    agendamento_id UUID NOT NULL,
    tipo TEXT NOT NULL,
    status_envio TEXT NOT NULL,
    tentativas INTEGER NOT NULL DEFAULT 0,
    proxima_tentativa_em TIMESTAMP NOT NULL,
    enviado_em TIMESTAMP,
    external_message_id TEXT,
    payload_resumido JSONB NOT NULL DEFAULT '{}'::jsonb,
    ultimo_erro TEXT,
    criado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    atualizado_em TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (estabelecimento_id, agendamento_id, tipo)
);`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("criar tabelas mínimas: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	const (
		tenantID       = "10000000-0000-0000-0000-000000000001"
		customerID     = "20000000-0000-0000-0000-000000000001"
		serviceID      = "30000000-0000-0000-0000-000000000001"
		professionalID = "40000000-0000-0000-0000-000000000001"
		appointmentID  = "50000000-0000-0000-0000-000000000001"
		managementID   = "60000000-0000-0000-0000-000000000001"
	)
	seeds := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO estabelecimentos VALUES ($1, 'Studio PG', TRUE, TRUE, 'CONECTADO', 'Rua Um', 'São Paulo', 'SP')`, []any{tenantID}},
		{`INSERT INTO configuracoes_notificacoes_agenda VALUES ($1, TRUE, 24, 1, 'Confirme {{servico}}', 'Lembrete {{servico}}')`, []any{tenantID}},
		{`INSERT INTO clientes VALUES ($1, $2, '5511999999999')`, []any{customerID, tenantID}},
		{`INSERT INTO servicos VALUES ($1, $2, 'Corte')`, []any{serviceID, tenantID}},
		{`INSERT INTO profissionais VALUES ($1, $2, 'Ana')`, []any{professionalID, tenantID}},
		{`INSERT INTO agendamentos VALUES ($1, $2, $3, $4, $5, $6, 'AGENDADO', 'PENDENTE', $7)`,
			[]any{appointmentID, tenantID, customerID, serviceID, professionalID, now.Add(30 * time.Minute), managementID}},
	}
	for _, seed := range seeds {
		if _, err := db.Exec(seed.query, seed.args...); err != nil {
			t.Fatalf("semear cenário do worker: %v", err)
		}
	}

	sender := &postgresWorkerSender{}
	worker := NewAgendaNotificationWorker(db, sender, AgendaNotificationWorkerOptions{
		BaseURL:    "https://app.example",
		MaxRetries: 3,
		Now:        func() time.Time { return now },
	})
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce no Postgres real: %v", err)
	}

	var sentCount int
	if err := db.Get(&sentCount, `
SELECT COUNT(*) FROM agendamento_notificacoes WHERE status_envio = 'ENVIADO'
`); err != nil {
		t.Fatalf("contar notificações enviadas: %v", err)
	}
	if sentCount != 2 || len(sender.sent) != 2 {
		t.Fatalf("envios: banco=%d sender=%d, want 2", sentCount, len(sender.sent))
	}

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("segundo RunOnce no Postgres real: %v", err)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("segunda execução duplicou envios: %d", len(sender.sent))
	}
}
