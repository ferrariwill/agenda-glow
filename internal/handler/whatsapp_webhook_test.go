package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
)

const (
	testAppointmentID = "appointment-a"
	testTenantID      = "tenant-a"
	testPhone         = "5511999999999"
	callbackRoute     = "/api/v1/webhook/whatsapp-callback"
	gatewayRoute      = "/api/v1/webhook/whatsapp-gateway"
)

func TestMapWhatsAppWebhookErrorKeepsDistinctConflictCodes(t *testing.T) {
	cases := map[error]struct {
		status int
		code   string
	}{
		service.ErrAgendamentoAguardandoAprovacaoProfissional: {http.StatusConflict, "appointment_pending_professional_approval"},
		service.ErrAgendamentoStatusFinal:                     {http.StatusConflict, "appointment_not_updatable"},
		service.ErrAgendamentoEscopoInvalido:                  {http.StatusForbidden, "appointment_scope_mismatch"},
		service.ErrAgendamentoNaoEncontrado:                   {http.StatusNotFound, "appointment_not_found"},
		service.ErrAcaoWhatsAppInvalida:                       {http.StatusBadRequest, "invalid_payload"},
	}
	for err, want := range cases {
		recorder := httptest.NewRecorder()
		mapWhatsAppWebhookError(recorder, err)
		if recorder.Code != want.status {
			t.Fatalf("%v: status got %d, want %d", err, recorder.Code, want.status)
		}
		if got := decodeField(t, recorder.Body.String(), "error"); got != want.code {
			t.Fatalf("%v: error got %q, want %q", err, got, want.code)
		}
	}
}

func TestWhatsAppWebhookConfirmPreservesOperationalStatus(t *testing.T) {
	for _, route := range []string{callbackRoute, gatewayRoute} {
		for _, status := range []string{"AGENDADO", "EM_APROVACAO"} {
			t.Run(route+"_"+status, func(t *testing.T) {
				mux, mock, cleanup := newWhatsAppWebhookTestServer(t)
				defer cleanup()
				mock.ExpectBegin()
				expectAppointmentLookup(mock, status, "PENDENTE")
				mock.ExpectExec("UPDATE agendamentos[\\s\\S]+CONFIRMADO_CLIENTE").
					WithArgs(testAppointmentID, testTenantID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()

				code, body := postWhatsAppWebhook(t, mux, route, map[string]any{
					"sistema_origem": "beleza", "tenant_id": testTenantID,
					"phone_number": testPhone, "event_type": "button_reply",
					"action": "confirm", "appointment_id": testAppointmentID,
				})
				if code != http.StatusOK || decodeField(t, body, "appointment_status") != status {
					t.Fatalf("status=%d body=%s", code, body)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestWhatsAppWebhookFinalStatusesArePreserved(t *testing.T) {
	for _, status := range []string{"CANCELADO", "CONCLUIDO"} {
		t.Run(status, func(t *testing.T) {
			mux, mock, cleanup := newWhatsAppWebhookTestServer(t)
			defer cleanup()
			mock.ExpectBegin()
			expectAppointmentLookup(mock, status, "PENDENTE")
			mock.ExpectRollback()

			code, body := postWhatsAppWebhook(t, mux, callbackRoute, map[string]any{
				"sistema_origem": "beleza", "tenant_id": testTenantID,
				"phone_number": testPhone, "action": "CONFIRM",
				"appointment_id": testAppointmentID,
			})
			if code != http.StatusConflict || decodeField(t, body, "error") != "appointment_not_updatable" {
				t.Fatalf("status=%d body=%s", code, body)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func newWhatsAppWebhookTestServer(t *testing.T) (*http.ServeMux, sqlmock.Sqlmock, func()) {
	t.Helper()
	t.Setenv("WHATSAPP_GATEWAY_KEY", "")
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	h := NewWhatsAppWebhookHandler(service.NewAgendaService(sqlx.NewDb(rawDB, "sqlmock")), nil)
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+callbackRoute, h.Callback)
	mux.HandleFunc("POST "+gatewayRoute, h.Gateway)
	return mux, mock, func() { _ = rawDB.Close() }
}

func postWhatsAppWebhook(t *testing.T, mux *http.ServeMux, route string, payload map[string]any) (int, string) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, route, bytes.NewReader(raw))
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)
	return recorder.Code, recorder.Body.String()
}

func decodeField(t *testing.T, body, field string) string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatal(err)
	}
	return out[field]
}

func expectAppointmentLookup(mock sqlmock.Sqlmock, status, confirmation string) {
	rows := sqlmock.NewRows([]string{
		"id", "estabelecimento_id", "data_hora_inicio", "status",
		"confirmacao_cliente", "janela_minima_cancelamento_horas",
		"motivo_cancelamento_obrigatorio", "telefone_contato",
	}).AddRow(testAppointmentID, testTenantID, time.Now().Add(24*time.Hour), status,
		confirmation, 2, false, "5511000000000")
	mock.ExpectQuery("(?s)SELECT.+FROM agendamentos a.+FOR UPDATE OF a").
		WithArgs(testTenantID, testPhone, testAppointmentID).
		WillReturnRows(rows)
}
