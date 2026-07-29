package handler

import (
	"errors"
	"net/http"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type WhatsAppIntegrationHandler struct {
	estabelecimentos *service.EstabelecimentoService
}

func NewWhatsAppIntegrationHandler(estabelecimentos *service.EstabelecimentoService) *WhatsAppIntegrationHandler {
	return &WhatsAppIntegrationHandler{estabelecimentos: estabelecimentos}
}

// GetIntegration GET /api/v1/whatsapp/integration — status + URL Embedded Signup.
func (h *WhatsAppIntegrationHandler) GetIntegration(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok || establishmentID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	view, err := h.estabelecimentos.GetWhatsAppIntegration(r.Context(), establishmentID)
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "establishment_not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, view)
}

// StartConnection POST /api/v1/whatsapp/integration/start — marca PENDENTE ao abrir o signup.
func (h *WhatsAppIntegrationHandler) StartConnection(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok || establishmentID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.estabelecimentos.MarkWhatsAppPending(r.Context(), establishmentID); err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "establishment_not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	view, err := h.estabelecimentos.GetWhatsAppIntegration(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

