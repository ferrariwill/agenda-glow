package service

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestProcessWhatsAppCallbackBlocksConfirmWhilePendingApproval(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	db := sqlx.NewDb(rawDB, "sqlmock")
	svc := NewAgendaService(db)
	expectWhatsAppAppointmentLookup(mock, "appointment-a", "tenant-a", "5511999999999", "EM_APROVACAO")

	status, err := svc.ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511999999999",
		Text:          "APPT_CONFIRM",
		Action:        WhatsAppActionConfirm,
		AppointmentID: "appointment-a",
	})
	if status != "" {
		t.Fatalf("status: got %q, want empty", status)
	}
	if !errors.Is(err, ErrAgendamentoAguardandoAprovacaoProfissional) {
		t.Fatalf("erro: got %v, want ErrAgendamentoAguardandoAprovacaoProfissional", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas SQL (nenhum UPDATE esperado): %v", err)
	}
}

func TestProcessWhatsAppCallbackAllowsCancelWhilePendingApproval(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	db := sqlx.NewDb(rawDB, "sqlmock")
	svc := NewAgendaService(db)
	expectWhatsAppAppointmentLookup(mock, "appointment-a", "tenant-a", "5511999999999", "EM_APROVACAO")
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE agendamentos
SET status = $2
WHERE id = $1
  AND estabelecimento_id = $3
`)).
		WithArgs("appointment-a", "CANCELADO", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))

	status, err := svc.ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511999999999",
		Text:          "APPT_CANCEL",
		Action:        WhatsAppActionCancel,
		AppointmentID: "appointment-a",
	})
	if err != nil {
		t.Fatalf("ProcessWhatsAppCallback: %v", err)
	}
	if status != "CANCELADO" {
		t.Fatalf("status: got %q, want CANCELADO", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas SQL: %v", err)
	}
}

func TestProcessWhatsAppCallbackConfirmScheduledStillWorks(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	db := sqlx.NewDb(rawDB, "sqlmock")
	svc := NewAgendaService(db)
	expectWhatsAppAppointmentLookup(mock, "appointment-a", "tenant-a", "5511999999999", "AGENDADO")
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE agendamentos
SET status = $2
WHERE id = $1
  AND estabelecimento_id = $3
`)).
		WithArgs("appointment-a", "CONFIRMADO", "tenant-a").
		WillReturnResult(sqlmock.NewResult(0, 1))

	status, err := svc.ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511999999999",
		Text:          "APPT_CONFIRM",
		Action:        WhatsAppActionConfirm,
		AppointmentID: "appointment-a",
	})
	if err != nil {
		t.Fatalf("ProcessWhatsAppCallback: %v", err)
	}
	if status != "CONFIRMADO" {
		t.Fatalf("status: got %q, want CONFIRMADO", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas SQL: %v", err)
	}
}

func TestProcessWhatsAppCallbackRejectsTenantMismatch(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	db := sqlx.NewDb(rawDB, "sqlmock")
	svc := NewAgendaService(db)
	expectWhatsAppAppointmentLookup(mock, "appointment-a", "tenant-a", "5511999999999", "AGENDADO")

	_, err = svc.ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-b",
		PhoneNumber:   "5511999999999",
		Action:        WhatsAppActionConfirm,
		AppointmentID: "appointment-a",
	})
	if !errors.Is(err, ErrAgendamentoEscopoInvalido) {
		t.Fatalf("erro: got %v, want ErrAgendamentoEscopoInvalido", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas SQL (sem UPDATE): %v", err)
	}
}

func TestProcessWhatsAppCallbackRejectsPhoneMismatch(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	db := sqlx.NewDb(rawDB, "sqlmock")
	svc := NewAgendaService(db)
	expectWhatsAppAppointmentLookup(mock, "appointment-a", "tenant-a", "5511999999999", "AGENDADO")

	_, err = svc.ProcessWhatsAppCallback(context.Background(), WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511888888888",
		Action:        WhatsAppActionConfirm,
		AppointmentID: "appointment-a",
	})
	if !errors.Is(err, ErrAgendamentoEscopoInvalido) {
		t.Fatalf("erro: got %v, want ErrAgendamentoEscopoInvalido", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectativas SQL (sem UPDATE): %v", err)
	}
}

func expectWhatsAppAppointmentLookup(
	mock sqlmock.Sqlmock,
	appointmentID, tenantID, phone, status string,
) {
	inicio := time.Date(2026, 7, 30, 15, 0, 0, 0, time.UTC)
	fim := inicio.Add(45 * time.Minute)
	rows := sqlmock.NewRows([]string{
		"id",
		"estabelecimento_id",
		"profissional_id",
		"servico_id",
		"data_hora_inicio",
		"data_hora_fim",
		"status",
		"cliente_nome",
		"cliente_telefone",
		"servico_nome",
		"profissional_nome",
		"profissional_email",
	}).AddRow(
		appointmentID,
		tenantID,
		"prof-a",
		"servico-a",
		inicio,
		fim,
		status,
		"Cliente Teste",
		phone,
		"Corte",
		"Profissional",
		nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT
    a.id,
    a.estabelecimento_id,
    a.profissional_id,
    a.servico_id,
    a.data_hora_inicio,
    a.data_hora_fim,
    a.status,
    c.nome AS cliente_nome,
    c.telefone AS cliente_telefone,
    s.nome AS servico_nome,
    p.nome AS profissional_nome,
    p.email AS profissional_email
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id
INNER JOIN servicos s ON s.id = a.servico_id
INNER JOIN profissionais p ON p.id = a.profissional_id
WHERE a.id = $1
`)).
		WithArgs(appointmentID).
		WillReturnRows(rows)
}
