package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrWebhookPayloadInvalido                     = errors.New("payload do webhook inválido")
	ErrAcaoWhatsAppInvalida                       = errors.New("action inválida: use CONFIRM ou CANCEL")
	ErrAgendamentoEscopoInvalido                  = errors.New("agendamento não pertence ao salão ou telefone informado")
	ErrAgendamentoStatusFinal                     = errors.New("agendamento em status final não pode ser alterado")
	ErrAgendamentoAguardandoAprovacaoProfissional = errors.New("agendamento aguarda aprovação da profissional")
)

const (
	WhatsAppActionConfirm = "CONFIRM"
	WhatsAppActionCancel  = "CANCEL"
	WhatsAppSistemaBeleza = "beleza"
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
	Reason        string `json:"reason"`
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
	p.Reason = strings.TrimSpace(p.Reason)
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
	if p.TenantID == "" {
		return fmt.Errorf("%w: tenant_id obrigatório", ErrWebhookPayloadInvalido)
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
	default:
		return ErrAcaoWhatsAppInvalida
	}

	return nil
}

// ProcessWhatsAppCallback separa confirmação do cliente do status operacional.
func (s *AgendaService) ProcessWhatsAppCallback(ctx context.Context, payload WhatsAppCallbackPayload) (string, error) {
	if err := payload.validate(); err != nil {
		return "", err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback() //nolint:errcheck

	ag, err := s.resolverAgendamentoWhatsApp(ctx, tx, payload)
	if err != nil {
		return "", err
	}

	if payload.Action == WhatsAppActionConfirm {
		if ag.Status == "CONCLUIDO" {
			return "", ErrAgendamentoStatusFinal
		}
		if ag.Status == "CANCELADO" {
			return "", ErrAgendamentoCancelado
		}
		if ag.CustomerConfirmation != "CONFIRMADO_CLIENTE" {
			if _, err := tx.ExecContext(ctx, `
UPDATE agendamentos
SET confirmacao_cliente = 'CONFIRMADO_CLIENTE'
WHERE id = $1 AND estabelecimento_id = $2
`, ag.ID, ag.EstablishmentID); err != nil {
				return "", fmt.Errorf("confirmar presença do cliente: %w", err)
			}
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return ag.Status, nil
	}

	management := &managementAppointment{
		ID: ag.ID, EstablishmentID: ag.EstablishmentID, StartsAt: ag.StartsAt,
		Status: ag.Status, CustomerConfirmation: ag.CustomerConfirmation,
		MinimumCancellationNoticeHours: ag.MinimumCancellationNoticeHours,
		ReasonRequired:                 ag.ReasonRequired, ContactPhone: ag.ContactPhone,
	}
	if _, err := cancelManagementAppointment(ctx, tx, management, payload.Reason, CancellationOriginWhatsApp, time.Now()); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return "CANCELADO", nil
}

type callbackAppointment struct {
	ID                             string    `db:"id"`
	EstablishmentID                string    `db:"estabelecimento_id"`
	StartsAt                       time.Time `db:"data_hora_inicio"`
	Status                         string    `db:"status"`
	CustomerConfirmation           string    `db:"confirmacao_cliente"`
	MinimumCancellationNoticeHours int       `db:"janela_minima_cancelamento_horas"`
	ReasonRequired                 bool      `db:"motivo_cancelamento_obrigatorio"`
	ContactPhone                   string    `db:"telefone_contato"`
}

func (s *AgendaService) resolverAgendamentoWhatsApp(
	ctx context.Context,
	tx *sqlx.Tx,
	payload WhatsAppCallbackPayload,
) (*callbackAppointment, error) {
	phoneDigits := normalizePhoneDigits(payload.PhoneNumber)
	if phoneDigits == "" {
		return nil, fmt.Errorf("%w: phone_number inválido", ErrWebhookPayloadInvalido)
	}

	q := `
SELECT
    a.id, a.estabelecimento_id, a.data_hora_inicio, a.status,
    a.confirmacao_cliente, cfg.janela_minima_cancelamento_horas,
    cfg.motivo_cancelamento_obrigatorio,
    COALESCE(e.whatsapp_phone_number, '') AS telefone_contato
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
INNER JOIN configuracoes_notificacoes_agenda cfg ON cfg.estabelecimento_id = a.estabelecimento_id
INNER JOIN estabelecimentos e ON e.id = a.estabelecimento_id
WHERE a.estabelecimento_id = $1
  AND ($3 = '' OR a.id::text = $3)
  AND (
    regexp_replace(c.telefone, '[^0-9]', '', 'g') = $2
    OR regexp_replace(c.telefone, '[^0-9]', '', 'g') = '55' || $2
    OR $2 LIKE '%' || regexp_replace(c.telefone, '[^0-9]', '', 'g')
    OR regexp_replace(c.telefone, '[^0-9]', '', 'g') LIKE '%' || RIGHT($2, 11)
  )
ORDER BY a.data_hora_inicio DESC
LIMIT 1
FOR UPDATE OF a
`
	var ag callbackAppointment
	if err := tx.GetContext(ctx, &ag, q, payload.TenantID, phoneDigits, payload.AppointmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if payload.AppointmentID != "" {
				var exists bool
				if checkErr := tx.GetContext(ctx, &exists, `
SELECT EXISTS(SELECT 1 FROM agendamentos WHERE id::text = $1)
`, payload.AppointmentID); checkErr == nil && exists {
					return nil, ErrAgendamentoEscopoInvalido
				}
			}
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
