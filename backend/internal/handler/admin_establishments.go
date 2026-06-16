package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/agendaglow/agendaglow/internal/service"
)

type AdminEstablishmentsHandler struct {
	estabelecimentos *service.EstabelecimentoService
}

func NewAdminEstablishmentsHandler(estabelecimentos *service.EstabelecimentoService) *AdminEstablishmentsHandler {
	return &AdminEstablishmentsHandler{estabelecimentos: estabelecimentos}
}

type createEstablishmentRequest struct {
	NomeComercial string `json:"nome_comercial"`
	Slug          string `json:"slug"`
}

type createEstablishmentResponse struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
}

type toggleStatusRequest struct {
	Ativo bool `json:"ativo"`
}

// List serve GET /api/v1/admin/establishments
func (h *AdminEstablishmentsHandler) List(w http.ResponseWriter, r *http.Request) {
	lista, err := h.estabelecimentos.ListAllEstablishments(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, lista)
}

// Create serve POST /api/v1/admin/establishments
func (h *AdminEstablishmentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createEstablishmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	id, slugFinal, err := h.estabelecimentos.RegisterEstablishment(r.Context(), req.NomeComercial, req.Slug)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSlugAlreadyExists):
			writeJSONError(w, http.StatusConflict, "slug_already_exists")
		case errors.Is(err, service.ErrSlugInvalido):
			writeJSONError(w, http.StatusBadRequest, "invalid_slug")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, createEstablishmentResponse{
		ID:   id,
		Slug: slugFinal,
	})
}

// ToggleStatus serve PUT /api/v1/admin/establishments/{id}/status
func (h *AdminEstablishmentsHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}

	var req toggleStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	if err := h.estabelecimentos.ToggleEstablishmentStatus(r.Context(), id, req.Ativo); err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ativo": req.Ativo})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
