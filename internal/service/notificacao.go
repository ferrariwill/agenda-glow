package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	whatsappSistemaOrigem      = "beleza"
	whatsappTemplateLembrete   = "lembrete_agenda"
	whatsappSendNotificationPath = "/send-notification"
	defaultWhatsAppHTTPTimeout = 15 * time.Second
	defaultWhatsAppLanguage    = "pt_BR"
)

// whatsAppSendNotificationRequest é o contrato Beleza → Gateway.
type whatsAppSendNotificationRequest struct {
	SistemaOrigem  string   `json:"sistema_origem"`
	TenantID       string   `json:"tenant_id"`
	PhoneNumber    string   `json:"phone_number"`
	TemplateName   string   `json:"template_name"`
	LanguageCode   string   `json:"language_code"`
	SimpleTemplate bool     `json:"simple_template"`
	AppointmentID  string   `json:"appointment_id,omitempty"`
	Variables      []string `json:"variables"`
}

// DispararLembreteWhatsApp envia template de lembrete via Gateway (POST /send-notification).
func DispararLembreteWhatsApp(
	ctx context.Context,
	telefoneCliente, nomeCliente, nomeProfissional, servico, horario, agendamentoID, idSalaoCliente string,
) error {
	return EnviarNotificacaoWhatsApp(ctx, WhatsAppNotificationInput{
		TenantID:      idSalaoCliente,
		PhoneNumber:   telefoneCliente,
		TemplateName:  whatsappTemplateLembrete,
		AppointmentID: agendamentoID,
		Variables: []string{
			strings.TrimSpace(nomeCliente),
			strings.TrimSpace(nomeProfissional),
			strings.TrimSpace(servico),
			strings.TrimSpace(horario),
		},
	})
}

// WhatsAppNotificationInput parâmetros para POST /send-notification no Gateway.
type WhatsAppNotificationInput struct {
	TenantID       string
	PhoneNumber    string
	TemplateName   string
	LanguageCode   string
	SimpleTemplate bool
	AppointmentID  string
	Variables      []string
}

// EnviarNotificacaoWhatsApp chama o Gateway WhatsApp (cliente Beleza → Gateway).
func EnviarNotificacaoWhatsApp(ctx context.Context, in WhatsAppNotificationInput) error {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WHATSAPP_GATEWAY_URL")), "/")
	apiKey := strings.TrimSpace(os.Getenv("WHATSAPP_GATEWAY_KEY"))
	if baseURL == "" || apiKey == "" {
		return fmt.Errorf("whatsapp gateway não configurado (WHATSAPP_GATEWAY_URL / WHATSAPP_GATEWAY_KEY)")
	}

	phone := strings.TrimSpace(in.PhoneNumber)
	if phone == "" {
		return fmt.Errorf("telefone do cliente vazio")
	}
	tenantID := strings.TrimSpace(in.TenantID)
	if tenantID == "" {
		return fmt.Errorf("tenant_id (salão) vazio")
	}
	template := strings.TrimSpace(in.TemplateName)
	if template == "" {
		return fmt.Errorf("template_name vazio")
	}
	lang := strings.TrimSpace(in.LanguageCode)
	if lang == "" {
		lang = defaultWhatsAppLanguage
	}

	payload := whatsAppSendNotificationRequest{
		SistemaOrigem:  whatsappSistemaOrigem,
		TenantID:       tenantID,
		PhoneNumber:    phone,
		TemplateName:   template,
		LanguageCode:   lang,
		SimpleTemplate: true,
		AppointmentID:  strings.TrimSpace(in.AppointmentID),
		Variables:      in.Variables,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializar payload whatsapp: %w", err)
	}

	endpoint := baseURL + whatsappSendNotificationPath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("montar requisição whatsapp: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: defaultWhatsAppHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("chamar whatsapp gateway: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("whatsapp gateway status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func (s *AgendaService) dispararLembreteWhatsAppAgendamento(agendamentoID string) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultWhatsAppHTTPTimeout)
	defer cancel()

	ag, err := s.buscarAgendamento(ctx, agendamentoID)
	if err != nil {
		log.Printf("whatsapp lembrete: buscar agendamento %s: %v", agendamentoID, err)
		return
	}

	horario := ag.DataHoraInicio.Format("02/01/2006 15:04")
	if err := DispararLembreteWhatsApp(
		ctx,
		ag.ClienteTelefone,
		ag.ClienteNome,
		ag.ProfissionalNome,
		ag.ServicoNome,
		horario,
		ag.ID,
		ag.EstabelecimentoID,
	); err != nil {
		log.Printf("whatsapp lembrete: agendamento %s: %v", agendamentoID, err)
	}
}
