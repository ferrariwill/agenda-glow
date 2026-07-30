package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	earlySlotOfferTTL             = 5 * time.Minute
	earlySlotTemplate             = "antecipacao_horario"
	earlySlotConfirmationTemplate = "confirmacao_antecipacao"

	// Único motivo de encerramento não natural de uma rodada: o canal de saída
	// caiu, então continuar ofertando prenderia o slot sem ninguém receber.
	earlySlotMotivoCanalIndisponivel = "whatsapp_indisponivel"

	// Uma oferta só é dada como não entregue depois de duas tentativas
	// espaçadas. A segunda fica com o worker: no request ela bloquearia a
	// resposta, e em memória se perderia num restart da API.
	earlySlotMaxSendAttempts = 2
	earlySlotSendRetryDelay  = 45 * time.Second
)

// earlySlotAcceptLookupSQL trava oferta, rodada e agendamento na mesma linha do
// tempo e decide `eligible` dentro do banco, para que a revalidação do cenário
// 12 não dependa de leitura feita fora do lock. Identidade do cliente vem de
// `clientes`: `agendamentos.cliente_nome` e `cliente_telefone` foram removidos
// na migração 000004.
const earlySlotAcceptLookupSQL = `
SELECT o.id AS offer_id, o.status AS offer_status, o.expira_em, r.id AS rodada_id,
       r.status AS round_status, o.estabelecimento_id, a.id AS agendamento_id,
       a.profissional_id, r.slot_inicio, r.slot_fim,
       a.data_hora_inicio AS current_start, a.data_hora_fim AS current_end,
       c.nome AS cliente_nome, c.telefone AS cliente_telefone,
       p.nome AS profissional_nome, sv.nome AS servico_nome,
       (a.aceita_adiantar AND a.status IN ('AGENDADO','CONFIRMADO')
        AND a.data_hora_inicio > r.slot_inicio
        AND (a.data_hora_fim-a.data_hora_inicio) <= (r.slot_fim-r.slot_inicio)
        -- Cenário 12: o candidato precisa ser exatamente o mesmo que recebeu a
        -- oferta. Troca de profissional ou reagendamento no meio dos 5 minutos
        -- invalidam o aceite em vez de mover o agendamento errado para o slot.
        AND a.profissional_id = o.profissional_snapshot_id
        AND a.profissional_id = r.profissional_id
        AND a.data_hora_inicio = o.inicio_snapshot
        AND a.data_hora_fim = o.fim_snapshot
        -- O expediente pode ter mudado depois da oferta; o slot precisa caber
        -- na jornada do profissional e ficar fora do almoço.
        AND EXISTS (
            SELECT 1 FROM expedientes_profissionais ep
            WHERE ep.profissional_id = r.profissional_id
              AND ep.dia_semana = EXTRACT(DOW FROM r.slot_inicio)::int
              AND r.slot_inicio::time >= ep.horario_entrada
              AND (r.slot_inicio + (a.data_hora_fim - a.data_hora_inicio))::time <= ep.horario_saida
              AND (ep.inicio_almoco IS NULL OR ep.fim_almoco IS NULL
                   OR NOT (r.slot_inicio::time < ep.fim_almoco
                           AND (r.slot_inicio + (a.data_hora_fim - a.data_hora_inicio))::time > ep.inicio_almoco)))
       ) AS eligible
FROM ofertas_antecipacao o
JOIN rodadas_antecipacao r ON r.id=o.rodada_id AND r.estabelecimento_id=o.estabelecimento_id
JOIN agendamentos a ON a.id=o.agendamento_candidato_id AND a.estabelecimento_id=o.estabelecimento_id
JOIN clientes c ON c.id=a.cliente_id AND c.estabelecimento_id=a.estabelecimento_id
JOIN profissionais p ON p.id=a.profissional_id AND p.estabelecimento_id=a.estabelecimento_id
JOIN servicos sv ON sv.id=a.servico_id AND sv.estabelecimento_id=a.estabelecimento_id
WHERE o.token_hash=$1
FOR UPDATE OF o, r, a`

// earlySlotRetryLookupQuery reúne os dados da segunda tentativa de envio.
const earlySlotRetryLookupQuery = `
SELECT o.id, o.estabelecimento_id, o.agendamento_candidato_id, o.tentativas_envio,
       c.nome AS cliente_nome, c.telefone AS cliente_telefone,
       e.nome_comercial AS salon_name, p.nome AS professional_name,
       a.data_hora_inicio, r.slot_inicio
FROM ofertas_antecipacao o
JOIN rodadas_antecipacao r ON r.id=o.rodada_id AND r.estabelecimento_id=o.estabelecimento_id
JOIN agendamentos a ON a.id=o.agendamento_candidato_id AND a.estabelecimento_id=o.estabelecimento_id
JOIN clientes c ON c.id=a.cliente_id AND c.estabelecimento_id=a.estabelecimento_id
JOIN estabelecimentos e ON e.id=o.estabelecimento_id
JOIN profissionais p ON p.id=a.profissional_id AND p.estabelecimento_id=o.estabelecimento_id
WHERE o.status='PENDENTE' AND r.status='ATIVA'
  AND o.proxima_tentativa_em IS NOT NULL AND o.proxima_tentativa_em <= $1
  AND o.tentativas_envio < $2 AND o.expira_em > $1
ORDER BY o.proxima_tentativa_em
LIMIT 1
FOR UPDATE OF o SKIP LOCKED`

var (
	ErrEarlySlotOfferNotFound         = errors.New("offer_not_found")
	ErrEarlySlotOfferExpired          = errors.New("offer_expired")
	ErrEarlySlotOfferUnavailable      = errors.New("offer_no_longer_available")
	ErrEarlySlotAppointmentIneligible = errors.New("appointment_no_longer_eligible")
)

type EarlySlotService struct {
	db           *sqlx.DB
	now          func() time.Time
	baseURL      string
	send         func(context.Context, WhatsAppNotificationInput) error
	channelReady func(context.Context, sqlx.QueryerContext, string) (bool, error)
}

type EarlySlotOfferView struct {
	Status           string    `json:"status" db:"status"`
	SalonName        string    `json:"salon_name" db:"salon_name"`
	ProfessionalName string    `json:"professional_name" db:"professional_name"`
	ServiceName      string    `json:"service_name" db:"service_name"`
	CurrentStart     time.Time `json:"current_start" db:"current_start"`
	CurrentEnd       time.Time `json:"current_end" db:"current_end"`
	OfferedStart     time.Time `json:"offered_start" db:"offered_start"`
	OfferedEnd       time.Time `json:"offered_end" db:"offered_end"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	SecondsRemaining int64     `json:"seconds_remaining"`
	ActionsAllowed   []string  `json:"actions_allowed"`
}

type EarlySlotAcceptResult struct {
	Status        string    `json:"status"`
	AppointmentID string    `json:"appointment_id"`
	NewStart      time.Time `json:"new_start"`
	NewEnd        time.Time `json:"new_end"`
}

type EarlySlotRound struct {
	ID                 string             `json:"id" db:"id"`
	Status             string             `json:"status" db:"status"`
	MotivoEncerramento *string            `json:"motivo_encerramento" db:"motivo_encerramento"`
	ProfessionalID     string             `json:"profissional_id" db:"profissional_id"`
	SlotStart          time.Time          `json:"slot_inicio" db:"slot_inicio"`
	SlotEnd            time.Time          `json:"slot_fim" db:"slot_fim"`
	Current            *EarlySlotCurrent  `json:"candidato_atual,omitempty"`
	History            []EarlySlotHistory `json:"historico"`
}

type EarlySlotCurrent struct {
	AppointmentID string    `json:"agendamento_id"`
	ClientName    string    `json:"cliente_nome"`
	Position      int       `json:"posicao"`
	OfferStatus   string    `json:"offer_status"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type EarlySlotHistory struct {
	Position  int       `json:"posicao" db:"posicao"`
	Status    string    `json:"status" db:"status"`
	SentAt    time.Time `json:"enviada_em" db:"enviada_em"`
	ExpiresAt time.Time `json:"expires_at" db:"expira_em"`
}

type earlySlotDelivery struct {
	OfferID       string
	TenantID      string
	AppointmentID string
	Phone         string
	ClientName    string
	SalonName     string
	Professional  string
	CurrentStart  time.Time
	OfferedStart  time.Time
	Token         string
	// Attempt é a tentativa que está sendo executada, contada a partir de 1.
	Attempt int
}

func NewEarlySlotService(db *sqlx.DB, baseURL string) *EarlySlotService {
	return &EarlySlotService{
		db: db, now: time.Now, baseURL: baseURL,
		send:         EnviarNotificacaoWhatsApp,
		channelReady: WhatsAppChannelReadyForTenant,
	}
}

// NotificationsAvailable expõe o mesmo gate composto usado pela fila. Erros
// são fail-closed para superfícies informativas: o payload continua válido.
func (s *EarlySlotService) NotificationsAvailable(ctx context.Context, tenantID string) bool {
	ready, err := s.channelReady(ctx, s.db, tenantID)
	if err != nil {
		log.Printf("antecipacao: checagem informativa de canal falhou tenant=%s: %v", tenantID, err)
		return false
	}
	return ready
}

// CancelAppointment cancela no escopo do tenant e só então tenta abrir a rodada.
// Repetições são idempotentes e nunca criam mais de uma rodada.
func (s *EarlySlotService) CancelAppointment(ctx context.Context, tenantID, appointmentID string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	var cancelled struct {
		ID             string    `db:"id"`
		ProfessionalID string    `db:"profissional_id"`
		Start          time.Time `db:"data_hora_inicio"`
		End            time.Time `db:"data_hora_fim"`
	}
	const cancel = `
UPDATE agendamentos
SET status = 'CANCELADO'
WHERE id = $1 AND estabelecimento_id = $2
  AND status IN ('AGENDADO', 'CONFIRMADO', 'EM_APROVACAO')
RETURNING id, profissional_id, data_hora_inicio, data_hora_fim`
	err = tx.GetContext(ctx, &cancelled, cancel, appointmentID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if getErr := tx.GetContext(ctx, &exists,
			`SELECT EXISTS (SELECT 1 FROM agendamentos WHERE id = $1 AND estabelecimento_id = $2 AND status = 'CANCELADO')`,
			appointmentID, tenantID); getErr != nil {
			return getErr
		}
		if !exists {
			return ErrAgendamentoNaoEncontrado
		}
		return tx.Commit()
	}
	if err != nil {
		return fmt.Errorf("cancelar agendamento: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Hook deliberadamente pós-commit: qualquer falha da fila (gate, banco ou
	// envio) não pode desfazer o cancelamento que liberou o slot.
	if err := s.OpenRoundForCancelledAppointment(ctx, tenantID, cancelled.ID); err != nil {
		log.Printf("antecipacao: hook pós-cancelamento falhou tenant=%s agendamento_cancelado=%s: %v",
			tenantID, cancelled.ID, err)
	}
	return nil
}

// OpenRoundForCancelledAppointment é o ponto de extensão para cancelamentos
// públicos integrados por outros fluxos.
func (s *EarlySlotService) OpenRoundForCancelledAppointment(ctx context.Context, tenantID, appointmentID string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	var a struct {
		ProfessionalID string    `db:"profissional_id"`
		Start          time.Time `db:"data_hora_inicio"`
		End            time.Time `db:"data_hora_fim"`
	}
	err = tx.GetContext(ctx, &a, `
SELECT profissional_id, data_hora_inicio, data_hora_fim
FROM agendamentos
WHERE id = $1 AND estabelecimento_id = $2 AND status = 'CANCELADO'
FOR UPDATE`, appointmentID, tenantID)
	if err != nil {
		return err
	}
	delivery, err := s.openRoundTx(ctx, tx, tenantID, appointmentID, a.ProfessionalID, a.Start, a.End)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.deliverOrAdvance(ctx, delivery)
	return nil
}

func (s *EarlySlotService) openRoundTx(ctx context.Context, tx *sqlx.Tx, tenantID, cancelledID, professionalID string, start, end time.Time) (*earlySlotDelivery, error) {
	// Sem canal de saída a fila só prenderia o slot por 5 minutos sem ninguém
	// receber a oferta. Erro de checagem é tratado como indisponível
	// (fail-closed) e nunca derruba o cancelamento, que já foi efetivado.
	ready, err := s.channelReady(ctx, tx, tenantID)
	if err != nil {
		log.Printf("antecipacao: checagem de canal falhou tenant=%s agendamento_cancelado=%s — rodada não aberta: %v",
			tenantID, cancelledID, err)
		return nil, nil
	}
	if !ready {
		log.Printf("antecipacao: canal whatsapp indisponível tenant=%s agendamento_cancelado=%s — rodada não aberta (motivo=%s)",
			tenantID, cancelledID, earlySlotMotivoCanalIndisponivel)
		return nil, nil
	}

	var roundID string
	err = tx.GetContext(ctx, &roundID, `
INSERT INTO rodadas_antecipacao
    (estabelecimento_id, profissional_id, slot_inicio, slot_fim, agendamento_cancelado_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (estabelecimento_id, agendamento_cancelado_id) DO NOTHING
RETURNING id`, tenantID, professionalID, start, end, cancelledID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("abrir rodada de antecipação: %w", err)
	}
	return s.advanceRoundTx(ctx, tx, tenantID, roundID)
}

func (s *EarlySlotService) advanceRoundTx(ctx context.Context, tx *sqlx.Tx, tenantID, roundID string) (*earlySlotDelivery, error) {
	var round struct {
		ID             string    `db:"id"`
		ProfessionalID string    `db:"profissional_id"`
		Start          time.Time `db:"slot_inicio"`
		End            time.Time `db:"slot_fim"`
	}
	err := tx.GetContext(ctx, &round, `
SELECT id, profissional_id, slot_inicio, slot_fim
FROM rodadas_antecipacao
WHERE id = $1 AND estabelecimento_id = $2 AND status = 'ATIVA'
FOR UPDATE`, roundID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Reavaliado a cada oferta, não só na abertura: o canal pode cair durante
	// os 5 minutos exclusivos do candidato anterior.
	ready, err := s.channelReady(ctx, tx, tenantID)
	if err != nil {
		// Fail-closed: não avança a fila e não encerra a rodada — um erro
		// transitório não é evidência de queda. A varredura tenta de novo.
		return nil, fmt.Errorf("checar canal whatsapp da rodada %s: %w", roundID, err)
	}
	if !ready {
		if err := s.closeRoundChannelDownTx(ctx, tx, tenantID, roundID); err != nil {
			return nil, err
		}
		return nil, nil
	}

	var candidate struct {
		ID             string    `db:"id"`
		ClientID       string    `db:"cliente_id"`
		ClientName     string    `db:"cliente_nome"`
		Phone          string    `db:"cliente_telefone"`
		SalonName      string    `db:"salon_name"`
		Professional   string    `db:"professional_name"`
		ProfessionalID string    `db:"profissional_id"`
		CurrentStart   time.Time `db:"data_hora_inicio"`
		CurrentEnd     time.Time `db:"data_hora_fim"`
		Position       int       `db:"posicao"`
	}
	err = tx.GetContext(ctx, &candidate, `
SELECT a.id, a.cliente_id, c.nome AS cliente_nome, c.telefone AS cliente_telefone,
       e.nome_comercial AS salon_name, p.nome AS professional_name,
       a.profissional_id, a.data_hora_inicio, a.data_hora_fim,
       (SELECT COUNT(*) + 1 FROM ofertas_antecipacao ox
        WHERE ox.rodada_id = $2) AS posicao
FROM agendamentos a
JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
JOIN estabelecimentos e ON e.id = a.estabelecimento_id
JOIN profissionais p ON p.id = a.profissional_id AND p.estabelecimento_id = a.estabelecimento_id
WHERE a.estabelecimento_id = $1
  AND a.profissional_id = $3
  AND a.aceita_adiantar = TRUE
  AND a.status IN ('AGENDADO', 'CONFIRMADO')
  AND a.data_hora_inicio > $4
  AND (a.data_hora_fim - a.data_hora_inicio) <= ($5 - $4)
  AND NOT EXISTS (
      SELECT 1 FROM ofertas_antecipacao prev
      WHERE prev.rodada_id = $2 AND prev.agendamento_candidato_id = a.id)
  AND NOT EXISTS (
      SELECT 1 FROM ofertas_antecipacao pending
      WHERE pending.estabelecimento_id = $1
        AND pending.agendamento_candidato_id = a.id
        AND pending.status = 'PENDENTE')
  AND EXISTS (
      SELECT 1 FROM expedientes_profissionais ep
      WHERE ep.profissional_id = a.profissional_id
        AND ep.dia_semana = EXTRACT(DOW FROM $4)::int
        AND $4::time >= ep.horario_entrada
        AND ($4 + (a.data_hora_fim - a.data_hora_inicio))::time <= ep.horario_saida
        AND (ep.inicio_almoco IS NULL OR ep.fim_almoco IS NULL
             OR NOT ($4::time < ep.fim_almoco
                     AND ($4 + (a.data_hora_fim - a.data_hora_inicio))::time > ep.inicio_almoco)))
  AND NOT EXISTS (
      SELECT 1 FROM agendamentos busy
      WHERE busy.estabelecimento_id = $1
        AND busy.profissional_id = $3
        AND busy.status IN ('AGENDADO', 'CONFIRMADO')
        AND busy.id <> a.id
        AND busy.data_hora_inicio < ($4 + (a.data_hora_fim - a.data_hora_inicio))
        AND busy.data_hora_fim > $4)
ORDER BY a.data_hora_inicio, a.aceita_adiantar_em ASC NULLS LAST, a.id
LIMIT 1
FOR UPDATE OF a`, tenantID, roundID, round.ProfessionalID, round.Start, round.End)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, `
UPDATE rodadas_antecipacao SET status = 'ESGOTADA', finished_at = NOW(),
    candidato_atual_agendamento_id = NULL
WHERE id = $1 AND estabelecimento_id = $2`, roundID, tenantID)
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("selecionar candidato de antecipação: %w", err)
	}

	token, tokenHash, err := newEarlySlotToken()
	if err != nil {
		return nil, err
	}
	var offerID string
	expiresAt := s.now().Add(earlySlotOfferTTL)
	err = tx.GetContext(ctx, &offerID, `
INSERT INTO ofertas_antecipacao
    (estabelecimento_id, rodada_id, agendamento_candidato_id, cliente_id,
     profissional_snapshot_id, inicio_snapshot, fim_snapshot,
     posicao, token_hash, enviada_em, expira_em, tentativas_envio)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 1)
RETURNING id`, tenantID, roundID, candidate.ID, candidate.ClientID,
		candidate.ProfessionalID, candidate.CurrentStart, candidate.CurrentEnd,
		candidate.Position, tokenHash[:], s.now(), expiresAt)
	if err != nil {
		return nil, fmt.Errorf("criar oferta de antecipação: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
UPDATE rodadas_antecipacao SET candidato_atual_agendamento_id = $3
WHERE id = $1 AND estabelecimento_id = $2`, roundID, tenantID, candidate.ID)
	if err != nil {
		return nil, err
	}
	return &earlySlotDelivery{
		OfferID: offerID, TenantID: tenantID, AppointmentID: candidate.ID,
		Phone: candidate.Phone, ClientName: candidate.ClientName, SalonName: candidate.SalonName,
		Professional: candidate.Professional, CurrentStart: candidate.CurrentStart,
		OfferedStart: round.Start, Token: token, Attempt: 1,
	}, nil
}

func newEarlySlotToken() (string, [32]byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", [32]byte{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, sha256.Sum256([]byte(token)), nil
}

func hashEarlySlotToken(token string) [32]byte { return sha256.Sum256([]byte(token)) }

func (s *EarlySlotService) deliverOrAdvance(ctx context.Context, d *earlySlotDelivery) {
	if d == nil {
		return
	}
	link := s.baseURL + "/p/antecipacao/" + d.Token
	input := WhatsAppNotificationInput{
		TenantID: d.TenantID, PhoneNumber: d.Phone, TemplateName: earlySlotTemplate,
		AppointmentID: d.AppointmentID,
		Variables: []string{d.ClientName, d.SalonName, d.Professional,
			d.CurrentStart.Format("02/01/2006 15:04"), d.OfferedStart.Format("02/01/2006 15:04"), link},
	}
	err := s.send(ctx, input)
	if err == nil {
		return
	}
	log.Printf("antecipacao: envio da oferta falhou tenant=%s oferta=%s tentativa=%d/%d: %v",
		d.TenantID, d.OfferID, d.Attempt, earlySlotMaxSendAttempts, err)

	if d.Attempt < earlySlotMaxSendAttempts {
		if err := s.scheduleSendRetry(ctx, d.TenantID, d.OfferID); err != nil {
			log.Printf("antecipacao: agendar reenvio falhou tenant=%s oferta=%s: %v",
				d.TenantID, d.OfferID, err)
		}
		return
	}
	_ = s.failOfferAndAdvance(ctx, d.TenantID, d.OfferID)
}

// scheduleSendRetry mantém a oferta PENDENTE e deixa a segunda tentativa para o
// worker. A janela de 5 minutos não é reiniciada: o slot fica exclusivo pelo
// orçamento original, não pelo número de tentativas.
func (s *EarlySlotService) scheduleSendRetry(ctx context.Context, tenantID, offerID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE ofertas_antecipacao SET proxima_tentativa_em=$3
WHERE id=$1 AND estabelecimento_id=$2 AND status='PENDENTE'`,
		offerID, tenantID, s.now().Add(earlySlotSendRetryDelay))
	return err
}

// ProcessSendRetries executa a segunda tentativa das ofertas cujo primeiro
// envio falhou. Só entram ofertas ainda PENDENTE e dentro do prazo.
func (s *EarlySlotService) ProcessSendRetries(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	retried := 0
	for retried < limit {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return retried, err
		}
		var pending struct {
			OfferID       string    `db:"id"`
			TenantID      string    `db:"estabelecimento_id"`
			AppointmentID string    `db:"agendamento_candidato_id"`
			Attempts      int       `db:"tentativas_envio"`
			ClientName    string    `db:"cliente_nome"`
			Phone         string    `db:"cliente_telefone"`
			SalonName     string    `db:"salon_name"`
			Professional  string    `db:"professional_name"`
			CurrentStart  time.Time `db:"data_hora_inicio"`
			OfferedStart  time.Time `db:"slot_inicio"`
		}
		err = tx.GetContext(ctx, &pending, earlySlotRetryLookupQuery, s.now(), earlySlotMaxSendAttempts)
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			break
		}
		if err != nil {
			_ = tx.Rollback()
			return retried, err
		}

		// O token é rotacionado: a tentativa anterior não chegou a ninguém, e
		// só o hash é persistido, então o link antigo não é recuperável.
		token, tokenHash, err := newEarlySlotToken()
		if err != nil {
			_ = tx.Rollback()
			return retried, err
		}
		if _, err = tx.ExecContext(ctx, `
UPDATE ofertas_antecipacao
SET token_hash=$3, tentativas_envio=tentativas_envio+1, proxima_tentativa_em=NULL
WHERE id=$1 AND estabelecimento_id=$2 AND status='PENDENTE'`,
			pending.OfferID, pending.TenantID, tokenHash[:]); err != nil {
			_ = tx.Rollback()
			return retried, err
		}
		if err := tx.Commit(); err != nil {
			return retried, err
		}

		s.deliverOrAdvance(ctx, &earlySlotDelivery{
			OfferID: pending.OfferID, TenantID: pending.TenantID,
			AppointmentID: pending.AppointmentID, Phone: pending.Phone,
			ClientName: pending.ClientName, SalonName: pending.SalonName,
			Professional: pending.Professional, CurrentStart: pending.CurrentStart,
			OfferedStart: pending.OfferedStart, Token: token,
			Attempt: pending.Attempts + 1,
		})
		retried++
	}
	return retried, nil
}

func (s *EarlySlotService) failOfferAndAdvance(ctx context.Context, tenantID, offerID string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	var roundID string
	err = tx.GetContext(ctx, &roundID, `
UPDATE ofertas_antecipacao
SET status = 'FALHA_ENVIO', respondida_em = NOW()
WHERE id = $1 AND estabelecimento_id = $2 AND status = 'PENDENTE'
RETURNING rodada_id`, offerID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	next, err := s.advanceRoundTx(ctx, tx, tenantID, roundID)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.deliverOrAdvance(ctx, next)
	return nil
}

func (s *EarlySlotService) GetOffer(ctx context.Context, token string) (*EarlySlotOfferView, error) {
	hash := hashEarlySlotToken(token)
	var view EarlySlotOfferView
	err := s.db.GetContext(ctx, &view, `
SELECT o.status, e.nome_comercial AS salon_name, p.nome AS professional_name,
       sv.nome AS service_name, a.data_hora_inicio AS current_start,
       a.data_hora_fim AS current_end, r.slot_inicio AS offered_start,
       r.slot_inicio + (a.data_hora_fim - a.data_hora_inicio) AS offered_end,
       o.expira_em AS expires_at
FROM ofertas_antecipacao o
JOIN rodadas_antecipacao r ON r.id = o.rodada_id AND r.estabelecimento_id = o.estabelecimento_id
JOIN agendamentos a ON a.id = o.agendamento_candidato_id AND a.estabelecimento_id = o.estabelecimento_id
JOIN estabelecimentos e ON e.id = o.estabelecimento_id
JOIN profissionais p ON p.id = a.profissional_id AND p.estabelecimento_id = o.estabelecimento_id
JOIN servicos sv ON sv.id = a.servico_id AND sv.estabelecimento_id = o.estabelecimento_id
WHERE o.token_hash = $1`, hash[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEarlySlotOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	if view.Status == "PENDENTE" && !s.now().Before(view.ExpiresAt) {
		view.Status = "EXPIRADA"
	}
	view.ActionsAllowed = []string{}
	if view.Status == "PENDENTE" {
		view.ActionsAllowed = []string{"accept", "decline"}
		view.SecondsRemaining = int64(view.ExpiresAt.Sub(s.now()).Seconds())
		if view.SecondsRemaining < 0 {
			view.SecondsRemaining = 0
		}
	}
	return &view, nil
}

func (s *EarlySlotService) Accept(ctx context.Context, token, origin string) (*EarlySlotAcceptResult, error) {
	hash := hashEarlySlotToken(token)
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	var item earlySlotAcceptItem
	err = tx.GetContext(ctx, &item, earlySlotAcceptLookupSQL, hash[:])
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEarlySlotOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	if item.OfferStatus == "ACEITA" {
		return &EarlySlotAcceptResult{Status: "ACEITA", AppointmentID: item.AppointmentID, NewStart: item.SlotStart, NewEnd: item.SlotStart.Add(item.CurrentEnd.Sub(item.CurrentStart))}, nil
	}
	if item.OfferStatus == "EXPIRADA" || !s.now().Before(item.ExpiresAt) {
		return nil, ErrEarlySlotOfferExpired
	}
	if item.OfferStatus != "PENDENTE" || item.RoundStatus != "ATIVA" {
		return nil, ErrEarlySlotOfferUnavailable
	}
	if !item.Eligible {
		return nil, ErrEarlySlotAppointmentIneligible
	}
	return s.completeAcceptTx(ctx, tx, item, origin)
}

type earlySlotAcceptItem struct {
	OfferID        string    `db:"offer_id"`
	OfferStatus    string    `db:"offer_status"`
	ExpiresAt      time.Time `db:"expira_em"`
	RoundID        string    `db:"rodada_id"`
	RoundStatus    string    `db:"round_status"`
	TenantID       string    `db:"estabelecimento_id"`
	AppointmentID  string    `db:"agendamento_id"`
	ProfessionalID string    `db:"profissional_id"`
	SlotStart      time.Time `db:"slot_inicio"`
	SlotEnd        time.Time `db:"slot_fim"`
	CurrentStart   time.Time `db:"current_start"`
	CurrentEnd     time.Time `db:"current_end"`
	Eligible       bool      `db:"eligible"`
	ClientName     string    `db:"cliente_nome"`
	Phone          string    `db:"cliente_telefone"`
	Professional   string    `db:"profissional_nome"`
	Service        string    `db:"servico_nome"`
}

func (s *EarlySlotService) completeAcceptTx(
	ctx context.Context, tx *sqlx.Tx, item earlySlotAcceptItem, origin string,
) (*EarlySlotAcceptResult, error) {
	newEnd := item.SlotStart.Add(item.CurrentEnd.Sub(item.CurrentStart))
	var conflictingIDs []string
	err := tx.SelectContext(ctx, &conflictingIDs, `
SELECT id FROM agendamentos
WHERE estabelecimento_id=$1 AND profissional_id=$2
  AND status IN ('AGENDADO','CONFIRMADO') AND id<>$3
  AND data_hora_inicio<$5 AND data_hora_fim>$4
FOR UPDATE`, item.TenantID, item.ProfessionalID, item.AppointmentID, item.SlotStart, newEnd)
	if err != nil {
		return nil, err
	}
	if len(conflictingIDs) > 0 {
		return nil, ErrEarlySlotOfferUnavailable
	}
	_, err = tx.ExecContext(ctx, `
UPDATE agendamentos SET data_hora_inicio=$3, data_hora_fim=$4
WHERE id=$1 AND estabelecimento_id=$2`, item.AppointmentID, item.TenantID, item.SlotStart, newEnd)
	if err != nil {
		return nil, err
	}
	// Statements separados: com placeholders o lib/pq usa o protocolo estendido,
	// que recusa mais de um comando por Exec.
	if _, err = tx.ExecContext(ctx, `
UPDATE ofertas_antecipacao SET status='ACEITA', respondida_em=NOW(), origem_resposta=$3
WHERE id=$1 AND estabelecimento_id=$2`,
		item.OfferID, item.TenantID, origin); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
UPDATE ofertas_antecipacao SET status='INVALIDADA', respondida_em=NOW()
WHERE rodada_id=$3 AND estabelecimento_id=$2 AND id<>$1 AND status='PENDENTE'`,
		item.OfferID, item.TenantID, item.RoundID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
UPDATE rodadas_antecipacao SET status='PREENCHIDA', finished_at=NOW()
WHERE id=$1 AND estabelecimento_id=$2`,
		item.RoundID, item.TenantID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
INSERT INTO antecipacao_auditoria
 (estabelecimento_id, rodada_id, oferta_id, evento, horario_anterior_inicio,
  horario_anterior_fim, horario_novo_inicio, horario_novo_fim, origem)
VALUES ($1,$2,$3,'ACEITE',$4,$5,$6,$7,$8)`,
		item.TenantID, item.RoundID, item.OfferID,
		item.CurrentStart, item.CurrentEnd, item.SlotStart, newEnd, origin); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.sendAcceptanceConfirmation(ctx, item.TenantID, item.AppointmentID, item.Phone,
		item.ClientName, item.Professional, item.Service, item.SlotStart)
	return &EarlySlotAcceptResult{Status: "ACEITA", AppointmentID: item.AppointmentID, NewStart: item.SlotStart, NewEnd: newEnd}, nil
}

// sendAcceptanceConfirmation é pós-commit: falha do Gateway não reverte o
// reagendamento já aceito. O erro fica observável sem expor telefone ou token.
func (s *EarlySlotService) sendAcceptanceConfirmation(
	ctx context.Context,
	tenantID, appointmentID, phone, clientName, professional, serviceName string,
	newStart time.Time,
) {
	if err := s.send(ctx, WhatsAppNotificationInput{
		TenantID:      tenantID,
		PhoneNumber:   phone,
		TemplateName:  earlySlotConfirmationTemplate,
		AppointmentID: appointmentID,
		Variables: []string{
			clientName,
			professional,
			serviceName,
			newStart.Format("02/01/2006 15:04"),
		},
	}); err != nil {
		log.Printf("antecipacao: confirmação pós-aceite falhou tenant=%s agendamento=%s: %v",
			tenantID, appointmentID, err)
	}
}

func (s *EarlySlotService) Decline(ctx context.Context, token, origin string) (string, error) {
	hash := hashEarlySlotToken(token)
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck
	var item struct {
		ID        string    `db:"id"`
		Status    string    `db:"status"`
		TenantID  string    `db:"estabelecimento_id"`
		RoundID   string    `db:"rodada_id"`
		ExpiresAt time.Time `db:"expira_em"`
	}
	err = tx.GetContext(ctx, &item, `
SELECT id,status,estabelecimento_id,rodada_id,expira_em FROM ofertas_antecipacao
WHERE token_hash=$1 FOR UPDATE`, hash[:])
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrEarlySlotOfferNotFound
	}
	if err != nil {
		return "", err
	}
	if item.Status == "RECUSADA" {
		return "RECUSADA", nil
	}
	// Fora da janela exclusiva a recusa não vale mais: quem avança a fila é o
	// worker de expiração, e registrar RECUSADA aqui consumiria a vez do
	// próximo candidato e reescreveria o histórico.
	if item.Status == "EXPIRADA" ||
		(item.Status == "PENDENTE" && !s.now().Before(item.ExpiresAt)) {
		return "", ErrEarlySlotOfferExpired
	}
	if item.Status != "PENDENTE" {
		return "", ErrEarlySlotOfferUnavailable
	}
	_, err = tx.ExecContext(ctx, `
UPDATE ofertas_antecipacao SET status='RECUSADA',respondida_em=NOW(),origem_resposta=$3
WHERE id=$1 AND estabelecimento_id=$2`, item.ID, item.TenantID, origin)
	if err != nil {
		return "", err
	}
	next, err := s.advanceRoundTx(ctx, tx, item.TenantID, item.RoundID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	s.deliverOrAdvance(ctx, next)
	return "RECUSADA", nil
}

// RespondWhatsApp valida tenant e telefone antes de reutilizar exatamente a
// mesma transação de aceite/recusa exposta pela API pública.
func (s *EarlySlotService) RespondWhatsApp(ctx context.Context, token, tenantID, phone string, accept bool) (string, error) {
	hash := hashEarlySlotToken(token)
	var scoped struct {
		TenantID string `db:"estabelecimento_id"`
		Phone    string `db:"telefone"`
	}
	err := s.db.GetContext(ctx, &scoped, `
SELECT o.estabelecimento_id,c.telefone
FROM ofertas_antecipacao o
JOIN agendamentos a ON a.id=o.agendamento_candidato_id AND a.estabelecimento_id=o.estabelecimento_id
JOIN clientes c ON c.id=a.cliente_id AND c.estabelecimento_id=a.estabelecimento_id
WHERE o.token_hash=$1 AND o.estabelecimento_id=$2`, hash[:], tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrEarlySlotOfferNotFound
	}
	if err != nil {
		return "", err
	}
	if !phonesMatch(scoped.Phone, phone) {
		return "", ErrEarlySlotOfferNotFound
	}
	if accept {
		result, err := s.Accept(ctx, token, "WHATSAPP")
		if err != nil {
			return "", err
		}
		return result.Status, nil
	}
	return s.Decline(ctx, token, "WHATSAPP")
}

func (s *EarlySlotService) ProcessExpired(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}
	processed := 0
	for processed < limit {
		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return processed, err
		}
		var item struct {
			ID       string `db:"id"`
			TenantID string `db:"estabelecimento_id"`
			RoundID  string `db:"rodada_id"`
		}
		err = tx.GetContext(ctx, &item, `
SELECT id,estabelecimento_id,rodada_id FROM ofertas_antecipacao
WHERE status='PENDENTE' AND expira_em <= $1
ORDER BY expira_em LIMIT 1 FOR UPDATE SKIP LOCKED`, s.now())
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			break
		}
		if err != nil {
			_ = tx.Rollback()
			return processed, err
		}
		_, err = tx.ExecContext(ctx, `
UPDATE ofertas_antecipacao SET status='EXPIRADA',respondida_em=NOW()
WHERE id=$1 AND estabelecimento_id=$2`, item.ID, item.TenantID)
		if err != nil {
			_ = tx.Rollback()
			return processed, err
		}
		next, err := s.advanceRoundTx(ctx, tx, item.TenantID, item.RoundID)
		if err != nil {
			_ = tx.Rollback()
			return processed, err
		}
		if err := tx.Commit(); err != nil {
			return processed, err
		}
		s.deliverOrAdvance(ctx, next)
		processed++
	}
	return processed, nil
}

// closeRoundChannelDownTx encerra a rodada e invalida a oferta pendente numa
// única transação, sem chamar o próximo candidato: o slot volta a ficar livre.
// Rodada já PREENCHIDA é terminal e não é alcançada por este caminho, então um
// aceite concluído antes da queda nunca é revertido.
func (s *EarlySlotService) closeRoundChannelDownTx(ctx context.Context, tx *sqlx.Tx, tenantID, roundID string) error {
	if _, err := tx.ExecContext(ctx, `
UPDATE ofertas_antecipacao SET status='INVALIDADA', respondida_em=NOW()
WHERE rodada_id=$1 AND estabelecimento_id=$2 AND status='PENDENTE'`,
		roundID, tenantID); err != nil {
		return fmt.Errorf("invalidar oferta pendente da rodada %s: %w", roundID, err)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE rodadas_antecipacao
SET status='CANCELADA', motivo_encerramento=$3, finished_at=NOW(),
    candidato_atual_agendamento_id=NULL
WHERE id=$1 AND estabelecimento_id=$2 AND status='ATIVA'`,
		roundID, tenantID, earlySlotMotivoCanalIndisponivel); err != nil {
		return fmt.Errorf("encerrar rodada %s por canal indisponível: %w", roundID, err)
	}
	log.Printf("antecipacao: rodada encerrada tenant=%s rodada=%s motivo=%s",
		tenantID, roundID, earlySlotMotivoCanalIndisponivel)
	return nil
}

// ProcessChannelDrops encerra rodadas ATIVA cujo salão perdeu o canal durante a
// janela exclusiva. Erro na checagem não encerra nada (fail-closed): a rodada
// fica para o próximo ciclo.
func (s *EarlySlotService) ProcessChannelDrops(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 50
	}

	var pending []struct {
		RoundID  string `db:"id"`
		TenantID string `db:"estabelecimento_id"`
	}
	if err := s.db.SelectContext(ctx, &pending, `
SELECT r.id, r.estabelecimento_id
FROM rodadas_antecipacao r
WHERE r.status='ATIVA'
  AND EXISTS (
      SELECT 1 FROM ofertas_antecipacao o
      WHERE o.rodada_id=r.id AND o.estabelecimento_id=r.estabelecimento_id
        AND o.status='PENDENTE')
ORDER BY r.created_at
LIMIT $1`, limit); err != nil {
		return 0, fmt.Errorf("listar rodadas ativas: %w", err)
	}

	closed := 0
	for _, round := range pending {
		ready, err := s.channelReady(ctx, s.db, round.TenantID)
		if err != nil {
			log.Printf("antecipacao: checagem de canal falhou tenant=%s rodada=%s — rodada mantida: %v",
				round.TenantID, round.RoundID, err)
			continue
		}
		if ready {
			continue
		}

		tx, err := s.db.BeginTxx(ctx, nil)
		if err != nil {
			return closed, err
		}
		var locked string
		err = tx.GetContext(ctx, &locked, `
SELECT id FROM rodadas_antecipacao
WHERE id=$1 AND estabelecimento_id=$2 AND status='ATIVA'
FOR UPDATE SKIP LOCKED`, round.RoundID, round.TenantID)
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			continue
		}
		if err != nil {
			_ = tx.Rollback()
			return closed, err
		}
		if err := s.closeRoundChannelDownTx(ctx, tx, round.TenantID, round.RoundID); err != nil {
			_ = tx.Rollback()
			return closed, err
		}
		if err := tx.Commit(); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

func (s *EarlySlotService) RunExpirationWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.ProcessExpired(ctx, 50)
			_, _ = s.ProcessSendRetries(ctx, 50)
			_, _ = s.ProcessChannelDrops(ctx, 50)
		}
	}
}

func (s *EarlySlotService) GetRound(ctx context.Context, tenantID, roundID string) (*EarlySlotRound, error) {
	var round EarlySlotRound
	err := s.db.GetContext(ctx, &round, `
SELECT id,status,motivo_encerramento,profissional_id,slot_inicio,slot_fim
FROM rodadas_antecipacao WHERE id=$1 AND estabelecimento_id=$2`, roundID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEarlySlotOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	round.History = []EarlySlotHistory{}
	if err := s.db.SelectContext(ctx, &round.History, `
SELECT posicao,status,enviada_em,expira_em FROM ofertas_antecipacao
WHERE rodada_id=$1 AND estabelecimento_id=$2 ORDER BY posicao`, roundID, tenantID); err != nil {
		return nil, err
	}
	var current struct {
		AppointmentID string    `db:"agendamento_id"`
		ClientName    string    `db:"cliente_nome"`
		Position      int       `db:"posicao"`
		Status        string    `db:"status"`
		ExpiresAt     time.Time `db:"expira_em"`
	}
	err = s.db.GetContext(ctx, &current, `
SELECT a.id AS agendamento_id,c.nome AS cliente_nome,o.posicao,o.status,o.expira_em
FROM ofertas_antecipacao o
JOIN agendamentos a ON a.id=o.agendamento_candidato_id AND a.estabelecimento_id=o.estabelecimento_id
JOIN clientes c ON c.id=a.cliente_id AND c.estabelecimento_id=a.estabelecimento_id
WHERE o.rodada_id=$1 AND o.estabelecimento_id=$2 AND o.status='PENDENTE'`, roundID, tenantID)
	if err == nil {
		round.Current = &EarlySlotCurrent{AppointmentID: current.AppointmentID, ClientName: current.ClientName,
			Position: current.Position, OfferStatus: current.Status, ExpiresAt: current.ExpiresAt}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return &round, nil
}

func (s *EarlySlotService) ListRounds(ctx context.Context, tenantID, professionalID, status string) ([]EarlySlotRound, error) {
	query := `
SELECT id,status,motivo_encerramento,profissional_id,slot_inicio,slot_fim
FROM rodadas_antecipacao
WHERE estabelecimento_id=$1
  AND ($2='' OR profissional_id::text=$2)
  AND ($3='' OR status=$3)
ORDER BY created_at DESC`
	var rounds []EarlySlotRound
	if err := s.db.SelectContext(ctx, &rounds, query, tenantID, professionalID, status); err != nil {
		return nil, err
	}
	if rounds == nil {
		rounds = []EarlySlotRound{}
	}
	return rounds, nil
}

// EarlySlotPreferenceResult carrega o eco canônico do PATCH de preferência,
// usado tanto pela rota staff quanto pela rota pública de gestão.
type EarlySlotPreferenceResult struct {
	TenantID             string     `json:"-"`
	AceitaAdiantar       bool       `json:"aceita_adiantar"`
	AceitaAdiantarEm     *time.Time `json:"aceita_adiantar_em"`
	NotificationsEnabled bool       `json:"early_slot_notifications_available"`
}

// SetPreferenceByManagementToken é o contrato público de opt-in por
// `gestao_token` (DEV-84). O opt-in é persistido mesmo com o canal fora do ar:
// o sinal devolvido apenas informa que ainda não haverá oferta.
func (s *EarlySlotService) SetPreferenceByManagementToken(
	ctx context.Context, token string, enabled bool, now time.Time,
) (*EarlySlotPreferenceResult, error) {
	token = strings.TrimSpace(token)
	if !managementTokenPattern.MatchString(token) {
		return nil, ErrEarlySlotOfferNotFound
	}

	var current struct {
		TenantID  string    `db:"estabelecimento_id"`
		Status    string    `db:"status"`
		StartsAt  time.Time `db:"data_hora_inicio"`
		ExpiresAt time.Time `db:"gestao_token_expires_at"`
	}
	err := s.db.GetContext(ctx, &current, `
SELECT estabelecimento_id, status, data_hora_inicio, gestao_token_expires_at
FROM agendamentos WHERE gestao_token::text = $1`, token)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEarlySlotOfferNotFound
	}
	if err != nil {
		return nil, err
	}
	// Token vencido é indistinguível de token inexistente para quem chama: não
	// vaza a existência do agendamento.
	if !now.Before(current.ExpiresAt) {
		return nil, ErrEarlySlotOfferNotFound
	}
	if current.Status != "AGENDADO" && current.Status != "CONFIRMADO" {
		return nil, ErrEarlySlotAppointmentIneligible
	}
	if !current.StartsAt.After(now) {
		return nil, ErrEarlySlotAppointmentIneligible
	}

	var at sql.NullTime
	err = s.db.GetContext(ctx, &at, `
UPDATE agendamentos SET aceita_adiantar = $2
WHERE gestao_token::text = $1
  AND status IN ('AGENDADO','CONFIRMADO')
  AND data_hora_inicio > $3
RETURNING aceita_adiantar_em`, token, enabled, now)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEarlySlotAppointmentIneligible
	}
	if err != nil {
		return nil, err
	}

	out := &EarlySlotPreferenceResult{
		TenantID:             current.TenantID,
		AceitaAdiantar:       enabled,
		NotificationsEnabled: s.NotificationsAvailable(ctx, current.TenantID),
	}
	if enabled && at.Valid {
		moment := at.Time
		out.AceitaAdiantarEm = &moment
	}
	return out, nil
}

func (s *EarlySlotService) SetPreference(ctx context.Context, tenantID, appointmentID string, enabled bool) (time.Time, error) {
	var at sql.NullTime
	err := s.db.GetContext(ctx, &at, `
UPDATE agendamentos SET aceita_adiantar=$3
WHERE id=$1 AND estabelecimento_id=$2
  AND status IN ('AGENDADO','CONFIRMADO') AND data_hora_inicio > NOW()
RETURNING aceita_adiantar_em`, appointmentID, tenantID, enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, ErrEarlySlotAppointmentIneligible
	}
	if err != nil {
		return time.Time{}, err
	}
	return at.Time, nil
}
