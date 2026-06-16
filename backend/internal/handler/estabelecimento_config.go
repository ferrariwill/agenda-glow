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
	"github.com/agendaglow/agendaglow/internal/service"
)

const maxLogoUploadBytes = 2 << 20 // 2MB

var extensoesLogoPermitidas = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
}

type EstabelecimentoConfigHandler struct {
	estabelecimentos *service.EstabelecimentoService
}

func NewEstabelecimentoConfigHandler(estabelecimentos *service.EstabelecimentoService) *EstabelecimentoConfigHandler {
	return &EstabelecimentoConfigHandler{estabelecimentos: estabelecimentos}
}

type configResponse struct {
	ID            string  `json:"id"`
	NomeComercial string  `json:"nome_comercial"`
	Slug          string  `json:"slug"`
	LogoURL       *string `json:"logo_url,omitempty"`
}

// ServeHTTP atende POST /v1/estabelecimentos/config (multipart/form-data).
func (h *EstabelecimentoConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxLogoUploadBytes); err != nil {
		http.Error(w, "Arquivo excede o limite de 2MB ou formulário inválido", http.StatusBadRequest)
		return
	}

	estabelecimentoID := strings.TrimSpace(r.FormValue("estabelecimento_id"))
	nomeComercial := strings.TrimSpace(r.FormValue("nome_comercial"))
	slug := strings.TrimSpace(r.FormValue("slug"))

	if estabelecimentoID == "" || nomeComercial == "" || slug == "" {
		http.Error(w, "estabelecimento_id, nome_comercial e slug são obrigatórios", http.StatusBadRequest)
		return
	}

	var logoURL *string
	file, header, err := r.FormFile("logo")
	if err == nil {
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if !extensoesLogoPermitidas[ext] {
			http.Error(w, "Formato de imagem inválido. Use .png, .jpg, .jpeg ou .webp", http.StatusBadRequest)
			return
		}

		if header.Size > maxLogoUploadBytes {
			http.Error(w, "Logo excede o limite de 2MB", http.StatusBadRequest)
			return
		}

		limited := io.LimitReader(file, maxLogoUploadBytes+1)
		data, err := io.ReadAll(limited)
		if err != nil {
			http.Error(w, "Erro ao ler arquivo de logo", http.StatusBadRequest)
			return
		}
		if int64(len(data)) > maxLogoUploadBytes {
			http.Error(w, "Logo excede o limite de 2MB", http.StatusBadRequest)
			return
		}

		fileName := fmt.Sprintf("%s-%d%s", estabelecimentoID, time.Now().UnixNano(), ext)
		publicURL, err := storagesvc.UploadLogoToSupabase(r.Context(), bytes.NewReader(data), fileName)
		if err != nil {
			http.Error(w, "Falha ao enviar logo para o storage", http.StatusBadGateway)
			return
		}
		logoURL = &publicURL
	} else if !errors.Is(err, http.ErrMissingFile) {
		http.Error(w, "Erro ao processar upload do logo", http.StatusBadRequest)
		return
	}

	atualizado, err := h.estabelecimentos.AtualizarConfig(r.Context(), service.ConfigEstabelecimentoInput{
		EstabelecimentoID: estabelecimentoID,
		NomeComercial:     nomeComercial,
		Slug:              slug,
		LogoURL:           logoURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSlugEmUso):
			http.Error(w, "Slug já está em uso por outro estabelecimento", http.StatusConflict)
		case errors.Is(err, service.ErrSlugInvalido):
			http.Error(w, "Slug inválido", http.StatusBadRequest)
		case errors.Is(err, service.ErrEstabelecimentoNaoEncontrado):
			http.NotFound(w, r)
		default:
			http.Error(w, "Erro ao atualizar configuração", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(configResponse{
		ID:            atualizado.ID,
		NomeComercial: atualizado.NomeComercial,
		Slug:          atualizado.Slug,
		LogoURL:       atualizado.LogoURL,
	})
}
