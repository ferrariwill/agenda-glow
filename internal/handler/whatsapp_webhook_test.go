package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
)

const whatsAppFlagQuery = "SELECT COALESCE(whatsapp_enabled, FALSE)"

func newWhatsAppWebhookHandler(t *testing.T) (*WhatsAppWebhookHandler, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("WHATSAPP_GATEWAY_KEY", "")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	return NewWhatsAppWebhookHandler(
		service.NewAgendaService(db),
		service.NewEstabelecimentoService(db),
	), mock
}

func postWebhook(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestConnectedWebhookIsForbiddenAndDoesNotWriteWhenFeatureIsOff(t *testing.T) {
	handler, mock := newWhatsAppWebhookHandler(t)
	mock.ExpectQuery(regexp.QuoteMeta(whatsAppFlagQuery)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	rec := httptest.NewRecorder()
	handler.Connected(rec, postWebhook("/api/v1/webhook/whatsapp-connected", `{
		"event": "whatsapp_connection_completed",
		"sistema_origem": "beleza",
		"tenant_id": "tenant-1",
		"waba_id": "waba-1",
		"phone_number_id": "phone-1",
		"status": "connected"
	}`))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "whatsapp_feature_disabled" ||
		body["message"] != "Recurso de WhatsApp desativado para este estabelecimento. Contate o administrador" {
		t.Fatalf("body = %#v", body)
	}
	// Só a consulta da flag foi emitida: nenhum UPDATE em estabelecimentos.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCallbackConfirmStaysAllowedWhenFeatureIsOff(t *testing.T) {
	handler, mock := newWhatsAppWebhookHandler(t)
	inicio := time.Now().Add(24 * time.Hour)
	mock.ExpectQuery(regexp.QuoteMeta("FROM agendamentos a")).
		WithArgs("agendamento-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "estabelecimento_id", "profissional_id", "servico_id",
			"data_hora_inicio", "data_hora_fim", "status",
			"cliente_nome", "cliente_telefone", "servico_nome",
			"profissional_nome", "profissional_email",
		}).AddRow(
			"agendamento-1", "tenant-1", "prof-1", "serv-1",
			inicio, inicio.Add(time.Hour), "AGENDADO",
			"Ana", "5511999998888", "Corte", "Bia", nil,
		))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE agendamentos")).
		WithArgs("agendamento-1", "CONFIRMADO", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	handler.Callback(rec, postWebhook("/api/v1/webhook/whatsapp-callback", `{
		"sistema_origem": "beleza",
		"tenant_id": "tenant-1",
		"appointment_id": "agendamento-1",
		"phone_number": "5511999998888",
		"event_type": "button_reply",
		"text": "APPT_CONFIRM",
		"action": "CONFIRM"
	}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (callback não é barrado pela flag)", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["appointment_status"] != "CONFIRMADO" {
		t.Fatalf("body = %#v", body)
	}
	// Nenhuma consulta à flag foi registrada no caminho do callback.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
