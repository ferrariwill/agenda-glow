package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/agendaglow/agendaglow/internal/service"
)

type WhatsAppWebhookHandler struct {
	agenda *service.AgendaService
}

func NewWhatsAppWebhookHandler(agenda *service.AgendaService) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{agenda: agenda}
}

// Callback processa POST /api/v1/webhook/whatsapp-callback
func (h *WhatsAppWebhookHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := validateWhatsAppWebhookKey(r); err != nil {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload service.WhatsAppCallbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
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
		"status":              "processed",
		"appointment_status":  status,
	})
}

func validateWhatsAppWebhookKey(r *http.Request) error {
	expected := strings.TrimSpace(os.Getenv("WHATSAPP_GATEWAY_KEY"))
	if expected == "" {
		return nil
	}
	if strings.TrimSpace(r.Header.Get("X-API-Key")) != expected {
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
