package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type AdminAvisosHandler struct {
	avisos *service.AvisoService
}

func NewAdminAvisosHandler(avisos *service.AvisoService) *AdminAvisosHandler {
	return &AdminAvisosHandler{avisos: avisos}
}

type createAvisoRequest struct {
	Titulo             string   `json:"titulo"`
	Corpo              *string  `json:"corpo"`
	Severidade         string   `json:"severidade"`
	AudienceTipo       string   `json:"audience_tipo"`
	EstabelecimentoIDs []string `json:"estabelecimento_ids"`
	ExpiresAt          *string  `json:"expires_at"`
}

type patchAvisoRequest struct {
	Titulo     *string          `json:"titulo"`
	Corpo      *string          `json:"corpo"`
	Severidade *string          `json:"severidade"`
	Ativo      *bool            `json:"ativo"`
	ExpiresAt  *json.RawMessage `json:"expires_at"`
}

type adminAvisosListResponse struct {
	Items []service.Aviso `json:"items"`
	Total int             `json:"total"`
}

// Create serve POST /api/v1/admin/avisos
func (h *AdminAvisosHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createAvisoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	expiresAt, err := parseOptionalFutureRFC3339(req.ExpiresAt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_expires_at")
		return
	}

	aviso, err := h.avisos.Create(r.Context(), service.CreateAvisoInput{
		Titulo:             req.Titulo,
		Corpo:              req.Corpo,
		Severidade:         req.Severidade,
		AudienceTipo:       req.AudienceTipo,
		EstabelecimentoIDs: req.EstabelecimentoIDs,
		ExpiresAt:          expiresAt,
		CreatedBy:          claims.UserID,
	})
	if err != nil {
		writeAvisoServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, aviso)
}

// List serve GET /api/v1/admin/avisos
func (h *AdminAvisosHandler) List(w http.ResponseWriter, r *http.Request) {
	var ativo *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("ativo")); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_ativo")
			return
		}
		ativo = &v
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	items, total, err := h.avisos.ListAdmin(r.Context(), ativo, limit, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, adminAvisosListResponse{Items: items, Total: total})
}

// Patch serve PATCH /api/v1/admin/avisos/{id}
func (h *AdminAvisosHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_id")
		return
	}

	var req patchAvisoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	in := service.UpdateAvisoInput{
		Titulo:     req.Titulo,
		Corpo:      req.Corpo,
		Severidade: req.Severidade,
		Ativo:      req.Ativo,
	}
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(string(*req.ExpiresAt))
		if raw == "null" {
			in.ClearExpiry = true
		} else {
			var s string
			if err := json.Unmarshal(*req.ExpiresAt, &s); err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_expires_at")
				return
			}
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "invalid_expires_at")
				return
			}
			utc := t.UTC()
			in.ExpiresAt = &utc
		}
	}

	aviso, err := h.avisos.Update(r.Context(), id, in)
	if err != nil {
		writeAvisoServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, aviso)
}

func parseOptionalFutureRFC3339(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}

func writeAvisoServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrAvisoTituloInvalido):
		writeJSONError(w, http.StatusBadRequest, "invalid_titulo")
	case errors.Is(err, service.ErrAvisoCorpoInvalido):
		writeJSONError(w, http.StatusBadRequest, "invalid_corpo")
	case errors.Is(err, service.ErrAvisoSeveridadeInvalida):
		writeJSONError(w, http.StatusBadRequest, "invalid_severidade")
	case errors.Is(err, service.ErrAvisoAudienceInvalida):
		writeJSONError(w, http.StatusBadRequest, "invalid_audience")
	case errors.Is(err, service.ErrAvisoEstabelecimentosFaltando):
		writeJSONError(w, http.StatusBadRequest, "missing_estabelecimento_ids")
	case errors.Is(err, service.ErrAvisoExpiresAtInvalido):
		writeJSONError(w, http.StatusBadRequest, "invalid_expires_at")
	case errors.Is(err, service.ErrAvisoNaoEncontrado):
		writeJSONError(w, http.StatusNotFound, "aviso_not_found")
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
	}
}
