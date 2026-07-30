package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
)

var (
	ErrWebhookPayloadInvalido                     = errors.New("payload do webhook inválido")
	ErrAcaoWhatsAppInvalida                       = errors.New("action inválida: use CONFIRM ou CANCEL")
	ErrAgendamentoEscopoInvalido                  = errors.New("agendamento não pertence ao salão ou telefone informado")
	ErrAgendamentoStatusFinal                     = errors.New("agendamento em status final não pode ser alterado")
	ErrAgendamentoAguardandoAprovacaoProfissional = errors.New("agendamento aguarda aprovação da profissional")
)

const (
	WhatsAppActionConfirm          = "CONFIRM"
	WhatsAppActionCancel           = "CANCEL"
	WhatsAppActionEarlySlotAccept  = "EARLY_SLOT_ACCEPT"
	WhatsAppActionEarlySlotDecline = "EARLY_SLOT_DECLINE"
	WhatsAppSistemaBeleza          = "beleza"
)

// WhatsAppCallbackPayload representa o JSON repassado pelo WhatsApp Gateway (Gateway → Beleza).
// Aceita o formato novo (sistema_origem + tenant_id) e o legado (appointment_id + external_client_id).
type WhatsAppCallbackPayload struct {
	SistemaOrigem string `json:"sistema_origem"`
	TenantID      string `json:"tenant_id"`
	SalonID       string `json:"salon_id"`
	PhoneNumber   string `json:"phone_number"`
	Text          string `json:"text"`
	EventType     string `json:"event_type"`
	Action        string `json:"action"`
	AppointmentID string `json:"appointment_id"`
	OfferToken    string `json:"offer_token"`
	// Legado (pré-contrato Gateway atual)
	SystemID         string `json:"system_id"`
	ExternalClientID string `json:"external_client_id"`
}

func (p *WhatsAppCallbackPayload) normalize() {
	p.SistemaOrigem = strings.ToLower(strings.TrimSpace(p.SistemaOrigem))
	p.TenantID = strings.TrimSpace(p.TenantID)
	p.SalonID = strings.TrimSpace(p.SalonID)
	p.PhoneNumber = strings.TrimSpace(p.PhoneNumber)
	p.Text = strings.TrimSpace(p.Text)
	p.EventType = strings.TrimSpace(p.EventType)
	p.Action = strings.ToUpper(strings.TrimSpace(p.Action))
	p.AppointmentID = strings.TrimSpace(p.AppointmentID)
	p.OfferToken = strings.TrimSpace(p.OfferToken)
	p.SystemID = strings.TrimSpace(p.SystemID)
	p.ExternalClientID = strings.TrimSpace(p.ExternalClientID)

	// Aliases → campos canônicos
	if p.TenantID == "" {
		p.TenantID = p.SalonID
	}
	if p.TenantID == "" {
		p.TenantID = p.ExternalClientID
	}
	if p.SistemaOrigem == "" && (p.SystemID == WhatsAppSistemaBeleza || p.SystemID == "") {
		if p.SystemID == WhatsAppSistemaBeleza {
			p.SistemaOrigem = WhatsAppSistemaBeleza
		}
	}
}

func (p *WhatsAppCallbackPayload) validate() error {
	p.normalize()

	if p.SistemaOrigem != "" && p.SistemaOrigem != WhatsAppSistemaBeleza {
		return fmt.Errorf("%w: sistema_origem deve ser beleza", ErrWebhookPayloadInvalido)
	}
	if p.TenantID == "" && p.AppointmentID == "" {
		return fmt.Errorf("%w: tenant_id ou appointment_id obrigatório", ErrWebhookPayloadInvalido)
	}
	if p.PhoneNumber == "" {
		return fmt.Errorf("%w: phone_number obrigatório", ErrWebhookPayloadInvalido)
	}
	if p.EventType != "" && p.EventType != "button_reply" {
		return fmt.Errorf("%w: event_type inválido", ErrWebhookPayloadInvalido)
	}
	// Gateway envia APPT_CONFIRM; aceita também vazio / CANCEL equivalentes
	if p.Text != "" && p.Text != "APPT_CONFIRM" && p.Text != "APPT_CANCEL" {
		// não bloqueia textos desconhecidos se action for válida
	}

	switch p.Action {
	case WhatsAppActionConfirm, WhatsAppActionCancel:
	case WhatsAppActionEarlySlotAccept, WhatsAppActionEarlySlotDecline:
		if p.TenantID == "" || p.OfferToken == "" {
			return fmt.Errorf("%w: tenant_id e offer_token obrigatórios", ErrWebhookPayloadInvalido)
		}
	default:
		return ErrAcaoWhatsAppInvalida
	}

	return nil
}

// ProcessWhatsAppCallback atualiza o status do agendamento conforme a ação da cliente.
func (s *AgendaService) ProcessWhatsAppCallback(ctx context.Context, payload WhatsAppCallbackPayload) (string, error) {
	if err := payload.validate(); err != nil {
		return "", err
	}
	if payload.Action == WhatsAppActionEarlySlotAccept || payload.Action == WhatsAppActionEarlySlotDecline {
		if s.earlySlot == nil {
			return "", ErrEarlySlotOfferUnavailable
		}
		return s.earlySlot.RespondWhatsApp(
			ctx, payload.OfferToken, payload.TenantID, payload.PhoneNumber,
			payload.Action == WhatsAppActionEarlySlotAccept,
		)
	}

	ag, err := s.resolverAgendamentoWhatsApp(ctx, payload)
	if err != nil {
		return "", err
	}

	if payload.TenantID != "" && ag.EstabelecimentoID != payload.TenantID {
		return "", ErrAgendamentoEscopoInvalido
	}

	if normalizePhoneDigits(ag.ClienteTelefone) != "" &&
		!phonesMatch(ag.ClienteTelefone, payload.PhoneNumber) {
		return "", ErrAgendamentoEscopoInvalido
	}

	targetStatus := "CONFIRMADO"
	if payload.Action == WhatsAppActionCancel {
		targetStatus = "CANCELADO"
	}

	if ag.Status == targetStatus {
		// Callback repetido é idempotente e não reabre fila antiga após uma
		// eventual reconexão do canal.
		return targetStatus, nil
	}

	switch ag.Status {
	case "CONCLUIDO":
		return "", ErrAgendamentoStatusFinal
	case "CANCELADO":
		if payload.Action == WhatsAppActionConfirm {
			return "", ErrAgendamentoCancelado
		}
		return targetStatus, nil
	case "EM_APROVACAO":
		// Cliente confirma presença; não autoriza o encaixe — só a profissional pode.
		if payload.Action == WhatsAppActionConfirm {
			return "", ErrAgendamentoAguardandoAprovacaoProfissional
		}
	}

	const update = `
UPDATE agendamentos
SET status = $2
WHERE id = $1
  AND estabelecimento_id = $3
`
	res, err := s.db.ExecContext(ctx, update, ag.ID, targetStatus, ag.EstabelecimentoID)
	if err != nil {
		return "", fmt.Errorf("atualizar status do agendamento: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if rows == 0 {
		return "", ErrAgendamentoNaoEncontrado
	}
	if targetStatus == "CANCELADO" && s.earlySlot != nil {
		if err := s.earlySlot.OpenRoundForCancelledAppointment(ctx, ag.EstabelecimentoID, ag.ID); err != nil {
			// O UPDATE do cancelamento já foi concluído. A fila é um hook
			// pós-commit e sua falha não altera o contrato de sucesso.
			log.Printf("antecipacao: hook pós-cancelamento whatsapp falhou tenant=%s agendamento_cancelado=%s: %v",
				ag.EstabelecimentoID, ag.ID, err)
		}
	}

	return targetStatus, nil
}

func (s *AgendaService) resolverAgendamentoWhatsApp(
	ctx context.Context,
	payload WhatsAppCallbackPayload,
) (*agendamentoPendente, error) {
	if payload.AppointmentID != "" {
		return s.buscarAgendamento(ctx, payload.AppointmentID)
	}

	phoneDigits := normalizePhoneDigits(payload.PhoneNumber)
	if phoneDigits == "" || payload.TenantID == "" {
		return nil, fmt.Errorf("%w: sem appointment_id, informe tenant_id e phone_number", ErrWebhookPayloadInvalido)
	}

	const q = `
SELECT
    a.id,
    a.estabelecimento_id,
    a.profissional_id,
    a.servico_id,
    a.data_hora_inicio,
    a.data_hora_fim,
    a.status,
    c.nome AS cliente_nome,
    c.telefone AS cliente_telefone,
    s.nome AS servico_nome,
    p.nome AS profissional_nome,
    p.email AS profissional_email
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
INNER JOIN servicos s ON s.id = a.servico_id
INNER JOIN profissionais p ON p.id = a.profissional_id
WHERE a.estabelecimento_id = $1
  AND a.status IN ('AGENDADO', 'CONFIRMADO', 'EM_APROVACAO')
  AND (
    regexp_replace(c.telefone, '[^0-9]', '', 'g') = $2
    OR regexp_replace(c.telefone, '[^0-9]', '', 'g') = '55' || $2
    OR $2 LIKE '%' || regexp_replace(c.telefone, '[^0-9]', '', 'g')
    OR regexp_replace(c.telefone, '[^0-9]', '', 'g') LIKE '%' || RIGHT($2, 11)
  )
ORDER BY a.data_hora_inicio DESC
LIMIT 1
`
	var ag agendamentoPendente
	if err := s.db.GetContext(ctx, &ag, q, payload.TenantID, phoneDigits); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgendamentoNaoEncontrado
		}
		return nil, fmt.Errorf("resolver agendamento por telefone: %w", err)
	}
	return &ag, nil
}

func phonesMatch(a, b string) bool {
	da, db := normalizePhoneDigits(a), normalizePhoneDigits(b)
	if da == "" || db == "" {
		return false
	}
	if da == db {
		return true
	}
	if strings.HasSuffix(da, db) || strings.HasSuffix(db, da) {
		return true
	}
	// compara últimos 11 dígitos (DDD+número BR)
	if len(da) >= 11 && len(db) >= 11 {
		return da[len(da)-11:] == db[len(db)-11:]
	}
	return false
}

func normalizePhoneDigits(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
