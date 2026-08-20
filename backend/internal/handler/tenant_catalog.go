package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	storagesvc "github.com/agendaglow/agendaglow/backend/internal/service"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type TenantCatalogHandler struct {
	profissionais  *service.ProfissionalService
	procedimentos  *service.ProcedimentoService
	servicoInsumos *service.ServicoInsumoService
	categorias     *service.CategoriaServicoService
}

func NewTenantCatalogHandler(
	profissionais *service.ProfissionalService,
	procedimentos *service.ProcedimentoService,
	servicoInsumos *service.ServicoInsumoService,
	categorias *service.CategoriaServicoService,
) *TenantCatalogHandler {
	return &TenantCatalogHandler{
		profissionais:  profissionais,
		procedimentos:  procedimentos,
		servicoInsumos: servicoInsumos,
		categorias:     categorias,
	}
}

type createServiceRequest struct {
	Nome            string   `json:"nome"`
	PrecoBase       float64  `json:"preco_base"`
	DuracaoBase     int      `json:"duracao_base_minutos"`
	ProfissionalIDs []string `json:"profissional_ids"`
	CategoriaID     *string  `json:"categoria_id"`
}

type updateServiceRequest struct {
	Nome            string    `json:"nome"`
	PrecoBase       float64   `json:"preco_base"`
	DuracaoBase     int       `json:"duracao_base_minutos"`
	Ativo           bool      `json:"ativo"`
	ProfissionalIDs *[]string `json:"profissional_ids"`
	CategoriaID     *string   `json:"categoria_id"`
}

type createAdditionalRequest struct {
	Nome       string  `json:"nome"`
	PrecoAdd   float64 `json:"preco_adicional"`
	DuracaoAdd int     `json:"duracao_adicional_minutos"`
}

type createProfessionalRequest struct {
	Nome            string  `json:"nome"`
	EspecialidadeID string  `json:"especialidade_id"`
	Comissao        float64 `json:"comissao_porcentagem"`
	DataNascimento  *string `json:"data_nascimento"`
	FotoURL         *string `json:"foto_url"`
}

type serviceCategoryRequest struct {
	Nome  string `json:"nome"`
	Icone string `json:"icone"`
}

type serviceSupplyRequest struct {
	InsumoID      string  `json:"insumo_id"`
	QuantidadeUso float64 `json:"quantidade_uso"`
}

type updateServiceSupplyRequest struct {
	QuantidadeUso float64 `json:"quantidade_uso"`
}

type idResponse struct {
	ID string `json:"id"`
}

type limitReachedResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type professionalFotoResponse struct {
	ID      string `json:"id"`
	FotoURL string `json:"foto_url"`
}

const maxProfessionalFotoBytes = 2 << 20 // 2MB

var extensoesFotoProfissional = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
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

	categoriaID := ""
	if req.CategoriaID != nil {
		categoriaID = *req.CategoriaID
	}

	id, err := h.procedimentos.CreateService(
		r.Context(),
		establishmentID,
		req.Nome,
		req.PrecoBase,
		req.DuracaoBase,
		req.ProfissionalIDs,
		categoriaID,
	)
	if err != nil {
		if errors.Is(err, service.ErrProfissionalVinculoInvalido) {
			writeJSONError(w, http.StatusBadRequest, "invalid_professional")
			return
		}
		if errors.Is(err, service.ErrCategoriaServicoNaoEncontrada) {
			writeJSONError(w, http.StatusBadRequest, "invalid_category")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

func (h *TenantCatalogHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	serviceID := strings.TrimSpace(r.PathValue("id"))
	if serviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_service_id")
		return
	}

	var req updateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	serv, err := h.procedimentos.UpdateService(
		r.Context(),
		establishmentID,
		serviceID,
		service.UpdateServiceInput{
			Nome:            req.Nome,
			PrecoBase:       req.PrecoBase,
			DuracaoBase:     req.DuracaoBase,
			Ativo:           req.Ativo,
			ProfissionalIDs: req.ProfissionalIDs,
			CategoriaID:     req.CategoriaID,
		},
	)
	if err != nil {
		if errors.Is(err, service.ErrServicoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "service_not_found")
			return
		}
		if errors.Is(err, service.ErrProfissionalVinculoInvalido) {
			writeJSONError(w, http.StatusBadRequest, "invalid_professional")
			return
		}
		if errors.Is(err, service.ErrCategoriaServicoNaoEncontrada) {
			writeJSONError(w, http.StatusBadRequest, "invalid_category")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	writeJSON(w, http.StatusOK, serv)
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

func (h *TenantCatalogHandler) ListServiceCategories(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	list, err := h.categorias.List(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *TenantCatalogHandler) CreateServiceCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	var req serviceCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	id, err := h.categorias.Create(r.Context(), establishmentID, req.Nome, req.Icone)
	if err != nil {
		if errors.Is(err, service.ErrCategoriaServicoNomeDuplicado) {
			writeJSONError(w, http.StatusConflict, "duplicate_name")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

func (h *TenantCatalogHandler) UpdateServiceCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}
	var req serviceCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := h.categorias.Update(r.Context(), establishmentID, id, req.Nome, req.Icone); err != nil {
		if errors.Is(err, service.ErrCategoriaServicoNaoEncontrada) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		if errors.Is(err, service.ErrCategoriaServicoNomeDuplicado) {
			writeJSONError(w, http.StatusConflict, "duplicate_name")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TenantCatalogHandler) DeleteServiceCategory(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}
	if err := h.categorias.Delete(r.Context(), establishmentID, id); err != nil {
		if errors.Is(err, service.ErrCategoriaServicoNaoEncontrada) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *TenantCatalogHandler) ListServiceSupplies(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	serviceID := strings.TrimSpace(r.PathValue("id"))
	if serviceID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_service_id")
		return
	}
	list, err := h.servicoInsumos.List(r.Context(), establishmentID, serviceID)
	if err != nil {
		if errors.Is(err, service.ErrServicoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *TenantCatalogHandler) CreateServiceSupply(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	serviceID := strings.TrimSpace(r.PathValue("id"))
	var req serviceSupplyRequest
	if serviceID == "" || json.NewDecoder(r.Body).Decode(&req) != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	id, err := h.servicoInsumos.Create(r.Context(), establishmentID, serviceID, req.InsumoID, req.QuantidadeUso)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrServicoNaoEncontrado), errors.Is(err, service.ErrInsumoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "not_found")
		case errors.Is(err, service.ErrServicoInsumoDuplicado):
			writeJSONError(w, http.StatusConflict, "duplicate_link")
		default:
			writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		}
		return
	}
	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

func (h *TenantCatalogHandler) UpdateServiceSupply(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	serviceID, linkID := strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("linkId"))
	var req updateServiceSupplyRequest
	if serviceID == "" || linkID == "" || json.NewDecoder(r.Body).Decode(&req) != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	if err := h.servicoInsumos.Update(r.Context(), establishmentID, serviceID, linkID, req.QuantidadeUso); err != nil {
		if errors.Is(err, service.ErrServicoInsumoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
		} else {
			writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *TenantCatalogHandler) DeleteServiceSupply(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	serviceID, linkID := strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("linkId"))
	if serviceID == "" || linkID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}
	if err := h.servicoInsumos.Delete(r.Context(), establishmentID, serviceID, linkID); err != nil {
		if errors.Is(err, service.ErrServicoInsumoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
		} else {
			writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
		req.EspecialidadeID,
		req.Comissao,
		req.DataNascimento,
		req.FotoURL,
	)
	if err != nil {
		if errors.Is(err, service.ErrPlanLimitExceeded) {
			writeJSON(w, http.StatusForbidden, limitReachedResponse{
				Error:   "limit_reached",
				Message: "Seu plano atingiu o limite de profissionais parceiras permitidas. Faça um upgrade no painel.",
			})
			return
		}
		if errors.Is(err, service.ErrDataNascimentoFutura) {
			writeJSONError(w, http.StatusBadRequest, "birthdate_in_future")
			return
		}
		if errors.Is(err, service.ErrDataNascimentoInvalida) {
			writeJSONError(w, http.StatusBadRequest, "invalid_birthdate")
			return
		}
		if errors.Is(err, service.ErrEspecialidadeNaoEncontrada) || errors.Is(err, service.ErrEspecialidadeInativa) {
			writeJSONError(w, http.StatusBadRequest, "invalid_specialty")
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

// UploadProfessionalFoto POST /api/v1/professionals/{id}/foto (multipart field "foto").
func (h *TenantCatalogHandler) UploadProfessionalFoto(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	profID := strings.TrimSpace(r.PathValue("id"))
	if profID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_professional_id")
		return
	}

	if err := r.ParseMultipartForm(maxProfessionalFotoBytes); err != nil {
		writeJSONErrorMessage(w, http.StatusBadRequest, "invalid_image", "Arquivo excede o limite de 2MB ou formulário inválido")
		return
	}

	file, header, err := r.FormFile("foto")
	if err != nil {
		writeJSONErrorMessage(w, http.StatusBadRequest, "invalid_image", "Campo foto é obrigatório")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !extensoesFotoProfissional[ext] {
		writeJSONErrorMessage(w, http.StatusBadRequest, "invalid_image", "Formato inválido. Use .png, .jpg, .jpeg ou .webp")
		return
	}

	limited := io.LimitReader(file, maxProfessionalFotoBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		writeJSONErrorMessage(w, http.StatusBadRequest, "invalid_image", "Erro ao ler arquivo de foto")
		return
	}
	if int64(len(data)) > maxProfessionalFotoBytes {
		writeJSONErrorMessage(w, http.StatusBadRequest, "invalid_image", "Foto excede o limite de 2MB")
		return
	}

	if _, err := h.profissionais.BuscarProfissionalPorID(r.Context(), establishmentID, profID); err != nil {
		if errors.Is(err, service.ErrProfissionalNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "professional_not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	objectPath := fmt.Sprintf("%s/%s-%d%s", establishmentID, profID, time.Now().UnixNano(), ext)
	publicURL, err := storagesvc.UploadAvatarToSupabase(r.Context(), bytes.NewReader(data), objectPath)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "storage_upload_failed")
		return
	}

	if err := h.profissionais.SetFotoURL(r.Context(), establishmentID, profID, publicURL); err != nil {
		if errors.Is(err, service.ErrProfissionalNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "professional_not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, professionalFotoResponse{ID: profID, FotoURL: publicURL})
}
