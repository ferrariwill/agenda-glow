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
	Nome            string   `json:"nome"`
	PrecoBase       float64  `json:"preco_base"`
	DuracaoBase     int      `json:"duracao_base_minutos"`
	ProfissionalIDs []string `json:"profissional_ids"`
}

type updateServiceRequest struct {
	Nome            string    `json:"nome"`
	PrecoBase       float64   `json:"preco_base"`
	DuracaoBase     int       `json:"duracao_base_minutos"`
	Ativo           bool      `json:"ativo"`
	ProfissionalIDs *[]string `json:"profissional_ids"`
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

	id, err := h.procedimentos.CreateService(
		r.Context(),
		establishmentID,
		req.Nome,
		req.PrecoBase,
		req.DuracaoBase,
		req.ProfissionalIDs,
	)
	if err != nil {
		if errors.Is(err, service.ErrProfissionalVinculoInvalido) {
			writeJSONError(w, http.StatusBadRequest, "invalid_professional")
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
