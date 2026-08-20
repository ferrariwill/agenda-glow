package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

// EstoqueHandler cobre ficha técnica (BOM) e previsão de estoque.
type EstoqueHandler struct {
	bom      *service.ServicoInsumoService
	previsao *service.EstoquePrevisaoService
}

func NewEstoqueHandler(
	bom *service.ServicoInsumoService,
	previsao *service.EstoquePrevisaoService,
) *EstoqueHandler {
	return &EstoqueHandler{bom: bom, previsao: previsao}
}

type replaceBOMRequest struct {
	Itens []service.ServicoInsumoInput `json:"itens"`
}

// ListServiceSupplies GET /api/v1/services/{id}/supplies
func (h *EstoqueHandler) ListServiceSupplies(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	servicoID := r.PathValue("id")
	if strings.TrimSpace(servicoID) == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_service_id")
		return
	}

	list, err := h.bom.List(r.Context(), establishmentID, servicoID)
	if err != nil {
		if errors.Is(err, service.ErrServicoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ReplaceServiceSupplies PUT /api/v1/services/{id}/supplies
func (h *EstoqueHandler) ReplaceServiceSupplies(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	servicoID := r.PathValue("id")
	if strings.TrimSpace(servicoID) == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_service_id")
		return
	}

	var req replaceBOMRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.Itens == nil {
		req.Itens = []service.ServicoInsumoInput{}
	}

	if err := h.bom.Replace(r.Context(), establishmentID, servicoID, req.Itens); err != nil {
		switch {
		case errors.Is(err, service.ErrServicoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "not_found")
		case errors.Is(err, service.ErrServicoInsumoInvalido),
			errors.Is(err, service.ErrInsumoBOMNaoEncontrado):
			writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ForecastSupplies GET /api/v1/supplies/forecast
func (h *EstoqueHandler) ForecastSupplies(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	janela, err := parseOptionalIntOrDefault(r.URL.Query().Get("janela_dias"), service.DefaultJanelaDias)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_query")
		return
	}
	horizonte, err := parseOptionalIntOrDefault(r.URL.Query().Get("horizonte_critico_dias"), service.DefaultHorizonteCriticoDias)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_query")
		return
	}
	apenasCriticos, err := parseOptionalBool(r.URL.Query().Get("apenas_criticos"), false)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_query")
		return
	}

	params, err := service.NormalizePrevisaoParams(janela, horizonte, apenasCriticos)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_query")
		return
	}

	result, err := h.previsao.Forecast(r.Context(), establishmentID, params)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// parseOptionalIntOrDefault: ausente → default; presente → valor parseado (0/negativo
// seguem para NormalizePrevisaoParams, que rejeita < 1).
func parseOptionalIntOrDefault(raw string, defaultVal int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func parseOptionalBool(raw string, defaultVal bool) (bool, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return defaultVal, nil
	}
	switch raw {
	case "1", "true", "t", "yes", "y":
		return true, nil
	case "0", "false", "f", "no", "n":
		return false, nil
	default:
		return false, errors.New("invalid bool")
	}
}
