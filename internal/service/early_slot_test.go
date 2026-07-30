package service

import (
	"bytes"
	"context"
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
}

// Cenário 19: canal desligado no momento do cancelamento — o slot é liberado,
// nenhuma rodada nasce e o cancelamento continua retornando sucesso.
func TestCancelWithChannelOffReleasesSlotWithoutOpeningRound(t *testing.T) {
	svc, mock := newEarlySlotServiceForTest(t,
		func(context.Context, sqlx.QueryerContext, string) (bool, error) { return false, nil })

	expectCancelledAppointment(mock)
	// Nenhum INSERT em rodadas_antecipacao é esperado entre o UPDATE e o COMMIT.
	mock.ExpectCommit()

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
	mock.ExpectCommit()

	if err := svc.CancelAppointment(context.Background(), "tenant-1", "agendamento-1"); err != nil {
		t.Fatalf("falha na checagem não pode falhar o cancelamento: %v", err)
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
