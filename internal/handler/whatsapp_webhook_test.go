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
		service.ErrCancellationReasonRequired:                 {http.StatusBadRequest, "reason_required"},
		service.ErrCancellationWindowClosed:                   {http.StatusUnprocessableEntity, "cancellation_window_closed"},
		service.ErrAppointmentNotCancellable:                  {http.StatusConflict, "appointment_not_cancellable"},
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

func TestWhatsAppWebhookCancelMapsDomainErrorsOnCallbackAndGateway(t *testing.T) {
	cases := []struct {
		name           string
		status         string
		startsAt       time.Time
		reasonRequired bool
		wantStatus     int
		wantCode       string
	}{
		{
			name: "reason_required", status: "AGENDADO", startsAt: time.Now().Add(24 * time.Hour),
			reasonRequired: true, wantStatus: http.StatusBadRequest, wantCode: "reason_required",
		},
		{
			name: "window_closed", status: "AGENDADO", startsAt: time.Now().Add(time.Hour),
			wantStatus: http.StatusUnprocessableEntity, wantCode: "cancellation_window_closed",
		},
		{
			name: "not_cancellable", status: "CONCLUIDO", startsAt: time.Now().Add(24 * time.Hour),
			wantStatus: http.StatusConflict, wantCode: "appointment_not_cancellable",
		},
	}
	for _, route := range []string{callbackRoute, gatewayRoute} {
		for _, tc := range cases {
			t.Run(route+"_"+tc.name, func(t *testing.T) {
				mux, mock, cleanup := newWhatsAppWebhookTestServer(t)
				defer cleanup()
				mock.ExpectBegin()
				expectAppointmentLookupWithPolicy(mock, tc.status, "PENDENTE", tc.startsAt, tc.reasonRequired)
				mock.ExpectRollback()

				code, body := postWhatsAppWebhook(t, mux, route, map[string]any{
					"sistema_origem": "beleza", "tenant_id": testTenantID,
					"phone_number": testPhone, "event_type": "button_reply",
					"action": "CANCEL", "appointment_id": testAppointmentID,
				})
				if code != tc.wantStatus || decodeField(t, body, "error") != tc.wantCode {
					t.Fatalf("status=%d body=%s", code, body)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
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
	expectAppointmentLookupWithPolicy(mock, status, confirmation, time.Now().Add(24*time.Hour), false)
}

func expectAppointmentLookupWithPolicy(
	mock sqlmock.Sqlmock,
	status, confirmation string,
	startsAt time.Time,
	reasonRequired bool,
) {
	rows := sqlmock.NewRows([]string{
		"id", "estabelecimento_id", "data_hora_inicio", "status",
		"confirmacao_cliente", "janela_minima_cancelamento_horas",
		"motivo_cancelamento_obrigatorio", "telefone_contato",
	}).AddRow(testAppointmentID, testTenantID, startsAt, status,
		confirmation, 2, reasonRequired, "5511000000000")
	mock.ExpectQuery("(?s)SELECT.+FROM agendamentos a.+FOR UPDATE OF a").
		WithArgs(testTenantID, testPhone, testAppointmentID).
		WillReturnRows(rows)
}
