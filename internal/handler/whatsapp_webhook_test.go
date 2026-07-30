package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"

	"github.com/agendaglow/agendaglow/internal/service"
)

const whatsAppFlagQuery = "SELECT COALESCE(whatsapp_enabled, FALSE)"

const (
	testAppointmentID = "appointment-a"
	testTenantID      = "tenant-a"
	testPhone         = "5511999999999"

	callbackRoute = "/api/v1/webhook/whatsapp-callback"
	gatewayRoute  = "/api/v1/webhook/whatsapp-gateway"
)

func TestMapWhatsAppWebhookErrorPendingProfessionalApproval(t *testing.T) {
	recorder := httptest.NewRecorder()
	mapWhatsAppWebhookError(recorder, service.ErrAgendamentoAguardandoAprovacaoProfissional)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", recorder.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decodificar resposta: %v", err)
	}
	if body["error"] != "appointment_pending_professional_approval" {
		t.Fatalf("error: got %q, want appointment_pending_professional_approval", body["error"])
	}
}

// TestWhatsAppWebhookRoutesConflictOnPendingEncaixe exercita a rota HTTP inteira — mux, decode do
// corpo, serviço e mapeamento de erro — para as duas URLs que despacham handleCallbackBody.
// O teste isolado de mapWhatsAppWebhookError não pega desvio de fluxo antes do mapper nem perda da
// distinção entre "aguarda aprovação da profissional" e "não atualizável".
func TestWhatsAppWebhookRoutesConflictOnPendingEncaixe(t *testing.T) {
	cases := []struct {
		nome   string
		rota   string
		acao   string
		evento string
	}{
		{nome: "callback com action CONFIRM", rota: callbackRoute, acao: "CONFIRM"},
		{nome: "gateway com action CONFIRM", rota: gatewayRoute, acao: "CONFIRM"},
		{nome: "callback com action minuscula", rota: callbackRoute, acao: "confirm"},
		{nome: "gateway com action minuscula", rota: gatewayRoute, acao: "confirm"},
		{nome: "callback com espacos ao redor", rota: callbackRoute, acao: "  CONFIRM  "},
		{nome: "gateway com espacos ao redor", rota: gatewayRoute, acao: "  confirm  "},
		{nome: "gateway despachado por button_reply", rota: gatewayRoute, acao: "CONFIRM", evento: "button_reply"},
	}

	for _, tc := range cases {
		t.Run(tc.nome, func(t *testing.T) {
			mux, mock, cleanup := newWhatsAppWebhookTestServer(t)
			defer cleanup()

			expectAppointmentLookup(mock, "EM_APROVACAO")

			code, body := postWhatsAppWebhook(t, mux, tc.rota, map[string]any{
				"sistema_origem": "beleza",
				"tenant_id":      testTenantID,
				"phone_number":   testPhone,
				"event_type":     tc.evento,
				"text":           "APPT_CONFIRM",
				"action":         tc.acao,
				"appointment_id": testAppointmentID,
			})

			if code != http.StatusConflict {
				t.Fatalf("status HTTP: got %d, want 409 (corpo: %s)", code, body)
			}
			if got := decodeField(t, body, "error"); got != "appointment_pending_professional_approval" {
				t.Fatalf("error: got %q, want appointment_pending_professional_approval", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("expectativas SQL (nenhum UPDATE esperado): %v", err)
			}
		})
	}
}

// TestWhatsAppWebhookRoutesStatusMatrix fixa, na fronteira HTTP, que cada status de agendamento
// tem sua própria resposta. Sem isso, colapsar o encaixe pendente dentro de appointment_not_updatable
// passaria despercebido: os dois são 409.
func TestWhatsAppWebhookRoutesStatusMatrix(t *testing.T) {
	cases := []struct {
		nome              string
		statusAgendamento string
		acao              string
		esperaUpdatePara  string
		wantHTTP          int
		wantErro          string
		wantStatusFinal   string
	}{
		{
			nome:              "encaixe pendente bloqueia CONFIRM",
			statusAgendamento: "EM_APROVACAO",
			acao:              "CONFIRM",
			wantHTTP:          http.StatusConflict,
			wantErro:          "appointment_pending_professional_approval",
		},
		{
			nome:              "encaixe pendente aceita CANCEL",
			statusAgendamento: "EM_APROVACAO",
			acao:              "CANCEL",
			esperaUpdatePara:  "CANCELADO",
			wantHTTP:          http.StatusOK,
			wantStatusFinal:   "CANCELADO",
		},
		{
			nome:              "cancelado nao volta com CONFIRM",
			statusAgendamento: "CANCELADO",
			acao:              "CONFIRM",
			wantHTTP:          http.StatusConflict,
			wantErro:          "appointment_not_updatable",
		},
		{
			nome:              "concluido nao aceita CONFIRM",
			statusAgendamento: "CONCLUIDO",
			acao:              "CONFIRM",
			wantHTTP:          http.StatusConflict,
			wantErro:          "appointment_not_updatable",
		},
		{
			nome:              "agendado confirma normalmente",
			statusAgendamento: "AGENDADO",
			acao:              "CONFIRM",
			esperaUpdatePara:  "CONFIRMADO",
			wantHTTP:          http.StatusOK,
			wantStatusFinal:   "CONFIRMADO",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nome, func(t *testing.T) {
			mux, mock, cleanup := newWhatsAppWebhookTestServer(t)
			defer cleanup()

			expectAppointmentLookup(mock, tc.statusAgendamento)
			if tc.esperaUpdatePara != "" {
				expectAppointmentStatusUpdate(mock, tc.esperaUpdatePara)
			}

			code, body := postWhatsAppWebhook(t, mux, callbackRoute, map[string]any{
				"sistema_origem": "beleza",
				"tenant_id":      testTenantID,
				"phone_number":   testPhone,
				"action":         tc.acao,
				"appointment_id": testAppointmentID,
			})

			if code != tc.wantHTTP {
				t.Fatalf("status HTTP: got %d, want %d (corpo: %s)", code, tc.wantHTTP, body)
			}
			if tc.wantErro != "" {
				if got := decodeField(t, body, "error"); got != tc.wantErro {
					t.Fatalf("error: got %q, want %q", got, tc.wantErro)
				}
			}
			if tc.wantStatusFinal != "" {
				if got := decodeField(t, body, "appointment_status"); got != tc.wantStatusFinal {
					t.Fatalf("appointment_status: got %q, want %q", got, tc.wantStatusFinal)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("expectativas SQL: %v", err)
			}
		})
	}
}

func newWhatsAppWebhookTestServer(t *testing.T) (*http.ServeMux, sqlmock.Sqlmock, func()) {
	t.Helper()
	// O webhook aceita requisição sem chave, mas só quando a variável não impõe uma.
	t.Setenv("WHATSAPP_GATEWAY_KEY", "")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	h := NewWhatsAppWebhookHandler(service.NewAgendaService(sqlx.NewDb(rawDB, "sqlmock")), nil)

	// Mesmas rotas registradas em backend/cmd/api/main.go.
	mux := http.NewServeMux()
	mux.HandleFunc("POST "+callbackRoute, h.Callback)
	mux.HandleFunc("POST "+gatewayRoute, h.Gateway)

	return mux, mock, func() { _ = rawDB.Close() }
}

func postWhatsAppWebhook(t *testing.T, mux *http.ServeMux, rota string, payload map[string]any) (int, string) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("serializar payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, rota, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, req)

	return recorder.Code, recorder.Body.String()
}

func decodeField(t *testing.T, body, campo string) string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decodificar resposta %q: %v", body, err)
	}
	return out[campo]
}

func expectAppointmentLookup(mock sqlmock.Sqlmock, status string) {
	inicio := time.Date(2026, 7, 30, 15, 0, 0, 0, time.UTC)
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
		testAppointmentID,
		testTenantID,
		"prof-a",
		"servico-a",
		inicio,
		inicio.Add(45*time.Minute),
		status,
		"Cliente Teste",
		testPhone,
		"Corte",
		"Profissional",
		nil,
	)
	mock.ExpectQuery(regexp.QuoteMeta(`FROM agendamentos a`)).
		WithArgs(testAppointmentID).
		WillReturnRows(rows)
}

func expectAppointmentStatusUpdate(mock sqlmock.Sqlmock, statusAlvo string) {
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE agendamentos`)).
		WithArgs(testAppointmentID, statusAlvo, testTenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

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
