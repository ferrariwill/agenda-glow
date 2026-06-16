package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrWebhookPayloadInvalido   = errors.New("payload do webhook inválido")
	ErrAcaoWhatsAppInvalida     = errors.New("action inválida: use CONFIRM ou CANCEL")
	ErrAgendamentoEscopoInvalido = errors.New("agendamento não pertence ao salão ou telefone informado")
	ErrAgendamentoStatusFinal   = errors.New("agendamento em status final não pode ser alterado")
)

const (
	WhatsAppActionConfirm = "CONFIRM"
	WhatsAppActionCancel  = "CANCEL"
)

// WhatsAppCallbackPayload representa o JSON repassado pelo WhatsApp Gateway.
type WhatsAppCallbackPayload struct {
	SystemID         string `json:"system_id"`
	ExternalClientID string `json:"external_client_id"`
	PhoneNumber      string `json:"phone_number"`
	Text             string `json:"text"`
	EventType        string `json:"event_type"`
	Action           string `json:"action"`
	AppointmentID    string `json:"appointment_id"`
}

func (p *WhatsAppCallbackPayload) normalize() {
	p.SystemID = strings.TrimSpace(p.SystemID)
	p.ExternalClientID = strings.TrimSpace(p.ExternalClientID)
	p.PhoneNumber = strings.TrimSpace(p.PhoneNumber)
	p.Text = strings.TrimSpace(p.Text)
	p.EventType = strings.TrimSpace(p.EventType)
	p.Action = strings.ToUpper(strings.TrimSpace(p.Action))
	p.AppointmentID = strings.TrimSpace(p.AppointmentID)
}

func (p *WhatsAppCallbackPayload) validate() error {
	p.normalize()

	if p.AppointmentID == "" {
		return fmt.Errorf("%w: appointment_id obrigatório", ErrWebhookPayloadInvalido)
	}
	if p.ExternalClientID == "" {
		return fmt.Errorf("%w: external_client_id obrigatório", ErrWebhookPayloadInvalido)
	}
	if p.PhoneNumber == "" {
		return fmt.Errorf("%w: phone_number obrigatório", ErrWebhookPayloadInvalido)
	}
	if p.EventType != "" && p.EventType != "button_reply" {
		return fmt.Errorf("%w: event_type inválido", ErrWebhookPayloadInvalido)
	}
	if p.Text != "" && p.Text != "APPT_CONFIRM" {
		return fmt.Errorf("%w: text inválido", ErrWebhookPayloadInvalido)
	}

	switch p.Action {
	case WhatsAppActionConfirm, WhatsAppActionCancel:
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

	ag, err := s.buscarAgendamento(ctx, payload.AppointmentID)
	if err != nil {
		return "", err
	}

	if ag.EstabelecimentoID != payload.ExternalClientID {
		return "", ErrAgendamentoEscopoInvalido
	}

	if normalizePhoneDigits(ag.ClienteTelefone) != normalizePhoneDigits(payload.PhoneNumber) {
		return "", ErrAgendamentoEscopoInvalido
	}

	targetStatus := "CONFIRMADO"
	if payload.Action == WhatsAppActionCancel {
		targetStatus = "CANCELADO"
	}

	if ag.Status == targetStatus {
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
	}

	const update = `
UPDATE agendamentos
SET status = $2
WHERE id = $1
  AND estabelecimento_id = $3
`
	res, err := s.db.ExecContext(ctx, update, payload.AppointmentID, targetStatus, payload.ExternalClientID)
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

	return targetStatus, nil
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
