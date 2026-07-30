package service

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func acceptLookupColumns() []string {
	return []string{
		"offer_id", "offer_status", "expira_em", "rodada_id", "round_status",
		"estabelecimento_id", "agendamento_id", "profissional_id",
		"slot_inicio", "slot_fim", "current_start", "current_end",
		"cliente_nome", "cliente_telefone", "profissional_nome", "servico_nome",
		"eligible",
	}
}

// Cenário 12: a revalidação do aceite acontece dentro do lock, em SQL. Estes
// predicados são o contrato — sem eles um candidato alterado depois da oferta
// conseguiria ocupar o slot da rodada.
func TestAcceptRevalidatesProfessionalRescheduleAndWorkingHours(t *testing.T) {
	required := map[string]string{
		"profissional do snapshot":   `a\.profissional_id = o\.profissional_snapshot_id`,
		"profissional da rodada":     `a\.profissional_id = r\.profissional_id`,
		"início não reagendado":      `a\.data_hora_inicio = o\.inicio_snapshot`,
		"fim não reagendado":         `a\.data_hora_fim = o\.fim_snapshot`,
		"expediente do profissional": `FROM expedientes_profissionais ep`,
		"janela de almoço":           `ep\.inicio_almoco`,
		"lock das três linhas":       `FOR UPDATE OF o, r, a`,
	}
	for nome, padrao := range required {
		if !regexp.MustCompile(padrao).MatchString(earlySlotAcceptLookupSQL) {
			t.Errorf("aceite não revalida %s (padrão %q ausente)", nome, padrao)
		}
	}
}

// Candidato que trocou de profissional depois de receber a oferta: o aceite é
// recusado e a oferta morre, mas o slot não. A lista estrita de expectativas
// prova que nenhum `UPDATE agendamentos` acontece — o slot não é ocupado por um
// agendamento que já não é o que foi convidado — e que a rodada chama o próximo.
func TestAcceptOnMutatedCandidateInvalidatesOfferAndAdvances(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close() //nolint:errcheck

	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }
	svc.channelReady = func(context.Context, sqlx.QueryerContext, string) (bool, error) {
		return true, nil
	}
	svc.send = func(context.Context, WhatsAppNotificationInput) error {
		t.Fatal("candidato inelegível não pode gerar notificação")
		return nil
	}

	slotStart := now.Add(time.Hour)
	slotEnd := now.Add(2 * time.Hour)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows(acceptLookupColumns()).AddRow(
			"oferta-1", "PENDENTE", now.Add(3*time.Minute), "rodada-1", "ATIVA",
			"tenant-1", "agendamento-1", "prof-2",
			slotStart, slotEnd,
			now.Add(5*time.Hour), now.Add(6*time.Hour),
			"Maria", "5511999999999", "Ana", "Corte", false,
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

	if _, err := svc.Accept(context.Background(), "token-alterado", "WEB"); !errors.Is(err, ErrEarlySlotAppointmentIneligible) {
		t.Fatalf("esperado appointment_no_longer_eligible, recebido: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 27: opt-in público por gestao_token persiste mesmo com o canal
// indisponível — só o sinal informa que ainda não haverá oferta.
func TestPublicPreferenceByManagementTokenPersistsWhenChannelIsDown(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer rawDB.Close() //nolint:errcheck

	svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
	svc.channelReady = func(context.Context, sqlx.QueryerContext, string) (bool, error) {
		return false, nil
	}
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	optedAt := now.Add(-time.Minute)
	token := "3f1d9b0e-5c4a-4f2b-9d7e-8a6c1b2d3e4f"

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT estabelecimento_id, status, data_hora_inicio, gestao_token_expires_at`)).
		WithArgs(token).
		WillReturnRows(sqlmock.NewRows([]string{
			"estabelecimento_id", "status", "data_hora_inicio", "gestao_token_expires_at",
		}).AddRow("tenant-1", "AGENDADO", now.Add(48*time.Hour), now.Add(72*time.Hour)))
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE agendamentos SET aceita_adiantar = $2`)).
		WithArgs(token, true, now).
		WillReturnRows(sqlmock.NewRows([]string{"aceita_adiantar_em"}).AddRow(optedAt))

	result, err := svc.SetPreferenceByManagementToken(context.Background(), token, true, now)
	if err != nil {
		t.Fatalf("opt-in deve persistir mesmo sem canal: %v", err)
	}
	if !result.AceitaAdiantar {
		t.Fatal("preferência não foi persistida")
	}
	if result.NotificationsEnabled {
		t.Fatal("sinal deve ser falso com o canal indisponível")
	}
	if result.AceitaAdiantarEm == nil || !result.AceitaAdiantarEm.Equal(optedAt) {
		t.Fatalf("aceita_adiantar_em = %v", result.AceitaAdiantarEm)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// Cenário 28: token malformado, desconhecido ou vencido é sempre 404 — não
// revela a existência do agendamento. Atendimento passado/cancelado é 422.
func TestPublicPreferenceByManagementTokenRejectsInvalidAndIneligible(t *testing.T) {
	token := "3f1d9b0e-5c4a-4f2b-9d7e-8a6c1b2d3e4f"
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		token string
		setup func(sqlmock.Sqlmock)
		want  error
	}{
		{
			name: "token malformado nem consulta o banco", token: "nao-e-uuid",
			setup: func(sqlmock.Sqlmock) {}, want: ErrEarlySlotOfferNotFound,
		},
		{
			name: "token desconhecido", token: token,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT estabelecimento_id, status`)).
					WithArgs(token).WillReturnError(sql.ErrNoRows)
			},
			want: ErrEarlySlotOfferNotFound,
		},
		{
			name: "token vencido", token: token,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT estabelecimento_id, status`)).
					WithArgs(token).
					WillReturnRows(sqlmock.NewRows([]string{
						"estabelecimento_id", "status", "data_hora_inicio", "gestao_token_expires_at",
					}).AddRow("tenant-1", "AGENDADO", now.Add(48*time.Hour), now.Add(-time.Hour)))
			},
			want: ErrEarlySlotOfferNotFound,
		},
		{
			name: "atendimento cancelado", token: token,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT estabelecimento_id, status`)).
					WithArgs(token).
					WillReturnRows(sqlmock.NewRows([]string{
						"estabelecimento_id", "status", "data_hora_inicio", "gestao_token_expires_at",
					}).AddRow("tenant-1", "CANCELADO", now.Add(48*time.Hour), now.Add(72*time.Hour)))
			},
			want: ErrEarlySlotAppointmentIneligible,
		},
		{
			name: "atendimento no passado", token: token,
			setup: func(m sqlmock.Sqlmock) {
				m.ExpectQuery(regexp.QuoteMeta(`SELECT estabelecimento_id, status`)).
					WithArgs(token).
					WillReturnRows(sqlmock.NewRows([]string{
						"estabelecimento_id", "status", "data_hora_inicio", "gestao_token_expires_at",
					}).AddRow("tenant-1", "AGENDADO", now.Add(-time.Hour), now.Add(72*time.Hour)))
			},
			want: ErrEarlySlotAppointmentIneligible,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer rawDB.Close() //nolint:errcheck
			svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
			tc.setup(mock)

			_, err = svc.SetPreferenceByManagementToken(context.Background(), tc.token, true, now)
			if !errors.Is(err, tc.want) {
				t.Fatalf("erro = %v, esperado %v", err, tc.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Dois aceites realmente concorrentes na mesma rodada. O mutex representa o
// `FOR UPDATE OF o, r, a`: quem entra primeiro vê ATIVA e preenche; o outro só
// consegue ler depois do commit e encontra a rodada PREENCHIDA.
func TestConcurrentAcceptsLeaveExactlyOneWinner(t *testing.T) {
	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	slotStart := now.Add(24 * time.Hour)
	slotEnd := slotStart.Add(time.Hour)
	currentStart := now.Add(48 * time.Hour)
	currentEnd := currentStart.Add(45 * time.Minute)
	newEnd := slotStart.Add(currentEnd.Sub(currentStart))

	var rowLock sync.Mutex
	roundFilled := false

	accept := func(offerID, appointmentID string) (*EarlySlotAcceptResult, error) {
		rawDB, mock, err := sqlmock.New()
		if err != nil {
			return nil, err
		}
		defer rawDB.Close() //nolint:errcheck
		svc := NewEarlySlotService(sqlx.NewDb(rawDB, "sqlmock"), "")
		svc.now = func() time.Time { return now }
		svc.send = func(context.Context, WhatsAppNotificationInput) error { return nil }

		rowLock.Lock()
		defer rowLock.Unlock()

		mock.ExpectBegin()
		if roundFilled {
			mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
				WithArgs(sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows(acceptLookupColumns()).AddRow(
					offerID, "INVALIDADA", now.Add(3*time.Minute), "rodada-1", "PREENCHIDA",
					"tenant-1", appointmentID, "prof-1",
					slotStart, slotEnd, currentStart, currentEnd,
					"Joana", "5511888888888", "Ana", "Corte", true,
				))
			mock.ExpectRollback()
			return svc.Accept(context.Background(), "token-"+offerID, "WEB")
		}

		mock.ExpectQuery(regexp.QuoteMeta(`FROM ofertas_antecipacao o`)).
			WithArgs(sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows(acceptLookupColumns()).AddRow(
				offerID, "PENDENTE", now.Add(3*time.Minute), "rodada-1", "ATIVA",
				"tenant-1", appointmentID, "prof-1",
				slotStart, slotEnd, currentStart, currentEnd,
				"Maria", "5511999999999", "Ana", "Corte", true,
			))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM agendamentos`)).
			WithArgs("tenant-1", "prof-1", appointmentID, slotStart, newEnd).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE agendamentos SET data_hora_inicio=$3`)).
			WithArgs(appointmentID, "tenant-1", slotStart, newEnd).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='ACEITA'`)).
			WithArgs(offerID, "tenant-1", "WEB").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE ofertas_antecipacao SET status='INVALIDADA'`)).
			WithArgs(offerID, "tenant-1", "rodada-1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE rodadas_antecipacao SET status='PREENCHIDA'`)).
			WithArgs("rodada-1", "tenant-1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO antecipacao_auditoria`)).
			WithArgs("tenant-1", "rodada-1", offerID,
				currentStart, currentEnd, slotStart, newEnd, "WEB").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		result, err := svc.Accept(context.Background(), "token-"+offerID, "WEB")
		if err == nil {
			roundFilled = true
		}
		return result, err
	}

	type outcome struct {
		result *EarlySlotAcceptResult
		err    error
	}
	results := make([]outcome, 2)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i, ids := range [][2]string{{"oferta-1", "agendamento-1"}, {"oferta-2", "agendamento-2"}} {
		wg.Add(1)
		go func(idx int, offerID, appointmentID string) {
			defer wg.Done()
			<-start
			r, err := accept(offerID, appointmentID)
			results[idx] = outcome{result: r, err: err}
		}(i, ids[0], ids[1])
	}
	close(start)
	wg.Wait()

	winners, losers := 0, 0
	for _, o := range results {
		switch {
		case o.err == nil && o.result != nil && o.result.Status == "ACEITA":
			winners++
		case errors.Is(o.err, ErrEarlySlotOfferUnavailable):
			losers++
		default:
			t.Fatalf("desfecho inesperado: result=%#v err=%v", o.result, o.err)
		}
	}
	if winners != 1 || losers != 1 {
		t.Fatalf("vencedores=%d perdedores=%d, esperado exatamente 1 de cada", winners, losers)
	}
}
