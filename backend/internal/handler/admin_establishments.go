package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	storagesvc "github.com/agendaglow/agendaglow/backend/internal/service"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type AdminEstablishmentsHandler struct {
	estabelecimentos *service.EstabelecimentoService
	auth             *service.AuthService
	whatsAppGate     *security.WhatsAppGate
}

func NewAdminEstablishmentsHandler(
	estabelecimentos *service.EstabelecimentoService,
	auth *service.AuthService,
	whatsAppGate *security.WhatsAppGate,
) *AdminEstablishmentsHandler {
	return &AdminEstablishmentsHandler{
		estabelecimentos: estabelecimentos,
		auth:             auth,
		whatsAppGate:     whatsAppGate,
	}
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

type toggleWhatsAppRequest struct {
	WhatsAppEnabled *bool `json:"whatsapp_enabled"`
}

type createDonaRequest struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type createDonaResponse struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	Nome         string `json:"nome"`
	SenhaInicial string `json:"senha_inicial"`
}

const senhaInicialDona = "AgendaGlow@2026"

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

// CreateDona serve POST /api/v1/admin/establishments/{id}/create-dona
func (h *AdminEstablishmentsHandler) CreateDona(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}

	var req createDonaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	nome := strings.TrimSpace(req.Nome)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if nome == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_nome")
		return
	}
	if email == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_email")
		return
	}

	if _, err := h.estabelecimentos.GetEstablishmentSuperAdminByID(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	userID, err := h.auth.CreateDonaForEstablishment(r.Context(), id, nome, email, senhaInicialDona)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailJaCadastrado):
			writeJSONError(w, http.StatusConflict, "email_already_exists")
		case errors.Is(err, service.ErrEstabelecimentoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "not_found")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, createDonaResponse{
		UserID:       userID,
		Email:        email,
		Nome:         nome,
		SenhaInicial: senhaInicialDona,
	})
}

// UploadLogo serve POST /api/v1/admin/establishments/{id}/logo
func (h *AdminEstablishmentsHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}

	if _, err := h.estabelecimentos.GetEstablishmentSuperAdminByID(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	if err := r.ParseMultipartForm(maxLogoUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_multipart")
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			writeJSONError(w, http.StatusBadRequest, "missing_logo")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_multipart")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !extensoesLogoPermitidas[ext] {
		writeJSONError(w, http.StatusBadRequest, "invalid_image_type")
		return
	}

	if header.Size > maxLogoUploadBytes {
		writeJSONError(w, http.StatusBadRequest, "file_too_large")
		return
	}

	limited := io.LimitReader(file, maxLogoUploadBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_multipart")
		return
	}
	if int64(len(data)) > maxLogoUploadBytes {
		writeJSONError(w, http.StatusBadRequest, "file_too_large")
		return
	}

	fileName := fmt.Sprintf("%s-%d%s", id, time.Now().UnixNano(), ext)
	publicURL, err := storagesvc.UploadLogoToSupabase(r.Context(), bytes.NewReader(data), fileName)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "storage_upload_failed")
		return
	}

	if err := h.estabelecimentos.UpdateLogoURL(r.Context(), id, publicURL); err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"logo_url": publicURL})
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

// ToggleWhatsApp serve PUT /api/v1/admin/establishments/{id}/toggle-whatsapp.
func (h *AdminEstablishmentsHandler) ToggleWhatsApp(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}

	var req toggleWhatsAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if req.WhatsAppEnabled == nil {
		writeJSONError(w, http.StatusBadRequest, "missing_whatsapp_enabled")
		return
	}

	view, err := h.estabelecimentos.SetWhatsAppEnabled(r.Context(), id, *req.WhatsAppEnabled)
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	h.whatsAppGate.InvalidateCache(id)
	actor := ""
	if claims, ok := security.ClaimsFromContext(r.Context()); ok {
		actor = claims.Email
	}
	log.Printf("audit whatsapp_toggle: actor=%s tenant=%s enabled=%v", actor, id, *req.WhatsAppEnabled)
	writeJSON(w, http.StatusOK, view)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func writeJSONErrorMessage(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"error": code, "message": message})
}
