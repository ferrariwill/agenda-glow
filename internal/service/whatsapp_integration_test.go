package service

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func newWhatsAppTestService(t *testing.T) (*EstabelecimentoService, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	return NewEstabelecimentoService(sqlx.NewDb(rawDB, "sqlmock")), mock
}

func TestSetWhatsAppEnabledReturnsConsolidatedView(t *testing.T) {
	svc, mock := newWhatsAppTestService(t)
	connectedAt := time.Date(2026, 7, 20, 14, 3, 11, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE estabelecimentos")).
		WithArgs("tenant-1", true).
		WillReturnRows(sqlmock.NewRows([]string{
			"estabelecimento_id", "nome_comercial", "whatsapp_enabled", "whatsapp_status", "connected_at",
		}).AddRow("tenant-1", "Studio Bella", true, "CONECTADO", connectedAt))

	view, err := svc.SetWhatsAppEnabled(context.Background(), "tenant-1", true)
	if err != nil {
		t.Fatalf("SetWhatsAppEnabled: %v", err)
	}
	if !view.WhatsAppEnabled || view.WhatsAppStatus != "CONECTADO" || view.ConnectedAt == nil {
		t.Fatalf("view inesperada: %#v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetWhatsAppEnabledIsIdempotent(t *testing.T) {
	svc, mock := newWhatsAppTestService(t)
	for range 2 {
		mock.ExpectQuery(regexp.QuoteMeta("UPDATE estabelecimentos")).
			WithArgs("tenant-1", false).
			WillReturnRows(sqlmock.NewRows([]string{
				"estabelecimento_id", "nome_comercial", "whatsapp_enabled", "whatsapp_status", "connected_at",
			}).AddRow("tenant-1", "Studio Bella", false, "CONECTADO", nil))
	}

	for range 2 {
		view, err := svc.SetWhatsAppEnabled(context.Background(), "tenant-1", false)
		if err != nil {
			t.Fatalf("SetWhatsAppEnabled: %v", err)
		}
		if view.WhatsAppEnabled || view.WhatsAppStatus != "CONECTADO" {
			t.Fatalf("estado da conexão foi alterado: %#v", view)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetWhatsAppEnabledUnknownEstablishment(t *testing.T) {
	svc, mock := newWhatsAppTestService(t)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE estabelecimentos")).
		WithArgs("missing", true).
		WillReturnError(sql.ErrNoRows)

	_, err := svc.SetWhatsAppEnabled(context.Background(), "missing", true)
	if !errors.Is(err, ErrEstabelecimentoNaoEncontrado) {
		t.Fatalf("erro = %v", err)
	}
}

func TestMarkWhatsAppConnectedRejectsDisabledFeatureWithoutUpdate(t *testing.T) {
	svc, mock := newWhatsAppTestService(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	_, err := svc.MarkWhatsAppConnected(context.Background(), WhatsAppConnectedPayload{
		State:  "beleza_tenant-1",
		WabaID: "waba-1",
	})
	if !errors.Is(err, ErrWhatsAppRecursoDesativado) {
		t.Fatalf("erro = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetWhatsAppIntegrationHidesSignupWhenDisabled(t *testing.T) {
	svc, mock := newWhatsAppTestService(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id,")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "whatsapp_enabled", "whatsapp_status", "whatsapp_waba_id",
			"whatsapp_phone_number_id", "whatsapp_connected_at",
		}).AddRow("tenant-1", false, "DESCONECTADO", nil, nil, nil))

	view, err := svc.GetWhatsAppIntegration(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("GetWhatsAppIntegration: %v", err)
	}
	if view.SignupURL != "" || view.State != "" {
		t.Fatalf("credenciais de signup expostas: %#v", view)
	}
}

func TestReminderDoesNotSendWhenWhatsAppFeatureIsDisabled(t *testing.T) {
	svc, mock := newWhatsAppTestService(t)
	requests := 0
	gateway := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer gateway.Close()
	t.Setenv("WHATSAPP_GATEWAY_URL", gateway.URL)
	t.Setenv("WHATSAPP_GATEWAY_KEY", "test-key")

	mock.ExpectQuery("FROM agendamentos a").
		WithArgs("appointment-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "estabelecimento_id", "profissional_id", "servico_id",
			"data_hora_inicio", "data_hora_fim", "status", "cliente_nome",
			"cliente_telefone", "servico_nome", "profissional_nome", "profissional_email",
		}).AddRow(
			"appointment-1", "tenant-1", "professional-1", "service-1",
			time.Now(), time.Now().Add(time.Hour), "AGENDADO", "Cliente",
			"5511999999999", "Corte", "Profissional", nil,
		))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	agenda := NewAgendaService(svc.db)
	agenda.dispararLembreteWhatsAppAgendamento("appointment-1")
	if requests != 0 {
		t.Fatalf("gateway recebeu %d requisição(ões)", requests)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
