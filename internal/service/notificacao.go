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
	whatsappTemplateLembrete   = "lembrete_agenda"
	whatsappSendTemplatePath   = "/v1/messages/send-template"
	defaultWhatsAppHTTPTimeout = 15 * time.Second
)

type whatsAppTemplateRequest struct {
	PhoneNumber      string   `json:"phone_number"`
	ExternalClientID string   `json:"external_client_id"`
	TemplateName     string   `json:"template_name"`
	AppointmentID    string   `json:"appointment_id"`
	Variables        []string `json:"variables"`
}

// DispararLembreteWhatsApp envia template de lembrete ao WhatsApp Gateway.
func DispararLembreteWhatsApp(
	ctx context.Context,
	telefoneCliente, nomeCliente, nomeProfissional, servico, horario, agendamentoID, idSalaoCliente string,
) error {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WHATSAPP_GATEWAY_URL")), "/")
	apiKey := strings.TrimSpace(os.Getenv("WHATSAPP_GATEWAY_KEY"))
	if baseURL == "" || apiKey == "" {
		return fmt.Errorf("whatsapp gateway não configurado (WHATSAPP_GATEWAY_URL / WHATSAPP_GATEWAY_KEY)")
	}

	telefoneCliente = strings.TrimSpace(telefoneCliente)
	if telefoneCliente == "" {
		return fmt.Errorf("telefone do cliente vazio")
	}

	payload := whatsAppTemplateRequest{
		PhoneNumber:      telefoneCliente,
		ExternalClientID: strings.TrimSpace(idSalaoCliente),
		TemplateName:     whatsappTemplateLembrete,
		AppointmentID:    strings.TrimSpace(agendamentoID),
		Variables: []string{
			strings.TrimSpace(nomeCliente),
			strings.TrimSpace(nomeProfissional),
			strings.TrimSpace(servico),
			strings.TrimSpace(horario),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("serializar payload whatsapp: %w", err)
	}

	endpoint := baseURL + whatsappSendTemplatePath
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
