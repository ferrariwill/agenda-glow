package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/service"
)

type PublicAppointmentsHandler struct {
	agenda *service.AgendaService
}

func NewPublicAppointmentsHandler(agenda *service.AgendaService) *PublicAppointmentsHandler {
	return &PublicAppointmentsHandler{agenda: agenda}
}

func (h *PublicAppointmentsHandler) Manage(w http.ResponseWriter, r *http.Request) {
	view, err := h.agenda.GetAppointmentManagement(r.Context(), r.PathValue("token"), time.Now())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAgendamentoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "appointment_not_found")
		case errors.Is(err, service.ErrManagementTokenExpired):
			writeJSONError(w, http.StatusGone, "management_token_expired")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writePublicJSON(w, http.StatusOK, view)
}

func (h *PublicAppointmentsHandler) CancelByManagementToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"motivo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := h.agenda.CancelAppointmentByManagementToken(
		r.Context(), r.PathValue("token"), body.Reason, time.Now(),
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAgendamentoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "appointment_not_found")
		case errors.Is(err, service.ErrManagementTokenExpired):
			writeJSONError(w, http.StatusGone, "management_token_expired")
		case errors.Is(err, service.ErrCancellationReasonRequired):
			writeJSONError(w, http.StatusBadRequest, "reason_required")
		case errors.Is(err, service.ErrAppointmentNotCancellable):
			writeJSONError(w, http.StatusConflict, "appointment_not_cancellable")
		case errors.Is(err, service.ErrCancellationWindowClosed):
			var detail *service.CancellationWindowError
			if errors.As(err, &detail) {
				writePublicJSON(w, http.StatusUnprocessableEntity, map[string]any{
					"error":                "cancellation_window_closed",
					"minimum_notice_hours": detail.MinimumNoticeHours,
					"contact_phone":        detail.ContactPhone,
				})
			} else {
				writeJSONError(w, http.StatusUnprocessableEntity, "cancellation_window_closed")
			}
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writePublicJSON(w, http.StatusOK, result)
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

func writePublicJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
