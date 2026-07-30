package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// Recusa registrada depois de `expira_em` não pode virar RECUSADA: a janela
// exclusiva já acabou e quem decide o destino da oferta é o worker.
func TestDeclineAfterExpiryIsRejectedWithoutMutatingOffer(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "estabelecimento_id", "rodada_id", "expira_em",
		}).AddRow("oferta-1", "PENDENTE", "tenant-1", "rodada-1", now.Add(-time.Second)))
	mock.ExpectRollback()

	status, err := svc.Decline(context.Background(), "token-expirado", "WEB")
	if !errors.Is(err, ErrEarlySlotOfferExpired) {
		t.Fatalf("recusa fora do prazo deve retornar offer_expired, recebido: %v (status=%q)", err, status)
	}
	// Nenhum UPDATE foi expectado: a oferta permanece intacta para o worker.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Dentro do prazo a recusa continua avançando a fila imediatamente.
func TestDeclineWithinWindowAdvancesQueue(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	slotStart := now.Add(2 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "status", "estabelecimento_id", "rodada_id", "expira_em",
		}).AddRow("oferta-1", "PENDENTE", "tenant-1", "rodada-1", now.Add(3*time.Minute)))
	mock.ExpectExec(regexp.QuoteMeta(`SET status='RECUSADA'`)).
		WithArgs("oferta-1", "tenant-1", "WEB").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM rodadas_antecipacao`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profissional_id", "slot_inicio", "slot_fim",
		}).AddRow("rodada-1", "prof-1", slotStart, slotEnd))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM agendamentos a`)).
		WithArgs("tenant-1", "rodada-1", "prof-1", slotStart, slotEnd).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rodadas_antecipacao SET status = 'ESGOTADA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	status, err := svc.Decline(context.Background(), "token-valido", "WEB")
	if err != nil {
		t.Fatal(err)
	}
	if status != "RECUSADA" {
		t.Fatalf("status = %q, esperado RECUSADA", status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func pendingRetryRow(currentStart, slotStart time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "estabelecimento_id", "rodada_id", "agendamento_candidato_id",
		"tentativas_envio", "cliente_nome", "cliente_telefone",
		"salon_name", "professional_name", "data_hora_inicio", "slot_inicio",
	}).AddRow(
		"oferta-1", "tenant-1", "rodada-1", "agendamento-1",
		1, "Maria", "5511999999999",
		"Glow", "Ana", currentStart, slotStart,
	)
}

// O gate é reavaliado antes da segunda tentativa: se o canal caiu durante os
// 45s de espera, a mensagem não sai e a rodada é encerrada ali mesmo, sem
// depender da ordem em que o worker chama as varreduras.
func TestSendRetryWithChannelDownCancelsRoundWithoutSending(t *testing.T) {
	// O helper falha o teste em qualquer chamada ao Gateway, então o próprio
	// setup já prova "zero chamadas".
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return false, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(now, earlySlotMaxSendAttempts).
		WillReturnRows(pendingRetryRow(now.Add(48*time.Hour), now.Add(24*time.Hour)))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='INVALIDADA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`SET status='CANCELADA'`)).
		WithArgs("rodada-1", "tenant-1", earlySlotMotivoCanalIndisponivel).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	// Iteração seguinte não encontra mais nada para reenviar.
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(now, earlySlotMaxSendAttempts).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	retried, err := svc.ProcessSendRetries(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if retried != 0 {
		t.Fatalf("reenvios = %d, esperado 0 com canal indisponível", retried)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Erro na checagem do gate é fail-closed: nada é enviado e nada é escrito, para
// a oferta continuar agendada e ser reavaliada no próximo ciclo.
func TestSendRetryWithGateFailureKeepsOfferUntouched(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) {
			return false, errors.New("timeout consultando canal")
		})
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(now, earlySlotMaxSendAttempts).
		WillReturnRows(pendingRetryRow(now.Add(48*time.Hour), now.Add(24*time.Hour)))
	// Sem UPDATE e sem commit: a transação inteira é descartada.
	mock.ExpectRollback()

	retried, err := svc.ProcessSendRetries(context.Background(), 50)
	if err != nil {
		t.Fatalf("erro de checagem não deve abortar a varredura: %v", err)
	}
	if retried != 0 {
		t.Fatalf("reenvios = %d, esperado 0 (fail-closed)", retried)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Candidato reagendado depois da oferta: mesmo que o horário novo continue
// elegível, a oferta antiga não pode sobrescrevê-lo. Ela é invalidada e a
// rodada segue para o próximo.
func TestAcceptInvalidatesOfferWhenCandidateWasRescheduled(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	slotStart := now.Add(24 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)
	// Retrato da oferta: 48h. O cliente remarcou para 72h — ainda posterior ao
	// slot, mesmo profissional e dentro do expediente, então `eligible` é true.
	originalStart := now.Add(48 * time.Hour)
	originalEnd := originalStart.Add(45 * time.Minute)
	currentStart := now.Add(72 * time.Hour)
	currentEnd := currentStart.Add(45 * time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"offer_id", "offer_status", "expira_em", "rodada_id", "round_status",
			"estabelecimento_id", "agendamento_id", "profissional_id",
			"slot_inicio", "slot_fim", "current_start", "current_end",
			"agendamento_inicio_original", "agendamento_fim_original",
			"cliente_nome", "cliente_telefone", "profissional_nome", "servico_nome",
			"eligible",
		}).AddRow(
			"oferta-1", "PENDENTE", now.Add(3*time.Minute), "rodada-1", "ATIVA",
			"tenant-1", "agendamento-1", "prof-1",
			slotStart, slotEnd, currentStart, currentEnd,
			originalStart, originalEnd,
			"Maria", "5511999999999", "Ana", "Corte", true,
		))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='INVALIDADA'`)).
		WithArgs("oferta-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM rodadas_antecipacao`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profissional_id", "slot_inicio", "slot_fim",
		}).AddRow("rodada-1", "prof-1", slotStart, slotEnd))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM agendamentos a`)).
		WithArgs("tenant-1", "rodada-1", "prof-1", slotStart, slotEnd).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rodadas_antecipacao SET status = 'ESGOTADA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err := svc.Accept(context.Background(), "token-oferta-velha", "WEB")
	if !errors.Is(err, ErrEarlySlotAppointmentIneligible) {
		t.Fatalf("aceite sobre candidato reagendado deve ser recusado, recebido: %v", err)
	}
	// A ausência de `UPDATE agendamentos SET data_hora_inicio` na lista estrita
	// é a prova de que o reagendamento do cliente não foi sobrescrito.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Contrapartida: retrato igual ao estado atual não bloqueia o aceite.
func TestRescheduleDetectionIgnoresUnchangedAndMissingSnapshots(t *testing.T) {
	start := time.Date(2026, 8, 5, 14, 0, 0, 0, time.UTC)
	end := start.Add(45 * time.Minute)

	if rescheduledSinceOffer(sql.NullTime{Time: start, Valid: true},
		sql.NullTime{Time: end, Valid: true}, start, end) {
		t.Fatal("retrato idêntico não pode ser lido como reagendamento")
	}
	// Ofertas anteriores à migração 000022 não têm retrato e seguem o fluxo
	// normal, sem serem invalidadas por falta de dado.
	if rescheduledSinceOffer(sql.NullTime{}, sql.NullTime{}, start, end) {
		t.Fatal("ausência de retrato não pode ser lida como reagendamento")
	}
	if !rescheduledSinceOffer(sql.NullTime{Time: start, Valid: true},
		sql.NullTime{Time: end, Valid: true}, start.Add(time.Hour), end.Add(time.Hour)) {
		t.Fatal("horário diferente do retrato deve ser reagendamento")
	}
}

func failingDelivery(offerID string) *earlySlotDelivery {
	return &earlySlotDelivery{
		OfferID: offerID, TenantID: "tenant-1", AppointmentID: "agendamento-1",
		Phone: "5511999999999", ClientName: "Maria", SalonName: "Glow",
		Professional: "Ana", Token: "token-oferta",
	}
}

// Primeira falha de envio não descarta o candidato: a oferta segue PENDENTE com
// a segunda tentativa agendada.
func TestFirstSendFailureSchedulesRetryInsteadOfFailing(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.send = func(context.Context, WhatsAppNotificationInput) error {
		return errors.New("gateway indisponível")
	}

	mock.ExpectExec(regexp.QuoteMeta(`SET proxima_tentativa_em=$3`)).
		WithArgs("oferta-1", "tenant-1", now.Add(earlySlotSendRetryDelay)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	delivery := failingDelivery("oferta-1")
	delivery.Attempt = 1
	svc.deliverOrAdvance(context.Background(), delivery)

	// Sem FALHA_ENVIO e sem avanço de fila nesta tentativa.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Esgotadas as duas tentativas, aí sim a oferta vira FALHA_ENVIO e a fila anda:
// um Gateway fora do ar não pode travar o slot.
func TestSecondSendFailureMarksFailureAndAdvances(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.send = func(context.Context, WhatsAppNotificationInput) error {
		return errors.New("gateway indisponível")
	}
	slotStart := now.Add(2 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SET status = 'FALHA_ENVIO'`)).
		WithArgs("oferta-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"rodada_id"}).AddRow("rodada-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM rodadas_antecipacao`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profissional_id", "slot_inicio", "slot_fim",
		}).AddRow("rodada-1", "prof-1", slotStart, slotEnd))
	mock.ExpectQuery(regexp.QuoteMeta(`FROM agendamentos a`)).
		WithArgs("tenant-1", "rodada-1", "prof-1", slotStart, slotEnd).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rodadas_antecipacao SET status = 'ESGOTADA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	delivery := failingDelivery("oferta-1")
	delivery.Attempt = earlySlotMaxSendAttempts
	svc.deliverOrAdvance(context.Background(), delivery)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// A retentativa reemite o link com token novo: o primeiro nunca chegou ao
// cliente e só o hash é persistido, então não há como reenviar o anterior.
func TestSendRetryRotatesTokenAndResends(t *testing.T) {
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
		WillReturnRows(pendingRetryRow(currentStart, slotStart))
	mock.ExpectExec(regexp.QuoteMeta(`SET token_hash=$3, tentativas_envio=tentativas_envio+1`)).
		WithArgs("oferta-1", "tenant-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	// Segunda iteração do laço: não há mais nada para reenviar.
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
		t.Fatalf("reenvios = %d, esperado 1", retried)
	}
	if sent == nil {
		t.Fatal("a segunda tentativa deve chamar o Gateway")
	}
	if sent.TemplateName != earlySlotTemplate || sent.TenantID != "tenant-1" {
		t.Fatalf("reenvio fora do contrato: %#v", sent)
	}
	link := sent.Variables[len(sent.Variables)-1]
	if strings.Contains(link, "token-oferta") || !strings.Contains(link, "/p/antecipacao/") {
		t.Fatalf("reenvio deve carregar um token novo: %q", link)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
