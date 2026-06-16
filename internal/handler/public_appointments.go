package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/agendaglow/agendaglow/internal/service"
)

type PublicAppointmentsHandler struct {
	agenda *service.AgendaService
}

func NewPublicAppointmentsHandler(agenda *service.AgendaService) *PublicAppointmentsHandler {
	return &PublicAppointmentsHandler{agenda: agenda}
}

// Approve confirma encaixe: POST /api/v1/public/appointments/{id}/approve
func (h *PublicAppointmentsHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if err := h.agenda.ApproveAppointment(r.Context(), id); err != nil {
		mapAppointmentError(w, err)
		return
	}

	respondAppointmentAction(w, "approved", "Agendamento confirmado com sucesso.")
}

// Reschedule move para horário livre: POST /api/v1/public/appointments/{id}/reschedule?new_time=HH:MM
func (h *PublicAppointmentsHandler) Reschedule(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.NotFound(w, r)
		return
	}

	newTime := strings.TrimSpace(r.URL.Query().Get("new_time"))
	if err := h.agenda.RescheduleAppointment(r.Context(), id, newTime); err != nil {
		mapAppointmentError(w, err)
		return
	}

	respondAppointmentAction(w, "rescheduled", "Horário atualizado e agendamento confirmado.")
}

func mapAppointmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAgendamentoNaoEncontrado):
		writeJSONError(w, http.StatusNotFound, "appointment_not_found")
	case errors.Is(err, service.ErrAgendamentoNaoEmAprovacao):
		writeJSONError(w, http.StatusConflict, "appointment_not_pending")
	case errors.Is(err, service.ErrHorarioIndisponivel):
		writeJSONError(w, http.StatusConflict, "slot_unavailable")
	case errors.Is(err, service.ErrHorarioInvalido):
		writeJSONError(w, http.StatusBadRequest, "invalid_time")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
	}
}

func respondAppointmentAction(w http.ResponseWriter, status, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  status,
		"message": message,
	})
}
