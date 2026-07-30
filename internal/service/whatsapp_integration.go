package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	WhatsAppStatusDesconectado = "DESCONECTADO"
	WhatsAppStatusPendente     = "PENDENTE"
	WhatsAppStatusConectado    = "CONECTADO"

	WhatsAppStatePrefix = "beleza"
)

var (
	ErrWhatsAppStateInvalido     = errors.New("state de integração WhatsApp inválido")
	ErrWhatsAppRecursoDesativado = errors.New("recurso de whatsapp desativado")
)

type WhatsAppFeatureView struct {
	EstabelecimentoID string     `db:"estabelecimento_id" json:"estabelecimento_id"`
	NomeComercial     string     `db:"nome_comercial" json:"nome_comercial"`
	WhatsAppEnabled   bool       `db:"whatsapp_enabled" json:"whatsapp_enabled"`
	WhatsAppStatus    string     `db:"whatsapp_status" json:"whatsapp_status"`
	ConnectedAt       *time.Time `db:"connected_at" json:"connected_at,omitempty"`
}

// WhatsAppIntegrationView é o estado da integração para o painel da dona.
type WhatsAppIntegrationView struct {
	EstabelecimentoID string     `json:"estabelecimento_id"`
	WhatsAppEnabled   bool       `json:"whatsapp_enabled"`
	Status            string     `json:"status"`
	State             string     `json:"state"`
	SignupURL         string     `json:"signup_url"`
	WabaID            *string    `json:"waba_id,omitempty"`
	PhoneNumberID     *string    `json:"phone_number_id,omitempty"`
	ConnectedAt       *time.Time `json:"connected_at,omitempty"`
}

// SetWhatsAppEnabled liga/desliga a permissão sem alterar o estado da conexão.
func (s *EstabelecimentoService) SetWhatsAppEnabled(
	ctx context.Context,
	estabelecimentoID string,
	enabled bool,
) (*WhatsAppFeatureView, error) {
	const query = `
UPDATE estabelecimentos
SET whatsapp_enabled = $2
WHERE id = $1
RETURNING id AS estabelecimento_id,
          nome_comercial,
          whatsapp_enabled,
          COALESCE(whatsapp_status, 'DESCONECTADO') AS whatsapp_status,
          whatsapp_connected_at AS connected_at
`
	var view WhatsAppFeatureView
	if err := s.db.GetContext(ctx, &view, query, estabelecimentoID, enabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("atualizar flag whatsapp: %w", err)
	}
	return &view, nil
}

// WhatsAppEnabledForTenant consulta a permissão no escopo explícito do salão.
func WhatsAppEnabledForTenant(
	ctx context.Context,
	q sqlx.QueryerContext,
	estabelecimentoID string,
) (bool, error) {
	const query = `
SELECT COALESCE(whatsapp_enabled, FALSE)
FROM estabelecimentos
WHERE id = $1
`
	var enabled bool
	if err := sqlx.GetContext(ctx, q, &enabled, query, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("consultar flag whatsapp: %w", err)
	}
	return enabled, nil
}

// WhatsAppChannelReadyForTenant compõe a permissão administrativa com o estado
// da conexão: só há canal de saída quando o recurso está liberado e o Embedded
// Signup concluiu. O erro sobe para o chamador decidir a política fail-closed.
func WhatsAppChannelReadyForTenant(
	ctx context.Context,
	q sqlx.QueryerContext,
	estabelecimentoID string,
) (bool, error) {
	enabled, err := WhatsAppEnabledForTenant(ctx, q, estabelecimentoID)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}

	const query = `
SELECT COALESCE(whatsapp_status, 'DESCONECTADO')
FROM estabelecimentos
WHERE id = $1
`
	var status string
	if err := sqlx.GetContext(ctx, q, &status, query, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("consultar status whatsapp: %w", err)
	}
	return status == WhatsAppStatusConectado, nil
}

// WhatsAppConnectedPayload é o aviso do Gateway após Embedded Signup bem-sucedido.
// Aceita o payload oficial (event/tenant_id/salon_id) e aliases legados.
type WhatsAppConnectedPayload struct {
	Event               string `json:"event"`
	Message             string `json:"message"`
	SistemaOrigem       string `json:"sistema_origem"`
	TenantID            string `json:"tenant_id"`
	SalonID             string `json:"salon_id"`
	WabaID              string `json:"waba_id"`
	PhoneNumberID       string `json:"phone_number_id"`
	WhatsAppPhoneNumber string `json:"whatsapp_phone_number"`
	Status              string `json:"status"`
	// Legado / aliases
	State             string `json:"state"`
	EstabelecimentoID string `json:"estabelecimento_id"`
}

// BuildWhatsAppState monta state=beleza_{idDoSalao}.
func BuildWhatsAppState(estabelecimentoID string) string {
	return WhatsAppStatePrefix + "_" + strings.TrimSpace(estabelecimentoID)
}

// ParseWhatsAppState extrai o ID do salão a partir de state=beleza_{id}.
func ParseWhatsAppState(state string) (string, error) {
	state = strings.TrimSpace(state)
	prefix := WhatsAppStatePrefix + "_"
	if !strings.HasPrefix(state, prefix) {
		return "", ErrWhatsAppStateInvalido
	}
	id := strings.TrimSpace(strings.TrimPrefix(state, prefix))
	if id == "" {
		return "", ErrWhatsAppStateInvalido
	}
	return id, nil
}

// BuildWhatsAppEmbeddedSignupURL concatena a URL base da Meta com &state=beleza_{idDoSalao}.
func BuildWhatsAppEmbeddedSignupURL(estabelecimentoID string) string {
	base := strings.TrimSpace(os.Getenv("META_WHATSAPP_EMBEDDED_SIGNUP_URL"))
	if base == "" {
		base = strings.TrimSpace(os.Getenv("WHATSAPP_EMBEDDED_SIGNUP_URL"))
	}
	id := strings.TrimSpace(estabelecimentoID)
	if base == "" {
		base = "https://business.facebook.com/messaging/whatsapp/onboard/?app_id=1743419366842637&config_id=3724327417718926&extras=%7B%22version%22%3A%22v4%22%2C%22sessionInfoVersion%22%3A%223%22%2C%22featureType%22%3A%22whatsapp_business_app_onboarding%22%7D&redirect_uri=https%3A%2F%2Fpreliminary-entrepreneur-cad-denver.trycloudflare.com%2Fmeta%2Fembedded-signup%2Fcallback"
	}

	// Remove state anterior, se a base já trouxer o parâmetro.
	if i := strings.Index(base, "&state="); i >= 0 {
		base = base[:i]
	} else if i := strings.Index(base, "?state="); i >= 0 {
		base = base[:i]
	}

	sep := "&"
	if !strings.Contains(base, "?") {
		sep = "?"
	}
	return base + sep + "state=beleza_" + id
}

// GetWhatsAppIntegration retorna status + URL de Embedded Signup do salão.
func (s *EstabelecimentoService) GetWhatsAppIntegration(ctx context.Context, estabelecimentoID string) (*WhatsAppIntegrationView, error) {
	const q = `
SELECT id,
       COALESCE(whatsapp_enabled, FALSE) AS whatsapp_enabled,
       COALESCE(whatsapp_status, 'DESCONECTADO') AS whatsapp_status,
       whatsapp_waba_id,
       whatsapp_phone_number_id,
       whatsapp_connected_at
FROM estabelecimentos
WHERE id = $1
`
	var row struct {
		ID            string         `db:"id"`
		Enabled       bool           `db:"whatsapp_enabled"`
		Status        string         `db:"whatsapp_status"`
		WabaID        sql.NullString `db:"whatsapp_waba_id"`
		PhoneNumberID sql.NullString `db:"whatsapp_phone_number_id"`
		ConnectedAt   sql.NullTime   `db:"whatsapp_connected_at"`
	}
	if err := s.db.GetContext(ctx, &row, q, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar integração WhatsApp: %w", err)
	}

	view := &WhatsAppIntegrationView{
		EstabelecimentoID: row.ID,
		WhatsAppEnabled:   row.Enabled,
		Status:            row.Status,
	}
	if row.Enabled {
		view.State = BuildWhatsAppState(row.ID)
		view.SignupURL = BuildWhatsAppEmbeddedSignupURL(row.ID)
	}
	if row.WabaID.Valid {
		view.WabaID = &row.WabaID.String
	}
	if row.PhoneNumberID.Valid {
		view.PhoneNumberID = &row.PhoneNumberID.String
	}
	if row.ConnectedAt.Valid {
		t := row.ConnectedAt.Time
		view.ConnectedAt = &t
	}
	return view, nil
}

// MarkWhatsAppPending marca o fluxo de Embedded Signup como em andamento.
func (s *EstabelecimentoService) MarkWhatsAppPending(ctx context.Context, estabelecimentoID string) error {
	const q = `
UPDATE estabelecimentos
SET whatsapp_status = $2
WHERE id = $1
  AND whatsapp_status <> $3
`
	res, err := s.db.ExecContext(ctx, q, estabelecimentoID, WhatsAppStatusPendente, WhatsAppStatusConectado)
	if err != nil {
		return fmt.Errorf("marcar WhatsApp pendente: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		// Já conectado ou ID inexistente — confirma existência.
		var exists bool
		if err := s.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM estabelecimentos WHERE id = $1)`, estabelecimentoID); err != nil {
			return err
		}
		if !exists {
			return ErrEstabelecimentoNaoEncontrado
		}
	}
	return nil
}

// MarkWhatsAppConnected aplica o status CONECTADO após sucesso no Gateway.
func (s *EstabelecimentoService) MarkWhatsAppConnected(ctx context.Context, payload WhatsAppConnectedPayload) (string, error) {
	payload.Event = strings.TrimSpace(payload.Event)
	payload.SistemaOrigem = strings.ToLower(strings.TrimSpace(payload.SistemaOrigem))
	payload.TenantID = strings.TrimSpace(payload.TenantID)
	payload.SalonID = strings.TrimSpace(payload.SalonID)
	payload.State = strings.TrimSpace(payload.State)
	payload.EstabelecimentoID = strings.TrimSpace(payload.EstabelecimentoID)
	payload.WabaID = strings.TrimSpace(payload.WabaID)
	payload.PhoneNumberID = strings.TrimSpace(payload.PhoneNumberID)
	payload.WhatsAppPhoneNumber = strings.TrimSpace(payload.WhatsAppPhoneNumber)
	payload.Status = strings.ToLower(strings.TrimSpace(payload.Status))

	if payload.Event != "" && payload.Event != "whatsapp_connection_completed" {
		return "", fmt.Errorf("%w: event inesperado %q", ErrWebhookPayloadInvalido, payload.Event)
	}
	if payload.SistemaOrigem != "" && payload.SistemaOrigem != WhatsAppSistemaBeleza {
		return "", fmt.Errorf("%w: sistema_origem deve ser beleza", ErrWebhookPayloadInvalido)
	}

	id := payload.EstabelecimentoID
	if id == "" {
		id = payload.TenantID
	}
	if id == "" {
		id = payload.SalonID
	}
	if id == "" && payload.State != "" {
		parsed, err := ParseWhatsAppState(payload.State)
		if err != nil {
			return "", err
		}
		id = parsed
	}
	if id == "" {
		return "", fmt.Errorf("%w: informe tenant_id, salon_id, state ou estabelecimento_id", ErrWhatsAppStateInvalido)
	}

	if payload.State != "" {
		parsed, err := ParseWhatsAppState(payload.State)
		if err != nil {
			return "", err
		}
		if parsed != id {
			return "", fmt.Errorf("%w: state e tenant_id divergem", ErrWhatsAppStateInvalido)
		}
	}

	if payload.Status != "" && payload.Status != "connected" && payload.Status != "conectado" && payload.Status != "success" {
		return "", fmt.Errorf("%w: status deve indicar sucesso", ErrWebhookPayloadInvalido)
	}

	enabled, err := WhatsAppEnabledForTenant(ctx, s.db, id)
	if err != nil {
		return "", err
	}
	if !enabled {
		return "", ErrWhatsAppRecursoDesativado
	}

	const q = `
UPDATE estabelecimentos
SET whatsapp_status = $2,
    whatsapp_waba_id = NULLIF($3, ''),
    whatsapp_phone_number_id = NULLIF($4, ''),
    whatsapp_phone_number = NULLIF($5, ''),
    whatsapp_connected_at = NOW()
WHERE id = $1
RETURNING id
`
	var updated string
	err = s.db.QueryRowContext(
		ctx,
		q,
		id,
		WhatsAppStatusConectado,
		payload.WabaID,
		payload.PhoneNumberID,
		payload.WhatsAppPhoneNumber,
	).Scan(&updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrEstabelecimentoNaoEncontrado
		}
		return "", fmt.Errorf("conectar WhatsApp: %w", err)
	}
	return updated, nil
}
