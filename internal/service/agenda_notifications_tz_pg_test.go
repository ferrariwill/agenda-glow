//go:build integration

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "time/tzdata"
)

// A fila de notificações é gravada por um caminho e lida por outro. Enquanto
// `enqueueNotification` usava o NOW() do Postgres e o worker filtrava pelo relógio do
// processo Go, os dois tipos que nascem por enfileiramento direto —
// `CONFIRMACAO_RESERVA` e `CANCELAMENTO_CONFIRMADO` — ficavam invisíveis pelo tamanho
// exato da diferença de fuso. `PEDIDO_CONFIRMACAO` e `LEMBRETE` escapavam porque
// `discoverDue` grava com o mesmo relógio do filtro.
//
// Estes testes fixam o processo em America/Sao_Paulo e o banco em UTC, que é o arranjo
// do ambiente de desenvolvimento documentado (API no host, Postgres no container).

func useSaoPauloProcessClock(t *testing.T) {
	t.Helper()
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("carregar America/Sao_Paulo: %v", err)
	}
	previous := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = previous })
}

// wallClock devolve a hora de parede de t desprezando a localização, que é exatamente
// o que uma coluna `timestamp without time zone` guarda: ao gravar um parâmetro nessa
// coluna o Postgres ignora o offset enviado e retém os dígitos.
func wallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// assertClockSkew garante que o teste está de fato exercitando o cenário: sem
// defasagem entre as horas de parede dos dois relógios ele passaria mesmo com o
// defeito presente.
func assertClockSkew(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec("SET TIME ZONE 'UTC'"); err != nil {
		t.Fatalf("fixar o banco em UTC: %v", err)
	}
	var databaseNow time.Time
	mustGet(t, db, &databaseNow, "SELECT NOW()::timestamp")
	skew := wallClock(databaseNow).Sub(wallClock(time.Now())).Abs()
	if skew < time.Hour {
		t.Fatalf("os relógios precisam divergir para este teste valer: NOW() do Postgres grava %s, "+
			"relógio do processo Go marca %s, defasagem de %s",
			databaseNow.Format("15:04:05"), time.Now().Format("15:04:05-07:00"), skew)
	}
}

func newRecordingGateway(t *testing.T, sent *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*sent++
		_, _ = w.Write([]byte(`{"external_message_id":"gw-tz"}`))
	}))
}

// newRealClockWorker usa time.Now direto, como o main faz em produção: é o relógio
// local do processo que precisa concordar com o que foi gravado na fila.
func newRealClockWorker(db *sqlx.DB, gateway *httptest.Server) *AgendaNotificationWorker {
	return NewAgendaNotificationWorker(db, &GatewayNotificationSender{
		BaseURL: gateway.URL, APIKey: "k", Client: gateway.Client(),
	}, AgendaNotificationWorkerOptions{BaseURL: "https://app.example"})
}

// TestPostgresCancellationConfirmedSurvivesTimezoneSkew cobre o cancelamento público
// por token, que enfileira CANCELAMENTO_CONFIRMADO na mesma transação.
func TestPostgresCancellationConfirmedSurvivesTimezoneSkew(t *testing.T) {
	useSaoPauloProcessClock(t)
	db := newPostgresTestDB(t)
	assertClockSkew(t, db)

	tenant := seedTenant(t, db, "tzcancel")
	appointmentID := seedAppointment(t, db, tenant, "AGENDADO", time.Now().Add(48*time.Hour))

	var token string
	mustGet(t, db, &token, `SELECT gestao_token::text FROM agendamentos WHERE id = $1`, appointmentID)

	svc := NewAgendaService(db)
	if _, err := svc.CancelAppointmentByManagementToken(
		context.Background(), token, "Imprevisto", time.Now()); err != nil {
		t.Fatalf("cancelar por token: %v", err)
	}

	var queued struct {
		Status string    `db:"status_envio"`
		DueAt  time.Time `db:"proxima_tentativa_em"`
	}
	mustGet(t, db, &queued, `
SELECT status_envio, proxima_tentativa_em FROM agendamento_notificacoes
WHERE agendamento_id = $1 AND tipo = $2`, appointmentID, NotificationTypeCancellationConfirmed)
	if queued.Status != NotificationStatusPending {
		t.Fatalf("cancelamento não enfileirou envio: status %q", queued.Status)
	}

	sent := 0
	gateway := newRecordingGateway(t, &sent)
	defer gateway.Close()
	if err := newRealClockWorker(db, gateway).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if sent != 1 {
		t.Fatalf("CANCELAMENTO_CONFIRMADO não saiu: envios=%d, proxima_tentativa_em=%s, relógio do worker=%s",
			sent, queued.DueAt.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	}

	var status string
	mustGet(t, db, &status, `
SELECT status_envio FROM agendamento_notificacoes
WHERE agendamento_id = $1 AND tipo = $2`, appointmentID, NotificationTypeCancellationConfirmed)
	if status != "ENVIADO" {
		t.Fatalf("status final da fila: got %q, want ENVIADO", status)
	}
}

// TestPostgresReservationConfirmationSurvivesTimezoneSkew cobre o outro tipo que nasce
// por enfileiramento direto, pelo mesmo caminho que CriarAgendamento usa.
func TestPostgresReservationConfirmationSurvivesTimezoneSkew(t *testing.T) {
	useSaoPauloProcessClock(t)
	db := newPostgresTestDB(t)
	assertClockSkew(t, db)

	tenant := seedTenant(t, db, "tzreserva")
	appointmentID := seedAppointment(t, db, tenant, "AGENDADO", time.Now().Add(72*time.Hour))

	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatalf("abrir transação: %v", err)
	}
	if err := enqueueNotification(context.Background(), tx, tenant.EstablishmentID, appointmentID,
		NotificationTypeReservationConfirmation, time.Now()); err != nil {
		t.Fatalf("enfileirar confirmação de reserva: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("confirmar transação: %v", err)
	}

	// +72h fica fora das janelas de confirmação (24h) e lembrete (1h), então a única
	// linha reservável é a CONFIRMACAO_RESERVA enfileirada acima.
	sent := 0
	gateway := newRecordingGateway(t, &sent)
	defer gateway.Close()
	if err := newRealClockWorker(db, gateway).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if sent != 1 {
		var dueAt time.Time
		mustGet(t, db, &dueAt, `
SELECT proxima_tentativa_em FROM agendamento_notificacoes
WHERE agendamento_id = $1 AND tipo = $2`, appointmentID, NotificationTypeReservationConfirmation)
		t.Fatalf("CONFIRMACAO_RESERVA não saiu: envios=%d, proxima_tentativa_em=%s, relógio do worker=%s",
			sent, dueAt.Format(time.RFC3339), time.Now().Format(time.RFC3339))
	}

	var row struct {
		Type   string `db:"tipo"`
		Status string `db:"status_envio"`
	}
	mustGet(t, db, &row, `
SELECT tipo, status_envio FROM agendamento_notificacoes WHERE agendamento_id = $1`, appointmentID)
	if row.Type != NotificationTypeReservationConfirmation || row.Status != "ENVIADO" {
		t.Fatalf("linha da fila após envio: %+v", row)
	}
}

// TestPostgresQueueWritesUseASingleClock trava o invariante diretamente: as duas
// origens de escrita da fila têm de gravar `proxima_tentativa_em` no mesmo relógio.
func TestPostgresQueueWritesUseASingleClock(t *testing.T) {
	useSaoPauloProcessClock(t)
	db := newPostgresTestDB(t)
	assertClockSkew(t, db)

	tenant := seedTenant(t, db, "tzclock")
	now := time.Now()
	discovered := seedAppointment(t, db, tenant, "AGENDADO", now.Add(20*time.Hour))
	enqueued := seedAppointment(t, db, tenant, "AGENDADO", now.Add(72*time.Hour))

	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatalf("abrir transação: %v", err)
	}
	if err := enqueueNotification(context.Background(), tx, tenant.EstablishmentID, enqueued,
		NotificationTypeReservationConfirmation, now); err != nil {
		t.Fatalf("enfileirar: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	sent := 0
	gateway := newRecordingGateway(t, &sent)
	defer gateway.Close()
	if err := newRealClockWorker(db, gateway).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	var enqueuedDue, discoveredDue time.Time
	mustGet(t, db, &enqueuedDue, `
SELECT proxima_tentativa_em FROM agendamento_notificacoes WHERE agendamento_id = $1`, enqueued)
	mustGet(t, db, &discoveredDue, `
SELECT proxima_tentativa_em FROM agendamento_notificacoes
WHERE agendamento_id = $1 AND tipo = $2`, discovered, NotificationTypeConfirmationRequest)

	if drift := enqueuedDue.Sub(discoveredDue).Abs(); drift > time.Minute {
		t.Fatalf("enfileiramento e descoberta gravaram em relógios diferentes: defasagem de %s (%s vs %s)",
			drift, enqueuedDue.Format(time.RFC3339), discoveredDue.Format(time.RFC3339))
	}
}
