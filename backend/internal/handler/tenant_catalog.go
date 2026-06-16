package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type TenantCatalogHandler struct {
	profissionais *service.ProfissionalService
	procedimentos *service.ProcedimentoService
}

func NewTenantCatalogHandler(
	profissionais *service.ProfissionalService,
	procedimentos *service.ProcedimentoService,
) *TenantCatalogHandler {
	return &TenantCatalogHandler{
		profissionais: profissionais,
		procedimentos: procedimentos,
	}
}

type createServiceRequest struct {
	Nome       string  `json:"nome"`
	PrecoBase  float64 `json:"preco_base"`
	DuracaoBase int    `json:"duracao_base_minutos"`
}

type createAdditionalRequest struct {
	Nome        string  `json:"nome"`
	PrecoAdd    float64 `json:"preco_adicional"`
	DuracaoAdd  int     `json:"duracao_adicional_minutos"`
}

type createProfessionalRequest struct {
	Nome          string  `json:"nome"`
	Especialidade string  `json:"especialidade"`
	Comissao      float64 `json:"comissao_porcentagem"`
}

type idResponse struct {
	ID string `json:"id"`
}

type limitReachedResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (h *TenantCatalogHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	servicos, err := h.procedimentos.ListServices(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, servicos)
}

func (h *TenantCatalogHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	var req createServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	id, err := h.procedimentos.CreateService(
		r.Context(),
		establishmentID,
		req.Nome,
		req.PrecoBase,
		req.DuracaoBase,
	)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

func (h *TenantCatalogHandler) CreateServiceAdditional(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	serviceID := r.PathValue("id")
	if serviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_service_id")
		return
	}

	var req createAdditionalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	id, err := h.procedimentos.CreateServiceAdditional(
		r.Context(),
		establishmentID,
		serviceID,
		req.Nome,
		req.PrecoAdd,
		req.DuracaoAdd,
	)
	if err != nil {
		if errors.Is(err, service.ErrServicoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "service_not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

func (h *TenantCatalogHandler) ListProfessionals(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	lista, err := h.profissionais.ListProfessionals(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, lista)
}

func (h *TenantCatalogHandler) CreateProfessional(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	var req createProfessionalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	id, err := h.profissionais.CreateProfessional(
		r.Context(),
		establishmentID,
		req.Nome,
		req.Especialidade,
		req.Comissao,
	)
	if err != nil {
		if errors.Is(err, service.ErrPlanLimitExceeded) {
			writeJSON(w, http.StatusForbidden, limitReachedResponse{
				Error:   "limit_reached",
				Message: "Seu plano atingiu o limite de profissionais parceiras permitidas. Faça um upgrade no painel.",
			})
			return
		}
		if errors.Is(err, service.ErrPlanoSaasNaoEncontrado) {
			writeJSONError(w, http.StatusPaymentRequired, "subscription_required")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}
