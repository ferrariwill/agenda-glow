//go:build integration

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Estes testes exercitam o ciclo anti-no-show contra um Postgres real com o schema de
// produção: cada um cria um banco descartável e aplica migrations/*.up.sql do zero.
// O sqlmock casa SQL por regex e nunca submete a query ao banco, então erros de
// tipagem de parâmetro e de escolha de linha só aparecem aqui.
//
// Rodar:
//
//	TEST_DATABASE_URL="postgres://postgres:senha@localhost:5435/postgres?sslmode=disable" \
//	    go test -tags integration -count=1 ./internal/service
//
// A URL aponta para um banco administrativo — o usuário precisa poder CREATE DATABASE.
func newPostgresTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	adminURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if adminURL == "" {
		adminURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if adminURL == "" {
		t.Skip("defina TEST_DATABASE_URL para rodar os testes de integração com Postgres")
	}

	admin, err := sqlx.Connect("postgres", adminURL)
	if err != nil {
		t.Fatalf("conectar no Postgres administrativo: %v", err)
	}
	defer admin.Close()

	name := fmt.Sprintf("agendaglow_it_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("criar banco descartável: %v", err)
	}

	db, err := sqlx.Connect("postgres", replaceDatabaseName(t, adminURL, name))
	if err != nil {
		t.Fatalf("conectar no banco descartável: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		dropper, err := sqlx.Connect("postgres", adminURL)
		if err != nil {
			return
		}
		defer dropper.Close()
		_, _ = dropper.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
	})

	applyMigrations(t, db)
	return db
}

func replaceDatabaseName(t *testing.T, dsn, name string) string {
	t.Helper()
	cut := strings.SplitN(dsn, "?", 2)
	base := strings.TrimRight(cut[0], "/")
	slash := strings.LastIndex(base, "/")
	if slash < 0 {
		t.Fatalf("DSN sem nome de banco: %q", dsn)
	}
	out := base[:slash+1] + name
	if len(cut) == 2 {
		out += "?" + cut[1]
	}
	return out
}

func applyMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.up.sql"))
	if err != nil || len(files) == 0 {
		t.Fatalf("localizar migrações: %v (%d arquivos)", err, len(files))
	}
	sort.Strings(files)
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		if _, err := db.Exec(string(body)); err != nil {
			t.Fatalf("aplicar %s: %v", filepath.Base(path), err)
		}
	}
}

type seededTenant struct {
	EstablishmentID string
	ClientID        string
	ServiceID       string
	ProfessionalID  string
	Phone           string
}

func seedTenant(t *testing.T, db *sqlx.DB, slug string) seededTenant {
	t.Helper()
	out := seededTenant{Phone: "5511999990000"}
	mustGet(t, db, &out.EstablishmentID, `
INSERT INTO estabelecimentos (nome_comercial, slug, ativo, whatsapp_enabled, whatsapp_status, whatsapp_phone_number,
                              logradouro, cidade, uf)
VALUES ($1, $2, TRUE, TRUE, 'CONECTADO', '5511000000000', 'Rua Um', 'São Paulo', 'SP')
RETURNING id`, "Studio "+slug, slug)

	mustExec(t, db, `
INSERT INTO configuracoes_notificacoes_agenda (estabelecimento_id) VALUES ($1)
ON CONFLICT (estabelecimento_id) DO NOTHING`, out.EstablishmentID)

	var specialtyID string
	mustGet(t, db, &specialtyID, `
INSERT INTO especialidades (estabelecimento_id, nome) VALUES ($1, 'Cabelo') RETURNING id`,
		out.EstablishmentID)

	mustGet(t, db, &out.ProfessionalID, `
INSERT INTO profissionais (nome, estabelecimento_id, especialidade_id) VALUES ('Ana', $1, $2)
RETURNING id`, out.EstablishmentID, specialtyID)

	mustGet(t, db, &out.ServiceID, `
INSERT INTO servicos (nome, preco_base, duracao_base_minutos, estabelecimento_id)
VALUES ('Corte', 100.00, 60, $1) RETURNING id`, out.EstablishmentID)

	mustGet(t, db, &out.ClientID, `
INSERT INTO clientes (nome, telefone, estabelecimento_id) VALUES ('Maria', $1, $2) RETURNING id`,
		out.Phone, out.EstablishmentID)

	return out
}

func seedAppointment(t *testing.T, db *sqlx.DB, tenant seededTenant, status string, startsAt time.Time) string {
	t.Helper()
	var id string
	mustGet(t, db, &id, `
INSERT INTO agendamentos (profissional_id, servico_id, data_hora_inicio, data_hora_fim, status,
                          cliente_id, origem_agendamento, estabelecimento_id)
VALUES ($1, $2, $3, $4, $5, $6, 'EXTERNO', $7)
RETURNING id`,
		tenant.ProfessionalID, tenant.ServiceID, startsAt, startsAt.Add(time.Hour), status,
		tenant.ClientID, tenant.EstablishmentID)
	return id
}

func mustExec(t *testing.T, db *sqlx.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec: %v\nquery: %s", err, query)
	}
}

func mustGet(t *testing.T, db *sqlx.DB, dest any, query string, args ...any) {
	t.Helper()
	if err := db.Get(dest, query, args...); err != nil {
		t.Fatalf("get: %v\nquery: %s", err, query)
	}
}

// TestPostgresWorkerSendsAndDoesNotDuplicate roda o worker ponta a ponta contra o
// Postgres: recuperação de reservas travadas, descoberta por janela, reserva com
// SKIP LOCKED, envio ao Gateway e conclusão. Antes do cast explícito de $1 a primeira
// instrução de RunOnce abortava toda a execução e nenhuma mensagem saía.
func TestPostgresWorkerSendsAndDoesNotDuplicate(t *testing.T) {
	db := newPostgresTestDB(t)
	tenant := seedTenant(t, db, "worker")
	now := time.Now().UTC().Truncate(time.Second)
	appointmentID := seedAppointment(t, db, tenant, "AGENDADO", now.Add(20*time.Hour))

	var sent []whatsAppSendNotificationRequest
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload whatsAppSendNotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decodificar payload do gateway: %v", err)
		}
		sent = append(sent, payload)
		_, _ = w.Write([]byte(`{"external_message_id":"gw-1"}`))
	}))
	defer gateway.Close()

	worker := NewAgendaNotificationWorker(db, &GatewayNotificationSender{
		BaseURL: gateway.URL, APIKey: "k", Client: gateway.Client(),
	}, AgendaNotificationWorkerOptions{
		BaseURL: "https://app.example",
		Now:     func() time.Time { return now },
	})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("primeiro RunOnce: %v", err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("segundo RunOnce: %v", err)
	}

	if len(sent) != 1 {
		t.Fatalf("chamadas ao Gateway: got %d, want 1 (%+v)", len(sent), sent)
	}
	if sent[0].TenantID != tenant.EstablishmentID || sent[0].AppointmentID != appointmentID {
		t.Fatalf("escopo do envio: %+v", sent[0])
	}
	if sent[0].PhoneNumber != tenant.Phone {
		t.Fatalf("telefone: got %q, want %q", sent[0].PhoneNumber, tenant.Phone)
	}

	var row struct {
		Type       string         `db:"tipo"`
		Status     string         `db:"status_envio"`
		Attempts   int            `db:"tentativas"`
		ExternalID sql.NullString `db:"external_message_id"`
	}
	mustGet(t, db, &row, `
SELECT tipo, status_envio, tentativas, external_message_id
FROM agendamento_notificacoes WHERE agendamento_id = $1`, appointmentID)
	if row.Type != NotificationTypeConfirmationRequest {
		t.Fatalf("tipo enfileirado: got %q, want %q", row.Type, NotificationTypeConfirmationRequest)
	}
	if row.Status != "ENVIADO" || row.Attempts != 1 || row.ExternalID.String != "gw-1" {
		t.Fatalf("linha da fila após envio: %+v", row)
	}
}

// TestPostgresWorkerRecoversStuckReservation cobre a primeira instrução de RunOnce, a
// que travava a execução inteira: a reserva ENVIANDO expirada volta para FALHOU.
func TestPostgresWorkerRecoversStuckReservation(t *testing.T) {
	db := newPostgresTestDB(t)
	tenant := seedTenant(t, db, "stuck")
	now := time.Now().UTC().Truncate(time.Second)
	appointmentID := seedAppointment(t, db, tenant, "AGENDADO", now.Add(20*time.Hour))

	mustExec(t, db, `
INSERT INTO agendamento_notificacoes (estabelecimento_id, agendamento_id, tipo, status_envio,
                                      tentativas, proxima_tentativa_em, atualizado_em)
VALUES ($1, $2, $3, 'ENVIANDO', 1, $4, $4)`,
		tenant.EstablishmentID, appointmentID, NotificationTypeReminder, now.Add(-30*time.Minute))

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"external_message_id":"gw-2"}`))
	}))
	defer gateway.Close()

	worker := NewAgendaNotificationWorker(db, &GatewayNotificationSender{
		BaseURL: gateway.URL, APIKey: "k", Client: gateway.Client(),
	}, AgendaNotificationWorkerOptions{Now: func() time.Time { return now }})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	var status string
	mustGet(t, db, &status, `
SELECT status_envio FROM agendamento_notificacoes
WHERE agendamento_id = $1 AND tipo = $2`, appointmentID, NotificationTypeReminder)
	if status != "ENVIADO" {
		t.Fatalf("reserva travada não foi recuperada e reenviada: status %q", status)
	}
}

// TestPostgresManagementTokenMalformedIsNotFound: os endpoints públicos são anônimos,
// então um token que não é UUID é um token desconhecido (404), nunca um 500.
func TestPostgresManagementTokenMalformedIsNotFound(t *testing.T) {
	db := newPostgresTestDB(t)
	seedTenant(t, db, "token")
	svc := NewAgendaService(db)
	now := time.Now().UTC()

	for _, token := range []string{"nao-e-uuid", "", "'; DROP TABLE agendamentos; --", "12345"} {
		if _, err := svc.GetAppointmentManagement(context.Background(), token, now); !errors.Is(err, ErrAgendamentoNaoEncontrado) {
			t.Fatalf("GET manage %q: got %v, want ErrAgendamentoNaoEncontrado", token, err)
		}
		if _, err := svc.CancelAppointmentByManagementToken(context.Background(), token, "motivo", now); !errors.Is(err, ErrAgendamentoNaoEncontrado) {
			t.Fatalf("POST cancel %q: got %v, want ErrAgendamentoNaoEncontrado", token, err)
		}
	}

	unknown := "11111111-2222-3333-4444-555555555555"
	if _, err := svc.GetAppointmentManagement(context.Background(), unknown, now); !errors.Is(err, ErrAgendamentoNaoEncontrado) {
		t.Fatalf("UUID desconhecido: got %v, want ErrAgendamentoNaoEncontrado", err)
	}
}

// TestPostgresWhatsAppCancelWithoutAppointmentIDHitsActive: uma cliente com um
// agendamento CANCELADO mais distante e um AGENDADO mais próximo responde CANCEL sem
// appointment_id. O alvo tem de ser o ativo.
func TestPostgresWhatsAppCancelWithoutAppointmentIDHitsActive(t *testing.T) {
	db := newPostgresTestDB(t)
	tenant := seedTenant(t, db, "cancel")
	now := time.Now().UTC()
	cancelledID := seedAppointment(t, db, tenant, "CANCELADO", now.Add(120*time.Hour))
	activeID := seedAppointment(t, db, tenant, "AGENDADO", now.Add(48*time.Hour))

	status, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      tenant.EstablishmentID,
		PhoneNumber:   tenant.Phone,
		Action:        WhatsAppActionCancel,
		Reason:        "Imprevisto",
	})
	if err != nil {
		t.Fatalf("CANCEL sem appointment_id: %v", err)
	}
	if status != "CANCELADO" {
		t.Fatalf("status retornado: got %q, want CANCELADO", status)
	}

	var active struct {
		Status       string `db:"status"`
		Confirmation string `db:"confirmacao_cliente"`
	}
	mustGet(t, db, &active, `
SELECT status, confirmacao_cliente FROM agendamentos WHERE id = $1`, activeID)
	if active.Status != "CANCELADO" || active.Confirmation != "CANCELADO_CLIENTE" {
		t.Fatalf("agendamento ativo não foi cancelado: %+v", active)
	}

	var previousConfirmation string
	mustGet(t, db, &previousConfirmation, `
SELECT confirmacao_cliente FROM agendamentos WHERE id = $1`, cancelledID)
	if previousConfirmation != "PENDENTE" {
		t.Fatalf("agendamento já cancelado foi tocado: confirmacao_cliente=%q", previousConfirmation)
	}
}

// TestPostgresWhatsAppCancelWithAppointmentIDKeepsSpecificError: com appointment_id o
// alvo é exato, então um agendamento concluído continua devolvendo o erro específico
// em vez de "não encontrado".
func TestPostgresWhatsAppCancelWithAppointmentIDKeepsSpecificError(t *testing.T) {
	db := newPostgresTestDB(t)
	tenant := seedTenant(t, db, "concluido")
	now := time.Now().UTC()
	completedID := seedAppointment(t, db, tenant, "CONCLUIDO", now.Add(72*time.Hour))

	_, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      tenant.EstablishmentID,
		PhoneNumber:   tenant.Phone,
		Action:        WhatsAppActionCancel,
		AppointmentID: completedID,
		Reason:        "Imprevisto",
	})
	if !errors.Is(err, ErrAppointmentNotCancellable) {
		t.Fatalf("CONCLUIDO com appointment_id: got %v, want ErrAppointmentNotCancellable", err)
	}
}
