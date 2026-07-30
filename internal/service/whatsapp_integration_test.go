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

// A varredura de lembretes é a única porta de saída do fluxo agendado, então o
// gate de DEV-6 vive no SQL de descoberta: um salão com o recurso desligado não
// pode sequer enfileirar notificação.
func TestReminderDiscoveryRequiresWhatsAppFeatureEnabled(t *testing.T) {
	if !regexp.MustCompile(`COALESCE\(e\.whatsapp_enabled, FALSE\) = TRUE`).
		MatchString(agendaNotificationDiscoverySQL) {
		t.Fatal("descoberta de notificações não filtra por whatsapp_enabled")
	}
	if !regexp.MustCompile(`e\.whatsapp_status = 'CONECTADO'`).
		MatchString(agendaNotificationDiscoverySQL) {
		t.Fatal("descoberta de notificações não filtra por whatsapp_status")
	}
}

func TestReminderDoesNotSendWhenWhatsAppFeatureIsDisabled(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	defer rawDB.Close() //nolint:errcheck

	requests := 0
	gateway := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer gateway.Close()
	t.Setenv("WHATSAPP_GATEWAY_URL", gateway.URL)
	t.Setenv("WHATSAPP_GATEWAY_KEY", "test-key")

	db := sqlx.NewDb(rawDB, "sqlmock")
	now := time.Now()
	worker := NewAgendaNotificationWorker(db, &recordingNotificationSender{},
		AgendaNotificationWorkerOptions{Now: func() time.Time { return now }})

	mock.ExpectExec("UPDATE agendamento_notificacoes").
		WillReturnResult(sqlmock.NewResult(0, 0))
	// O salão está com o recurso desligado: a descoberta não insere nada e a
	// reserva seguinte não encontra trabalho, então nada é enviado.
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectBegin()
	mock.ExpectQuery("FROM agendamento_notificacoes").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if requests != 0 {
		t.Fatalf("gateway recebeu %d requisição(ões)", requests)
	}
}

type recordingNotificationSender struct{ calls int }

func (s *recordingNotificationSender) Send(context.Context, WhatsAppNotificationInput) (string, error) {
	s.calls++
	return "", nil
}
