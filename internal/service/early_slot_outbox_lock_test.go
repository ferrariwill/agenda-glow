package service

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// Worker recovers first Gateway attempt when the process crashes after the offer
// is committed with proxima_tentativa_em due and tentativas_envio=0.
func TestWorkerRecoversFirstDeliveryAfterCrashBeforeGateway(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	var sent *WhatsAppNotificationInput
	svc.send = func(_ context.Context, in WhatsAppNotificationInput) error {
		copia := in
		sent = &copia
		return nil
	}

	currentStart := now.Add(48 * time.Hour)
	slotStart := now.Add(24 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(now, earlySlotMaxSendAttempts).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "estabelecimento_id", "rodada_id", "agendamento_candidato_id",
			"tentativas_envio", "cliente_nome", "cliente_telefone",
			"salon_name", "professional_name", "data_hora_inicio", "slot_inicio",
		}).AddRow(
			"oferta-1", "tenant-1", "rodada-1", "agendamento-1",
			0, "Maria", "5511999999999",
			"Glow", "Ana", currentStart, slotStart,
		))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET token_hash=$3`)).
		WithArgs("oferta-1", "tenant-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(regexp.QuoteMeta(`SET proxima_tentativa_em=NULL`)).
		WithArgs("oferta-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(now, earlySlotMaxSendAttempts).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	retried, err := svc.ProcessSendRetries(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if retried != 1 {
		t.Fatalf("recovered = %d, want 1", retried)
	}
	if sent == nil {
		t.Fatal("worker must send the first attempt after a crash")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Accept takes FOR UPDATE on profissionais before the collision SELECT — the
// same lock CriarAgendamento uses, serializing agenda writes.
func TestAcceptLocksProfessionalBeforeCollisionCheck(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer rawDB.Close()
	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.send = func(context.Context, WhatsAppNotificationInput) error { return nil }

	slotStart := now.Add(24 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)
	currentStart := now.Add(48 * time.Hour)
	currentEnd := currentStart.Add(45 * time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows(acceptLookupColumns()).AddRow(
			"oferta-1", "PENDENTE", now.Add(5*time.Minute), "rodada-1", "ATIVA",
			"tenant-1", "agendamento-1", "prof-1",
			slotStart, slotEnd, currentStart, currentEnd,
			"Maria", "5511999999999", "Ana", "Corte", true,
		))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM profissionais`)).
		WithArgs("prof-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("prof-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM agendamentos`)).
		WithArgs("tenant-1", "prof-1", "agendamento-1", slotStart, slotStart.Add(45*time.Minute)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE agendamentos SET data_hora_inicio=$3`)).
		WithArgs("agendamento-1", "tenant-1", slotStart, slotStart.Add(45*time.Minute)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='ACEITA'`)).
		WithArgs("oferta-1", "tenant-1", "WEB").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='INVALIDADA'`)).
		WithArgs("oferta-1", "tenant-1", "rodada-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rodadas_antecipacao SET status='PREENCHIDA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO antecipacao_auditoria`)).
		WithArgs("tenant-1", "rodada-1", "oferta-1",
			currentStart, currentEnd, slotStart, slotStart.Add(45*time.Minute), "WEB").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if _, err := svc.Accept(context.Background(), "token", "WEB"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
