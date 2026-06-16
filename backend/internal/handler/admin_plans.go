package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type AdminPlansHandler struct {
	planos     *service.PlanoSaasService
	saasGuard  *security.SaaSGuard
}

func NewAdminPlansHandler(planos *service.PlanoSaasService, saasGuard *security.SaaSGuard) *AdminPlansHandler {
	return &AdminPlansHandler{planos: planos, saasGuard: saasGuard}
}

type createPlanRequest struct {
	Nome                string  `json:"nome"`
	PrecoMensal         float64 `json:"preco_mensal"`
	LimiteProfissionais int     `json:"limite_profissionais"`
}

type createPlanResponse struct {
	ID string `json:"id"`
}

type assignPlanRequest struct {
	PlanoID string `json:"plano_id"`
	Meses   int    `json:"meses"`
}

type assignPlanResponse struct {
	EstabelecimentoID string `json:"estabelecimento_id"`
	PlanoID           string `json:"plano_id"`
	Meses             int    `json:"meses"`
	Status            string `json:"status"`
}

// List serve GET /api/v1/admin/plans
func (h *AdminPlansHandler) List(w http.ResponseWriter, r *http.Request) {
	planos, err := h.planos.ListSaasPlans(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, planos)
}

// Create serve POST /api/v1/admin/plans
func (h *AdminPlansHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	id, err := h.planos.CreateSaasPlan(r.Context(), req.Nome, req.PrecoMensal, req.LimiteProfissionais)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrLimiteProfissionaisInvalido):
			writeJSONError(w, http.StatusBadRequest, "invalid_professional_limit")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, createPlanResponse{ID: id})
}

// AssignPlan serve POST /api/v1/admin/establishments/{id}/assign-plan
func (h *AdminPlansHandler) AssignPlan(w http.ResponseWriter, r *http.Request) {
	establishmentID := r.PathValue("id")
	if establishmentID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}

	var req assignPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	if req.PlanoID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_plan_id")
		return
	}

	if err := h.planos.AssignPlanToEstablishment(r.Context(), establishmentID, req.PlanoID, req.Meses); err != nil {
		switch {
		case errors.Is(err, service.ErrEstabelecimentoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "establishment_not_found")
		case errors.Is(err, service.ErrPlanoSaasNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "plan_not_found")
		case errors.Is(err, service.ErrMesesContratacaoInvalidos):
			writeJSONError(w, http.StatusBadRequest, "invalid_months")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	if h.saasGuard != nil {
		h.saasGuard.InvalidateCache(establishmentID)
	}

	writeJSON(w, http.StatusOK, assignPlanResponse{
		EstabelecimentoID: establishmentID,
		PlanoID:           req.PlanoID,
		Meses:             req.Meses,
		Status:            "ATIVO",
	})
}
