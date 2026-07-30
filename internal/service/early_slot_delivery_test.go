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
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "estabelecimento_id", "agendamento_candidato_id", "tentativas_envio",
			"cliente_nome", "cliente_telefone", "salon_name", "professional_name",
			"data_hora_inicio", "slot_inicio",
		}).AddRow(
			"oferta-1", "tenant-1", "agendamento-1", 1,
			"Maria", "5511999999999", "Glow", "Ana",
			currentStart, slotStart,
		))
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
