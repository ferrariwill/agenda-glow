package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/service"
)

type PublicSlotsHandler struct {
	agenda          *service.AgendaService
	estabelecimentos *service.EstabelecimentoService
}

func NewPublicSlotsHandler(
	agenda *service.AgendaService,
	estabelecimentos *service.EstabelecimentoService,
) *PublicSlotsHandler {
	return &PublicSlotsHandler{
		agenda:          agenda,
		estabelecimentos: estabelecimentos,
	}
}

// ServeHTTP atende GET /api/v1/public/{slug}/slots
func (h *PublicSlotsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		http.NotFound(w, r)
		return
	}

	est, err := h.estabelecimentos.BuscarPorSlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) || errors.Is(err, service.ErrSlugInvalido) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query()
	dataStr := strings.TrimSpace(query.Get("data"))
	profissionalID := strings.TrimSpace(query.Get("profissional_id"))
	procedimentoID := strings.TrimSpace(query.Get("procedimento_id"))

	if dataStr == "" || profissionalID == "" || procedimentoID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_query_params")
		return
	}

	date, err := time.ParseInLocation("2006-01-02", dataStr, time.Local)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_date")
		return
	}

	adicionais := parseAdicionaisQuery(query["adicionais"], query.Get("adicionais"))

	slots, err := h.agenda.GetAvailableSlots(
		r.Context(),
		est.ID,
		profissionalID,
		date,
		procedimentoID,
		adicionais,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProfissionalNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "professional_not_found")
		case errors.Is(err, service.ErrServicoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "service_not_found")
		case errors.Is(err, service.ErrAdicionalInvalido):
			writeJSONError(w, http.StatusBadRequest, "invalid_additionals")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	if slots == nil {
		slots = []string{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(slots)
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

func parseAdicionaisQuery(values []string, single string) []string {
	if len(values) == 0 && single == "" {
		return nil
	}

	if len(values) == 1 && strings.Contains(values[0], ",") {
		parts := strings.Split(values[0], ",")
		return trimNonEmpty(parts)
	}

	if len(values) > 0 {
		return trimNonEmpty(values)
	}

	if strings.Contains(single, ",") {
		return trimNonEmpty(strings.Split(single, ","))
	}

	return trimNonEmpty([]string{single})
}

func trimNonEmpty(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
