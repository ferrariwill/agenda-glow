package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type EarlySlotHandler struct {
	service *service.EarlySlotService
}

func NewEarlySlotHandler(svc *service.EarlySlotService) *EarlySlotHandler {
	return &EarlySlotHandler{service: svc}
}

func (h *EarlySlotHandler) GetOffer(w http.ResponseWriter, r *http.Request) {
	view, err := h.service.GetOffer(r.Context(), r.PathValue("token"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (h *EarlySlotHandler) Accept(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Accept(r.Context(), r.PathValue("token"), "WEB")
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *EarlySlotHandler) Decline(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.Decline(r.Context(), r.PathValue("token"), "WEB")
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h *EarlySlotHandler) GetRound(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	round, err := h.service.GetRound(r.Context(), tenantID, r.PathValue("id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, round)
}

func (h *EarlySlotHandler) ListRounds(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	rounds, err := h.service.ListRounds(
		r.Context(), tenantID,
		strings.TrimSpace(r.URL.Query().Get("profissional_id")),
		strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))),
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, rounds)
}

func (h *EarlySlotHandler) SetPreference(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	var req struct {
		Enabled *bool `json:"aceita_adiantar"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Enabled == nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	at, err := h.service.SetPreference(r.Context(), tenantID, r.PathValue("id"), *req.Enabled)
	if err != nil {
		h.writeError(w, err)
		return
	}
	response := map[string]any{"aceita_adiantar": *req.Enabled, "aceita_adiantar_em": nil}
	response["early_slot_notifications_available"] =
		h.service.NotificationsAvailable(r.Context(), tenantID)
	if *req.Enabled {
		response["aceita_adiantar_em"] = at
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *EarlySlotHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEarlySlotOfferNotFound):
		writeJSONError(w, http.StatusNotFound, "offer_not_found")
	case errors.Is(err, service.ErrEarlySlotOfferExpired):
		writeJSONError(w, http.StatusGone, "offer_expired")
	case errors.Is(err, service.ErrEarlySlotOfferUnavailable):
		writeJSONError(w, http.StatusConflict, "offer_no_longer_available")
	case errors.Is(err, service.ErrEarlySlotAppointmentIneligible):
		writeJSONError(w, http.StatusUnprocessableEntity, "appointment_no_longer_eligible")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
	}
}
