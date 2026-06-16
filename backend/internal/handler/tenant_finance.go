package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type TenantFinanceHandler struct {
	financeiro *service.FinanceiroService
}

func NewTenantFinanceHandler(financeiro *service.FinanceiroService) *TenantFinanceHandler {
	return &TenantFinanceHandler{financeiro: financeiro}
}

type payCommissionsRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// GetReport serve GET /api/v1/finance/report
func (h *TenantFinanceHandler) GetReport(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	startDate, endDate, err := parseDateRange(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_date_range")
		return
	}

	relatorio, err := h.financeiro.GetEstablishmentFinancialReport(r.Context(), establishmentID, startDate, endDate)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, relatorio)
}

// GetProfessionalPending serve GET /api/v1/finance/professionals/{id}/pending
func (h *TenantFinanceHandler) GetProfessionalPending(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	professionalID := r.PathValue("id")
	if professionalID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_professional_id")
		return
	}

	startDate, endDate, err := parseOptionalDateRange(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_date_range")
		return
	}

	pending, err := h.financeiro.GetPartnerPendingCommissions(
		r.Context(),
		establishmentID,
		professionalID,
		startDate,
		endDate,
	)
	if err != nil {
		if errors.Is(err, service.ErrProfissionalFinanceiroNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "professional_not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}

	writeJSON(w, http.StatusOK, pending)
}

// PayProfessionalCommissions serve POST /api/v1/finance/professionals/{id}/pay
func (h *TenantFinanceHandler) PayProfessionalCommissions(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}

	professionalID := r.PathValue("id")
	if professionalID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing_professional_id")
		return
	}

	var req payCommissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	startDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.StartDate), time.Local)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_start_date")
		return
	}
	endDate, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(req.EndDate), time.Local)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_end_date")
		return
	}

	if err := h.financeiro.PayPartnerCommissions(
		r.Context(),
		establishmentID,
		professionalID,
		startDate,
		endDate,
	); err != nil {
		switch {
		case errors.Is(err, service.ErrProfissionalFinanceiroNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "professional_not_found")
		case errors.Is(err, service.ErrNenhumaComissaoPendente):
			writeJSONError(w, http.StatusBadRequest, "no_pending_commissions")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "paid",
		"message": "Comissões quitadas com sucesso.",
	})
}

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	startStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endStr := strings.TrimSpace(r.URL.Query().Get("end_date"))
	if startStr == "" || endStr == "" {
		return time.Time{}, time.Time{}, errors.New("missing dates")
	}

	startDate, err := time.ParseInLocation("2006-01-02", startStr, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endDate, err := time.ParseInLocation("2006-01-02", endStr, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return startDate, endDate, nil
}

func parseOptionalDateRange(r *http.Request) (time.Time, time.Time, error) {
	startStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endStr := strings.TrimSpace(r.URL.Query().Get("end_date"))

	if startStr == "" && endStr == "" {
		// Período amplo padrão: últimos 12 meses até hoje.
		endDate := time.Now()
		startDate := endDate.AddDate(-1, 0, 0)
		return startDate, endDate, nil
	}

	return parseDateRange(r)
}
