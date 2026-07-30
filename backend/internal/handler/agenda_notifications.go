package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type AgendaNotificationsHandler struct {
	agenda *service.AgendaService
}

func NewAgendaNotificationsHandler(agenda *service.AgendaService) *AgendaNotificationsHandler {
	return &AgendaNotificationsHandler{agenda: agenda}
}

func (h *AgendaNotificationsHandler) Get(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := notificationSettingsScope(w, r)
	if !ok {
		return
	}
	out, err := h.agenda.GetNotificationAgendaConfig(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AgendaNotificationsHandler) Put(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := notificationSettingsScope(w, r)
	if !ok {
		return
	}
	var input service.NotificationAgendaConfig
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_notification_settings")
		return
	}
	out, err := h.agenda.UpdateNotificationAgendaConfig(r.Context(), establishmentID, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidNotificationSettings) {
			writeJSONError(w, http.StatusBadRequest, "invalid_notification_settings")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func notificationSettingsScope(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims.EstabelecimentoID == nil {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return "", false
	}
	claimID := strings.TrimSpace(*claims.EstabelecimentoID)
	if strings.TrimSpace(r.PathValue("id")) != claimID {
		writeJSONError(w, http.StatusForbidden, "establishment_scope_mismatch")
		return "", false
	}
	return claimID, true
}
