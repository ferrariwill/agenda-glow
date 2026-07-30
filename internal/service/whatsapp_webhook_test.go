package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestProcessWhatsAppCallbackConfirmPreservesOperationalStatus(t *testing.T) {
	for _, status := range []string{"AGENDADO", "EM_APROVACAO"} {
		t.Run(status, func(t *testing.T) {
			db, mock, closeDB := callbackTestDB(t)
			defer closeDB()
			mock.ExpectBegin()
			expectCallbackLookup(mock, status, "PENDENTE")
			mock.ExpectExec("UPDATE agendamentos[\\s\\S]+confirmacao_cliente = 'CONFIRMADO_CLIENTE'").
				WithArgs("appointment-a", "tenant-a").
				WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectCommit()

			got, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), callbackPayload(WhatsAppActionConfirm))
			if err != nil {
				t.Fatalf("ProcessWhatsAppCallback: %v", err)
			}
			if got != status {
				t.Fatalf("status operacional alterado: got %q, want %q", got, status)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProcessWhatsAppCallbackConfirmIsIdempotent(t *testing.T) {
	db, mock, closeDB := callbackTestDB(t)
	defer closeDB()
	mock.ExpectBegin()
	expectCallbackLookup(mock, "AGENDADO", "CONFIRMADO_CLIENTE")
	mock.ExpectCommit()

	got, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), callbackPayload(WhatsAppActionConfirm))
	if err != nil || got != "AGENDADO" {
		t.Fatalf("got status=%q err=%v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessWhatsAppCallbackRejectsCrossTenantAppointment(t *testing.T) {
	db, mock, closeDB := callbackTestDB(t)
	defer closeDB()
	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT.+FROM agendamentos a.+FOR UPDATE OF a").
		WithArgs("tenant-a", "5511999999999", "appointment-a").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("appointment-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	payload := callbackPayload(WhatsAppActionConfirm)
	payload.TenantID = "tenant-a"
	_, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), payload)
	if !errors.Is(err, ErrAgendamentoEscopoInvalido) {
		t.Fatalf("erro: got %v, want ErrAgendamentoEscopoInvalido", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessWhatsAppCallbackRejectsPhoneMismatch(t *testing.T) {
	db, mock, closeDB := callbackTestDB(t)
	defer closeDB()
	mock.ExpectBegin()
	mock.ExpectQuery("(?s)SELECT.+FROM agendamentos a.+FOR UPDATE OF a").
		WithArgs("tenant-a", "5511888888888", "appointment-a").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("appointment-a").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectRollback()

	payload := callbackPayload(WhatsAppActionConfirm)
	payload.PhoneNumber = "5511888888888"
	_, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), payload)
	if !errors.Is(err, ErrAgendamentoEscopoInvalido) {
		t.Fatalf("erro: got %v, want ErrAgendamentoEscopoInvalido", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessWhatsAppCallbackCancelUpdatesBothStatesAndAudits(t *testing.T) {
	db, mock, closeDB := callbackTestDB(t)
	defer closeDB()
	mock.ExpectBegin()
	expectCallbackLookup(mock, "EM_APROVACAO", "PENDENTE")
	mock.ExpectExec("UPDATE agendamentos[\\s\\S]+status = 'CANCELADO'").
		WithArgs("appointment-a", "tenant-a", "Imprevisto").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WithArgs("tenant-a", "appointment-a", NotificationTypeCancellationByCustomer,
			NotificationStatusRecorded, sqlmock.AnyArg(),
			safeAuditSummary{reason: "Imprevisto", origin: CancellationOriginWhatsApp}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WithArgs("tenant-a", "appointment-a", NotificationTypeCancellationConfirmed,
			NotificationStatusPending, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	payload := callbackPayload(WhatsAppActionCancel)
	payload.Reason = "Imprevisto"
	got, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), payload)
	if err != nil || got != "CANCELADO" {
		t.Fatalf("got status=%q err=%v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessWhatsAppCallbackWithoutAppointmentIDTargetsNextActiveFutureAppointment(t *testing.T) {
	db, mock, closeDB := callbackTestDB(t)
	defer closeDB()
	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{
		"id", "estabelecimento_id", "data_hora_inicio", "status",
		"confirmacao_cliente", "janela_minima_cancelamento_horas",
		"motivo_cancelamento_obrigatorio", "telefone_contato",
	}).AddRow(
		"appointment-next", "tenant-a", time.Now().Add(2*time.Hour), "AGENDADO",
		"PENDENTE", 0, false, "5511000000000",
	)
	mock.ExpectQuery(
		"(?s)SELECT.+FROM agendamentos a.+a.status IN \\('AGENDADO', 'CONFIRMADO', 'EM_APROVACAO'\\).+"+
			"a.data_hora_inicio >= NOW\\(\\).+ORDER BY a.data_hora_inicio ASC.+FOR UPDATE OF a",
	).
		WithArgs("tenant-a", "5511999999999", "").
		WillReturnRows(rows)
	mock.ExpectExec("UPDATE agendamentos[\\s\\S]+status = 'CANCELADO'").
		WithArgs("appointment-next", "tenant-a", "").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	payload := callbackPayload(WhatsAppActionCancel)
	payload.AppointmentID = ""
	got, err := NewAgendaService(db).ProcessWhatsAppCallback(context.Background(), payload)
	if err != nil || got != "CANCELADO" {
		t.Fatalf("got status=%q err=%v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func callbackTestDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return sqlx.NewDb(raw, "sqlmock"), mock, func() { _ = raw.Close() }
}

func callbackPayload(action string) WhatsAppCallbackPayload {
	return WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511999999999",
		Action:        action,
		AppointmentID: "appointment-a",
	}
}

func expectCallbackLookup(mock sqlmock.Sqlmock, status, confirmation string) {
	startsAt := time.Now().Add(24 * time.Hour)
	rows := sqlmock.NewRows([]string{
		"id", "estabelecimento_id", "data_hora_inicio", "status",
		"confirmacao_cliente", "janela_minima_cancelamento_horas",
		"motivo_cancelamento_obrigatorio", "telefone_contato",
	}).AddRow("appointment-a", "tenant-a", startsAt, status, confirmation, 2, false, "5511000000000")
	mock.ExpectQuery("(?s)SELECT.+FROM agendamentos a.+FOR UPDATE OF a").
		WithArgs("tenant-a", "5511999999999", "appointment-a").
		WillReturnRows(rows)
}
