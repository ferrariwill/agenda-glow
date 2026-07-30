package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestValidateNotificationAgendaConfig(t *testing.T) {
	valid := NotificationAgendaConfig{
		AntecedenciaConfirmacaoHoras:  24,
		AntecedenciaLembreteHoras:     1,
		JanelaMinimaCancelamentoHoras: 2,
		TemplateConfirmacao:           "Olá {{nome_salao}}, {{link_gestao}}",
		TemplateLembrete:              "{{servico}} com {{profissional}} em {{data_hora}}",
	}
	if err := ValidateNotificationAgendaConfig(valid); err != nil {
		t.Fatalf("configuração válida rejeitada: %v", err)
	}

	cases := []NotificationAgendaConfig{
		func() NotificationAgendaConfig { c := valid; c.AntecedenciaConfirmacaoHoras = 169; return c }(),
		func() NotificationAgendaConfig { c := valid; c.AntecedenciaLembreteHoras = 0; return c }(),
		func() NotificationAgendaConfig { c := valid; c.TemplateLembrete = "{{token}}"; return c }(),
		func() NotificationAgendaConfig { c := valid; c.TemplateConfirmacao = "<b>Olá</b>"; return c }(),
	}
	for i, input := range cases {
		if !errors.Is(ValidateNotificationAgendaConfig(input), ErrInvalidNotificationSettings) {
			t.Fatalf("caso inválido %d aceito", i)
		}
	}
}

func TestCancellationAvailability(t *testing.T) {
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	open := cancellationAvailability("AGENDADO", now.Add(3*time.Hour), now, 2, true)
	if !open.Allowed || !open.ReasonRequired {
		t.Fatalf("janela deveria estar aberta: %+v", open)
	}
	closed := cancellationAvailability("AGENDADO", now.Add(90*time.Minute), now, 2, false)
	if closed.Allowed || closed.DenialReason == nil || *closed.DenialReason != "cancellation_window_closed" {
		t.Fatalf("janela deveria estar fechada: %+v", closed)
	}
}

func TestRetryBackoffIsBounded(t *testing.T) {
	if got := retryBackoff(1); got != time.Minute {
		t.Fatalf("primeiro retry: got %s", got)
	}
	if got := retryBackoff(20); got != time.Hour {
		t.Fatalf("backoff deveria limitar em 1h: got %s", got)
	}
}

func TestCancelManagementAppointmentValidationsAndIdempotency(t *testing.T) {
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	base := managementAppointment{
		ID: "appointment-a", EstablishmentID: "tenant-a", Status: "AGENDADO",
		CustomerConfirmation: "PENDENTE", StartsAt: now.Add(4 * time.Hour),
		MinimumCancellationNoticeHours: 2, ReasonRequired: true,
	}
	origin := CancellationOriginPublicManagement
	if _, err := cancelManagementAppointment(context.Background(), nil, &base, "", origin, now); !errors.Is(err, ErrCancellationReasonRequired) {
		t.Fatalf("motivo obrigatório: %v", err)
	}
	closed := base
	closed.ReasonRequired = false
	closed.StartsAt = now.Add(time.Hour)
	if _, err := cancelManagementAppointment(context.Background(), nil, &closed, "", origin, now); !errors.Is(err, ErrCancellationWindowClosed) {
		t.Fatalf("janela fechada: %v", err)
	}
	completed := base
	completed.ReasonRequired = false
	completed.Status = "CONCLUIDO"
	if _, err := cancelManagementAppointment(context.Background(), nil, &completed, "", origin, now); !errors.Is(err, ErrAppointmentNotCancellable) {
		t.Fatalf("concluído deveria ser preservado: %v", err)
	}
	repeated := base
	repeated.Status = "CANCELADO"
	repeated.CustomerConfirmation = "CANCELADO_CLIENTE"
	result, err := cancelManagementAppointment(context.Background(), nil, &repeated, "", origin, now)
	if err != nil || result.SlotReleased {
		t.Fatalf("repetição deveria ser idempotente: result=%+v err=%v", result, err)
	}
}

func TestCancelManagementAppointmentPersistsAndEnqueues(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "sqlmock")
	mock.ExpectBegin()
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	ag := managementAppointment{
		ID: "appointment-a", EstablishmentID: "tenant-a", Status: "AGENDADO",
		CustomerConfirmation: "PENDENTE", StartsAt: now.Add(4 * time.Hour),
		MinimumCancellationNoticeHours: 2,
	}
	expectCancellationWrites(mock, "Imprevisto", CancellationOriginPublicManagement, now)
	mock.ExpectCommit()

	result, err := cancelManagementAppointment(context.Background(), tx, &ag, "Imprevisto", CancellationOriginPublicManagement, now)
	if err != nil || !result.SlotReleased {
		t.Fatalf("cancelamento: result=%+v err=%v", result, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// expectCancellationWrites fixa a ordem do cancelamento: atualização do agendamento,
// linha terminal de auditoria do evento do cliente e, separadamente, a confirmação
// de saída pendente.
func expectCancellationWrites(mock sqlmock.Sqlmock, reason, origin string, now time.Time) {
	mock.ExpectExec("UPDATE agendamentos[\\s\\S]+CANCELADO_CLIENTE").
		WithArgs("appointment-a", "tenant-a", reason).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WithArgs("tenant-a", "appointment-a", NotificationTypeCancellationByCustomer,
			NotificationStatusRecorded, now, safeAuditSummary{reason: reason, origin: origin}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// A confirmação de saída é enfileirada com o relógio do chamador, nunca com NOW().
	mock.ExpectExec("INSERT INTO agendamento_notificacoes").
		WithArgs("tenant-a", "appointment-a", NotificationTypeCancellationConfirmed,
			NotificationStatusPending, now).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

// safeAuditSummary valida o resumo persistido: registra que houve motivo, mas nunca
// o texto do motivo, telefone ou token de gestão.
type safeAuditSummary struct {
	reason string
	origin string
}

func (s safeAuditSummary) Match(value driver.Value) bool {
	raw, ok := value.([]byte)
	if !ok {
		text, isString := value.(string)
		if !isString {
			return false
		}
		raw = []byte(text)
	}
	if s.reason != "" && strings.Contains(string(raw), s.reason) {
		return false
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false
	}
	return parsed["evento"] == "cancelamento_cliente" &&
		parsed["origem"] == s.origin &&
		parsed["motivo_informado"] == (s.reason != "")
}

func TestEligibleAppointmentStatusesExcludeEmAprovacao(t *testing.T) {
	scheduled := []string{
		NotificationTypeReservationConfirmation,
		NotificationTypeConfirmationRequest,
		NotificationTypeReminder,
	}
	for _, notificationType := range scheduled {
		got := EligibleAppointmentStatuses(notificationType)
		if !reflect.DeepEqual(got, []string{"AGENDADO", "CONFIRMADO"}) {
			t.Fatalf("%s: got %v, want [AGENDADO CONFIRMADO]", notificationType, got)
		}
	}
	got := EligibleAppointmentStatuses(NotificationTypeCancellationConfirmed)
	if !reflect.DeepEqual(got, []string{"CANCELADO"}) {
		t.Fatalf("confirmação de cancelamento: got %v, want [CANCELADO]", got)
	}
}

func TestBuildManagementURL(t *testing.T) {
	cases := map[string]string{
		"http://localhost:8081":  "http://localhost:8081/p/agendamento/tok",
		"http://localhost:8081/": "http://localhost:8081/p/agendamento/tok",
		"":                       "/p/agendamento/tok",
	}
	for base, want := range cases {
		if got := BuildManagementURL(base, "tok"); got != want {
			t.Fatalf("base %q: got %q, want %q", base, got, want)
		}
	}
}

func TestAgendaNotificationWorkerRunOnceTwiceDoesNotDuplicate(t *testing.T) {
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	startsAt := now.Add(24 * time.Hour)
	requests := make(chan whatsAppSendNotificationRequest, 2)
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != whatsappSendNotificationPath {
			t.Errorf("requisição gateway: got %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != "worker-test-key" {
			t.Errorf("X-API-Key: got %q", got)
		}
		var payload whatsAppSendNotificationRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decodificar payload gateway: %v", err)
		}
		requests <- payload
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"external_message_id":"gateway-message-1"}`))
	}))
	defer gateway.Close()

	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	db := sqlx.NewDb(raw, "sqlmock")

	expectWorkerRunPrelude(mock, now)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT n.id,[\\s\\S]+FOR UPDATE OF n SKIP LOCKED").
		WithArgs(now, 5, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "estabelecimento_id", "agendamento_id", "tipo", "tentativas",
			"telefone", "nome_salao", "servico", "profissional", "data_hora_inicio",
			"endereco", "gestao_token", "template_confirmacao", "template_lembrete",
		}).AddRow(
			"notification-a", "tenant-a", "appointment-a", NotificationTypeReminder, 0,
			"5511999999999", "Studio Glow", "Corte", "Ana", startsAt,
			"Rua Um, São Paulo, SP", "management-token-a",
			"Confirme {{servico}}", "Personalizado: {{servico}} com {{profissional}} em {{data_hora}} — {{link_gestao}}",
		))
	mock.ExpectExec("UPDATE agendamento_notificacoes[\\s\\S]+status_envio = 'ENVIANDO'").
		WithArgs("notification-a", now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("UPDATE agendamento_notificacoes[\\s\\S]+status_envio = 'ENVIADO'").
		WithArgs("notification-a", now, "gateway-message-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoReservableNotification(mock, now)

	// A segunda execução repete descoberta idempotente, mas não encontra linha
	// PENDENTE/FALHOU para reservar e, portanto, não chama o Gateway.
	expectWorkerRunPrelude(mock, now)
	expectNoReservableNotification(mock, now)

	worker := NewAgendaNotificationWorker(db, &GatewayNotificationSender{
		BaseURL: gateway.URL,
		APIKey:  "worker-test-key",
		Client:  gateway.Client(),
	}, AgendaNotificationWorkerOptions{
		BaseURL:    "https://app.example",
		MaxRetries: 5,
		Now:        func() time.Time { return now },
	})

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("primeiro RunOnce: %v", err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("segundo RunOnce: %v", err)
	}
	if got := len(requests); got != 1 {
		t.Fatalf("chamadas ao Gateway: got %d, want 1", got)
	}
	payload := <-requests
	if payload.TenantID != "tenant-a" || payload.AppointmentID != "appointment-a" {
		t.Fatalf("escopo do payload: tenant=%q appointment=%q", payload.TenantID, payload.AppointmentID)
	}
	if payload.TemplateName != whatsappTemplateLembrete || payload.PhoneNumber != "5511999999999" {
		t.Fatalf("payload gateway inesperado: %+v", payload)
	}
	wantMessage := "Personalizado: Corte com Ana em 02/08/2026 10:00 — https://app.example/p/agendamento/management-token-a"
	if !reflect.DeepEqual(payload.Variables, []string{wantMessage}) {
		t.Fatalf("template persistido não aplicado: got %v, want [%q]", payload.Variables, wantMessage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAgendaNotificationWorkerStartReportsRunErrors(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	runErr := errors.New("postgres unavailable")
	mock.ExpectExec("UPDATE agendamento_notificacoes[\\s\\S]+reserva de envio expirada").
		WithArgs(now).
		WillReturnError(runErr)

	ctx, cancel := context.WithCancel(context.Background())
	var reported error
	worker := NewAgendaNotificationWorker(
		sqlx.NewDb(raw, "sqlmock"),
		nil,
		AgendaNotificationWorkerOptions{
			Now: func() time.Time { return now },
			OnError: func(err error) {
				reported = err
				cancel()
			},
		},
	)
	worker.Start(ctx)
	if !errors.Is(reported, runErr) {
		t.Fatalf("erro reportado=%v, want %v", reported, runErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectWorkerRunPrelude(mock sqlmock.Sqlmock, now time.Time) {
	mock.ExpectExec("UPDATE agendamento_notificacoes[\\s\\S]+reserva de envio expirada").
		WithArgs(now).
		WillReturnResult(sqlmock.NewResult(0, 0))
	// $1 relógio da fila, $3 horário do salão: a descoberta grava com o mesmo relógio
	// que o reserveNext usa para filtrar.
	mock.ExpectExec("INSERT INTO agendamento_notificacoes[\\s\\S]+ON CONFLICT").
		WithArgs(now, sqlmock.AnyArg(), now).
		WillReturnResult(sqlmock.NewResult(0, 0))
}

func expectNoReservableNotification(mock sqlmock.Sqlmock, now time.Time) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT n.id,[\\s\\S]+FOR UPDATE OF n SKIP LOCKED").
		WithArgs(now, 5, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
}
