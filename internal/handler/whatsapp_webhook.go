package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/agendaglow/agendaglow/internal/service"
)

type WhatsAppWebhookHandler struct {
	agenda           *service.AgendaService
	estabelecimentos *service.EstabelecimentoService
}

func NewWhatsAppWebhookHandler(
	agenda *service.AgendaService,
	estabelecimentos *service.EstabelecimentoService,
) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{
		agenda:           agenda,
		estabelecimentos: estabelecimentos,
	}
}

// Gateway processa POST /api/v1/webhook/whatsapp-gateway
// URL recomendada em BELEZA_SAAS_WEBHOOK_URL — despacha conexão ou resposta de botão.
func (h *WhatsAppWebhookHandler) Gateway(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := validateWhatsAppWebhookKey(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	var peek struct {
		Event     string `json:"event"`
		EventType string `json:"event_type"`
		Action    string `json:"action"`
	}
	if err := json.Unmarshal(body, &peek); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	event := strings.TrimSpace(peek.Event)
	action := strings.ToUpper(strings.TrimSpace(peek.Action))
	eventType := strings.TrimSpace(peek.EventType)

	switch {
	case event == "whatsapp_connection_completed":
		h.handleConnectedBody(w, r, body)
	case action == service.WhatsAppActionConfirm || action == service.WhatsAppActionCancel || eventType == "button_reply":
		h.handleCallbackBody(w, r, body)
	default:
		writeJSONError(w, http.StatusBadRequest, "unknown_gateway_event")
	}
}

// Callback processa POST /api/v1/webhook/whatsapp-callback (respostas CONFIRM/CANCEL).
func (h *WhatsAppWebhookHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := validateWhatsAppWebhookKey(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	h.handleCallbackBody(w, r, body)
}

// Connected processa POST /api/v1/webhook/whatsapp-connected (Embedded Signup concluído).
func (h *WhatsAppWebhookHandler) Connected(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := validateWhatsAppWebhookKey(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	h.handleConnectedBody(w, r, body)
}

func (h *WhatsAppWebhookHandler) handleCallbackBody(w http.ResponseWriter, r *http.Request, body []byte) {
	var payload service.WhatsAppCallbackPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	status, err := h.agenda.ProcessWhatsAppCallback(r.Context(), payload)
	if err != nil {
		mapWhatsAppWebhookError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":             "processed",
		"appointment_status": status,
	})
}

func (h *WhatsAppWebhookHandler) handleConnectedBody(w http.ResponseWriter, r *http.Request, body []byte) {
	if h.estabelecimentos == nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	var payload service.WhatsAppConnectedPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	id, err := h.estabelecimentos.MarkWhatsAppConnected(r.Context(), payload)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrWhatsAppRecursoDesativado):
			log.Printf("webhook whatsapp-connected recusado: recurso desativado tenant=%s",
				whatsAppTenantHint(payload))
			writeJSONErrorMessage(w, http.StatusForbidden, "whatsapp_feature_disabled",
				"Recurso de WhatsApp desativado para este estabelecimento. Contate o administrador")
		case errors.Is(err, service.ErrWhatsAppStateInvalido),
			errors.Is(err, service.ErrWebhookPayloadInvalido):
			writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		case errors.Is(err, service.ErrEstabelecimentoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "establishment_not_found")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":             "connected",
		"estabelecimento_id": id,
		"integration_status": service.WhatsAppStatusConectado,
	})
}

func validateWhatsAppWebhookKey(r *http.Request) error {
	expected := strings.TrimSpace(os.Getenv("WHATSAPP_GATEWAY_KEY"))
	if expected == "" {
		return nil
	}
	got := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if got == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			got = strings.TrimSpace(auth[7:])
		}
	}
	// Gateway pode não enviar chave no webhook de callback; se enviar, deve bater.
	if got == "" {
		return nil
	}
	if got != expected {
		return errors.New("chave inválida")
	}
	return nil
}

func mapWhatsAppWebhookError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrWebhookPayloadInvalido),
		errors.Is(err, service.ErrAcaoWhatsAppInvalida):
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
	case errors.Is(err, service.ErrAgendamentoNaoEncontrado):
		writeJSONError(w, http.StatusNotFound, "appointment_not_found")
	case errors.Is(err, service.ErrAgendamentoEscopoInvalido):
		writeJSONError(w, http.StatusForbidden, "appointment_scope_mismatch")
	case errors.Is(err, service.ErrAgendamentoCancelado),
		errors.Is(err, service.ErrAgendamentoStatusFinal),
		errors.Is(err, service.ErrAgendamentoJaConcluido):
		writeJSONError(w, http.StatusConflict, "appointment_not_updatable")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
	}
}

func whatsAppTenantHint(payload service.WhatsAppConnectedPayload) string {
	for _, value := range []string{
		payload.EstabelecimentoID,
		payload.TenantID,
		payload.SalonID,
		strings.TrimPrefix(payload.State, service.WhatsAppStatePrefix+"_"),
	} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "desconhecido"
}

func writeJSONErrorMessage(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}
