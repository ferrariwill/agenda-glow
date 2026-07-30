package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// newEarlySlotServiceForTest injeta um canal controlado pelo teste no lugar da
// consulta real, para exercitar o gate sem depender do Postgres.
func newEarlySlotServiceForTest(
	t *testing.T,
	channel func(context.Context, sqlx.QueryerContext, string) (bool, error),
) (*EarlySlotService, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "http://localhost:8081")
	svc.channelReady = channel
	svc.send = func(context.Context, WhatsAppNotificationInput) error {
		t.Fatal("nenhuma notificação deve ser enviada com o canal indisponível")
		return nil
	}
	return svc, mock
}

func expectCancelledAppointment(mock sqlmock.Sqlmock) {
	inicio := time.Date(2026, 8, 3, 14, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE agendamentos`)).
		WithArgs("agendamento-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profissional_id", "data_hora_inicio", "data_hora_fim",
		}).AddRow("agendamento-1", "prof-1", inicio, inicio.Add(45*time.Minute)))
	mock.ExpectCommit()
	// Só depois do commit o hook relê o cancelado e avalia o canal.
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM agendamentos`)).
		WithArgs("agendamento-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"profissional_id", "data_hora_inicio", "data_hora_fim",
		}).AddRow("prof-1", inicio, inicio.Add(45*time.Minute)))
	mock.ExpectCommit()
}

// Cenário 19: canal desligado no momento do cancelamento — o slot é liberado,
// nenhuma rodada nasce e o cancelamento continua retornando sucesso.
func TestCancelWithChannelOffReleasesSlotWithoutOpeningRound(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return false, nil })

	expectCancelledAppointment(mock)

	if err := svc.CancelAppointment(context.Background(), "tenant-1", "agendamento-1"); err != nil {
		t.Fatalf("indisponibilidade de canal não pode falhar o cancelamento: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 23: erro/timeout na checagem do canal é fail-closed — não abre rodada
// e ainda assim não derruba o cancelamento já efetivado.
func TestCancelWithChannelCheckFailureIsFailClosed(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) {
			return false, context.DeadlineExceeded
		})

	expectCancelledAppointment(mock)

	if err := svc.CancelAppointment(context.Background(), "tenant-1", "agendamento-1"); err != nil {
		t.Fatalf("falha na checagem não pode falhar o cancelamento: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestQueueHookFailureAfterCommitDoesNotRollbackCancellation(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })

	inicio := time.Date(2026, 8, 3, 14, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE agendamentos`)).
		WithArgs("agendamento-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "profissional_id", "data_hora_inicio", "data_hora_fim",
		}).AddRow("agendamento-1", "prof-1", inicio, inicio.Add(45*time.Minute)))
	mock.ExpectCommit()
	// A fila falha ao iniciar uma nova transação, depois do commit acima.
	mock.ExpectBegin().WillReturnError(errors.New("banco indisponível no hook"))

	if err := svc.CancelAppointment(context.Background(), "tenant-1", "agendamento-1"); err != nil {
		t.Fatalf("falha pós-commit da fila não pode alterar sucesso do cancelamento: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 20: canal cai com rodada ATIVA — encerra como CANCELADA/whatsapp_indisponivel,
// invalida a oferta PENDENTE e não chama o próximo candidato.
func TestChannelDropCancelsActiveRoundAndInvalidatesPendingOffer(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return false, nil })

	mock.ExpectQuery(regexp.QuoteMeta(`FROM rodadas_antecipacao r`)).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estabelecimento_id"}).
			AddRow("rodada-1", "tenant-1"))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FOR UPDATE SKIP LOCKED`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("rodada-1"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='INVALIDADA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`SET status='CANCELADA'`)).
		WithArgs("rodada-1", "tenant-1", earlySlotMotivoCanalIndisponivel).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	closed, err := svc.ProcessChannelDrops(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if closed != 1 {
		t.Fatalf("rodadas encerradas = %d, esperado 1", closed)
	}
	// Nenhuma seleção de sucessor foi emitida: o slot permanece livre.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 21: rodada já PREENCHIDA por um aceite anterior é terminal — a
// varredura só enxerga ATIVA, então a queda não reverte o reagendamento.
func TestChannelDropNeverTouchesAlreadyFilledRound(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return false, nil })

	mock.ExpectQuery(regexp.QuoteMeta(`WHERE r.status='ATIVA'`)).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estabelecimento_id"}))

	closed, err := svc.ProcessChannelDrops(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if closed != 0 {
		t.Fatalf("rodadas encerradas = %d, esperado 0", closed)
	}
	// Sem ExpectBegin: nenhuma escrita foi tentada sobre a rodada PREENCHIDA.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 22: com o canal de volta, a varredura não reabre nem reativa rodadas
// encerradas — só um novo cancelamento origina uma fila.
func TestReconnectionDoesNotResurrectClosedRounds(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })

	mock.ExpectQuery(regexp.QuoteMeta(`FROM rodadas_antecipacao r`)).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estabelecimento_id"}))

	closed, err := svc.ProcessChannelDrops(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if closed != 0 {
		t.Fatalf("rodadas encerradas = %d, esperado 0", closed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 23 (varredura): erro na checagem mantém a rodada ATIVA intocada.
func TestChannelCheckFailureKeepsRoundUntouched(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) {
			return false, errors.New("timeout consultando canal")
		})

	mock.ExpectQuery(regexp.QuoteMeta(`FROM rodadas_antecipacao r`)).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estabelecimento_id"}).
			AddRow("rodada-1", "tenant-1"))

	closed, err := svc.ProcessChannelDrops(context.Background(), 50)
	if err != nil {
		t.Fatalf("erro de checagem não deve propagar e abortar a varredura: %v", err)
	}
	if closed != 0 {
		t.Fatalf("rodadas encerradas = %d, esperado 0 (fail-closed)", closed)
	}
	// Sem ExpectBegin: nada foi escrito enquanto a checagem não é confiável.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// O gate é composto: liberação administrativa E conexão concluída.
func TestWhatsAppChannelReadyRequiresBothFlagAndConnection(t *testing.T) {
	casos := []struct {
		nome    string
		enabled bool
		status  string
		pronto  bool
	}{
		{"liberado e conectado", true, WhatsAppStatusConectado, true},
		{"liberado mas desconectado", true, WhatsAppStatusDesconectado, false},
		{"liberado mas pendente", true, WhatsAppStatusPendente, false},
		{"conectado mas sem liberação", false, WhatsAppStatusConectado, false},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			rawDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer rawDB.Close()
			db := sqlx.NewDb(rawDB, "sqlmock")

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
				WithArgs("tenant-1").
				WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(caso.enabled))
			if caso.enabled {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_status, 'DESCONECTADO')`)).
					WithArgs("tenant-1").
					WillReturnRows(sqlmock.NewRows([]string{"whatsapp_status"}).AddRow(caso.status))
			}

			pronto, err := WhatsAppChannelReadyForTenant(context.Background(), db, "tenant-1")
			if err != nil {
				t.Fatal(err)
			}
			if pronto != caso.pronto {
				t.Fatalf("canal pronto = %v, esperado %v", pronto, caso.pronto)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Erro na consulta do gate sobe para o chamador aplicar fail-closed, em vez de
// virar silenciosamente "canal disponível".
func TestWhatsAppChannelReadyPropagatesQueryFailure(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(whatsapp_enabled, FALSE)`)).
		WithArgs("tenant-1").
		WillReturnError(errors.New("conexão perdida"))

	pronto, err := WhatsAppChannelReadyForTenant(
		context.Background(), sqlx.NewDb(rawDB, "sqlmock"), "tenant-1")
	if err == nil {
		t.Fatal("erro de consulta deve ser propagado")
	}
	if pronto {
		t.Fatal("canal não pode ser considerado pronto após falha")
	}
}

func TestAcceptCommitsRescheduleBeforeSendingWhatsAppConfirmation(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()

	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "http://localhost:8081")
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	var sent *WhatsAppNotificationInput
	svc.send = func(_ context.Context, in WhatsAppNotificationInput) error {
		copy := in
		sent = &copy
		return nil
	}

	currentStart := now.Add(48 * time.Hour)
	currentEnd := currentStart.Add(45 * time.Minute)
	slotStart := now.Add(24 * time.Hour)
	slotEnd := slotStart.Add(60 * time.Minute)

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
			"oferta-1", "PENDENTE", now.Add(5*time.Minute), "rodada-1", "ATIVA",
			"tenant-1", "agendamento-1", "prof-1",
			slotStart, slotEnd, currentStart, currentEnd,
			// Retrato igual ao estado atual: o candidato não foi reagendado.
			currentStart, currentEnd,
			"Maria", "5511999999999", "Ana", "Corte", true,
		))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM agendamentos`)).
		WithArgs("tenant-1", "prof-1", "agendamento-1", slotStart, slotStart.Add(45*time.Minute)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE agendamentos SET data_hora_inicio=$3`)).
		WithArgs("agendamento-1", "tenant-1", slotStart, slotStart.Add(45*time.Minute)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='ACEITA'`)).
		WithArgs("oferta-1", "tenant-1", "WEB").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='INVALIDADA'`)).
		WithArgs("oferta-1", "tenant-1", "rodada-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE rodadas_antecipacao SET status='PREENCHIDA'`)).
		WithArgs("rodada-1", "tenant-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO antecipacao_auditoria`)).
		WithArgs("tenant-1", "rodada-1", "oferta-1",
			currentStart, currentEnd, slotStart, slotStart.Add(45*time.Minute), "WEB").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := svc.Accept(context.Background(), "token-opaco", "WEB")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ACEITA" || result.NewStart != slotStart {
		t.Fatalf("resultado inesperado: %#v", result)
	}
	if sent == nil {
		t.Fatal("aceite deve enviar confirmação WhatsApp após o commit")
	}
	if sent.TenantID != "tenant-1" ||
		sent.AppointmentID != "agendamento-1" ||
		sent.TemplateName != earlySlotConfirmationTemplate {
		t.Fatalf("confirmação fora do escopo/contrato: %#v", sent)
	}
	// A lista estrita também comprova que o aceite não cria fluxo_caixa nem
	// comissão: qualquer statement financeiro faria o sqlmock falhar.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAcceptanceConfirmationFailureDoesNotRevertCommittedReschedule(t *testing.T) {
	svc := &EarlySlotService{
		send: func(context.Context, WhatsAppNotificationInput) error {
			return errors.New("gateway indisponível")
		},
	}
	// O helper é deliberadamente best-effort e não retorna erro: ele só é
	// chamado após o commit do aceite.
	svc.sendAcceptanceConfirmation(
		context.Background(),
		"tenant-1", "agendamento-1", "5511999999999",
		"Maria", "Ana", "Corte", time.Now(),
	)
}

func TestExpiredOfferAdvancesAndExhaustsRoundAtomically(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return true, nil })
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	slotStart := now.Add(time.Hour)
	slotEnd := slotStart.Add(time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao`)).
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "estabelecimento_id", "rodada_id"}).
			AddRow("oferta-1", "tenant-1", "rodada-1"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='EXPIRADA'`)).
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
	// Próxima iteração não encontra outra oferta expirada.
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao`)).
		WithArgs(now).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	processed, err := svc.ProcessExpired(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if processed != 1 {
		t.Fatalf("ofertas processadas = %d, esperado 1", processed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCrossTenantRoundLookupReturnsNotFoundWithoutLeak(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()
	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")

	mock.ExpectQuery(regexp.QuoteMeta(`WHERE id=$1 AND estabelecimento_id=$2`)).
		WithArgs("rodada-tenant-a", "tenant-b").
		WillReturnError(sql.ErrNoRows)

	_, err = svc.GetRound(context.Background(), "tenant-b", "rodada-tenant-a")
	if !errors.Is(err, ErrEarlySlotOfferNotFound) {
		t.Fatalf("lookup cruzado deve ser 404 estável, recebido: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCompetitiveLoserCannotMutateFilledRound(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close()
	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
	now := time.Now()
	svc.now = func() time.Time { return now }

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"offer_id", "offer_status", "expira_em", "rodada_id", "round_status",
			"estabelecimento_id", "agendamento_id", "profissional_id",
			"slot_inicio", "slot_fim", "current_start", "current_end",
			"cliente_nome", "cliente_telefone", "profissional_nome", "servico_nome",
			"eligible",
		}).AddRow(
			"oferta-2", "INVALIDADA", now.Add(time.Minute), "rodada-1", "PREENCHIDA",
			"tenant-1", "agendamento-2", "prof-1",
			now.Add(time.Hour), now.Add(2*time.Hour), now.Add(3*time.Hour), now.Add(4*time.Hour),
			"Joana", "5511888888888", "Ana", "Corte", true,
		))
	mock.ExpectRollback()

	_, err = svc.Accept(context.Background(), "token-perdedor", "WEB")
	if !errors.Is(err, ErrEarlySlotOfferUnavailable) {
		t.Fatalf("segundo competidor deve perder com 409 estável, recebido: %v", err)
	}
	// Nenhum UPDATE é esperado depois do lock revelar a rodada preenchida.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNewEarlySlotTokenIsOpaqueHashedAndUnique(t *testing.T) {
	tokenA, hashA, err := newEarlySlotToken()
	if err != nil {
		t.Fatal(err)
	}
	tokenB, hashB, err := newEarlySlotToken()
	if err != nil {
		t.Fatal(err)
	}
	if tokenA == "" || tokenB == "" || tokenA == tokenB {
		t.Fatalf("tokens must be non-empty and unique")
	}
	if bytes.Equal([]byte(tokenA), hashA[:]) {
		t.Fatal("plaintext token must not be persisted as its hash")
	}
	if hashEarlySlotToken(tokenA) != hashA {
		t.Fatal("lookup hash must equal persisted SHA-256")
	}
	if hashA == hashB {
		t.Fatal("independent tokens must have distinct hashes")
	}
}

func TestWhatsAppEarlySlotCallbackRequiresScopedOpaqueToken(t *testing.T) {
	valid := WhatsAppCallbackPayload{
		SistemaOrigem: WhatsAppSistemaBeleza,
		TenantID:      "tenant-a",
		PhoneNumber:   "5511999999999",
		Action:        WhatsAppActionEarlySlotAccept,
		OfferToken:    "opaque-token",
	}
	if err := valid.validate(); err != nil {
		t.Fatalf("valid callback rejected: %v", err)
	}

	withoutToken := valid
	withoutToken.OfferToken = ""
	if err := withoutToken.validate(); err == nil {
		t.Fatal("callback without offer_token must be rejected")
	}

	withoutTenant := valid
	withoutTenant.TenantID = ""
	if err := withoutTenant.validate(); err == nil {
		t.Fatal("callback without tenant_id must be rejected")
	}
}
