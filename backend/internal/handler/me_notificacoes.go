package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type MeNotificacoesHandler struct {
	avisos *service.AvisoService
}

func NewMeNotificacoesHandler(avisos *service.AvisoService) *MeNotificacoesHandler {
	return &MeNotificacoesHandler{avisos: avisos}
}

// List serve GET /api/v1/me/notificacoes
func (h *MeNotificacoesHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := h.avisos.ListInbox(r.Context(), claims.UserID, claims.Role, claims.EstabelecimentoID, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// MarkRead serve POST /api/v1/me/notificacoes/{id}/read
func (h *MeNotificacoesHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, http.StatusNotFound, "aviso_not_found")
		return
	}

	item, err := h.avisos.MarkRead(r.Context(), claims.UserID, claims.Role, id, claims.EstabelecimentoID)
	if err != nil {
		writeAvisoServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
