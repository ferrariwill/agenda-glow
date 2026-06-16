package handler

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/frontend"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type DashboardDonaHandler struct {
	financeiro       *service.FinanceiroService
	estabelecimentos *service.EstabelecimentoService
	tmpl             *template.Template
}

func NewDashboardDonaHandler(
	financeiro *service.FinanceiroService,
	estabelecimentos *service.EstabelecimentoService,
) (*DashboardDonaHandler, error) {
	tmpl, err := frontend.LoadDashboardTemplates()
	if err != nil {
		return nil, err
	}
	return &DashboardDonaHandler{
		financeiro:       financeiro,
		estabelecimentos: estabelecimentos,
		tmpl:             tmpl,
	}, nil
}

// ServeHTTP renderiza GET /dashboard/gerencial
func (h *DashboardDonaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	data, err := h.buildDashboard(r.Context(), establishmentID, r)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "dashboard_page", data); err != nil {
		http.Error(w, "Erro ao renderizar painel", http.StatusInternalServerError)
	}
}

// PayProfessional quita comissões via HTMX e retorna fragmentos atualizados.
func (h *DashboardDonaHandler) PayProfessional(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	professionalID := strings.TrimSpace(r.PathValue("id"))
	if professionalID == "" {
		http.Error(w, "Profissional inválida", http.StatusBadRequest)
		return
	}

	startDate, endDate, err := parseDashboardPeriod(r)
	if err != nil {
		http.Error(w, "Período inválido", http.StatusBadRequest)
		return
	}

	if err := h.financeiro.PayPartnerCommissions(r.Context(), establishmentID, professionalID, startDate, endDate); err != nil {
		switch {
		case errors.Is(err, service.ErrProfissionalFinanceiroNaoEncontrado):
			http.Error(w, "Profissional não encontrada", http.StatusNotFound)
		case errors.Is(err, service.ErrNenhumaComissaoPendente):
			http.Error(w, "Nenhuma comissão pendente", http.StatusBadRequest)
		default:
			http.Error(w, "Erro ao quitar comissões", http.StatusInternalServerError)
		}
		return
	}

	data, err := h.buildDashboard(r.Context(), establishmentID, r)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	h.renderHTMXRefresh(w, data, professionalID)
}

// RegisterExpense lança despesa/receita rápida via HTMX.
func (h *DashboardDonaHandler) RegisterExpense(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	descricao := strings.TrimSpace(r.FormValue("descricao"))
	tipo := strings.TrimSpace(r.FormValue("tipo"))
	valorStr := strings.ReplaceAll(strings.TrimSpace(r.FormValue("valor")), ",", ".")
	valor, err := strconv.ParseFloat(valorStr, 64)
	if err != nil {
		http.Error(w, "Valor inválido", http.StatusBadRequest)
		return
	}

	if err := h.financeiro.RegistrarLancamentoCaixa(r.Context(), establishmentID, descricao, tipo, valor); err != nil {
		switch {
		case errors.Is(err, service.ErrDescricaoLancamentoObrigatoria),
			errors.Is(err, service.ErrValorLancamentoInvalido),
			errors.Is(err, service.ErrTipoLancamentoInvalido):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Erro ao registrar lançamento", http.StatusInternalServerError)
		}
		return
	}

	data, err := h.buildDashboard(r.Context(), establishmentID, r)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	h.renderHTMXRefresh(w, data, "")
}

func (h *DashboardDonaHandler) buildDashboard(ctx context.Context, establishmentID string, r *http.Request) (*service.DashboardGerencial, error) {
	startDate, endDate, err := parseDashboardPeriod(r)
	if err != nil {
		return nil, err
	}

	nomeEstabelecimento := "Meu Salão"
	if est, err := h.estabelecimentos.BuscarPorID(ctx, establishmentID); err == nil {
		nomeEstabelecimento = est.NomeComercial
	}

	return h.financeiro.GetDashboardGerencial(ctx, establishmentID, nomeEstabelecimento, startDate, endDate)
}

func (h *DashboardDonaHandler) renderHTMXRefresh(w http.ResponseWriter, data *service.DashboardGerencial, professionalID string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := h.tmpl.ExecuteTemplate(w, "metricas_cards", data); err != nil {
		http.Error(w, "Erro ao renderizar métricas", http.StatusInternalServerError)
		return
	}

	if professionalID != "" {
		for _, prof := range data.Equipe {
			if prof.ID == professionalID {
				if err := h.tmpl.ExecuteTemplate(w, "equipe_row_oob", profRowData{
					StartDate:    data.StartDate,
					EndDate:      data.EndDate,
					Profissional: prof,
				}); err != nil {
					return
				}
				break
			}
		}
	}

	if err := h.tmpl.ExecuteTemplate(w, "flash_success_oob", nil); err != nil {
		return
	}
}

type profRowData struct {
	StartDate    string
	EndDate      string
	Profissional service.DesempenhoProfissional
}

func parseDashboardPeriod(r *http.Request) (time.Time, time.Time, error) {
	startStr := strings.TrimSpace(r.URL.Query().Get("start_date"))
	endStr := strings.TrimSpace(r.URL.Query().Get("end_date"))

	if startStr == "" {
		startStr = strings.TrimSpace(r.FormValue("start_date"))
	}
	if endStr == "" {
		endStr = strings.TrimSpace(r.FormValue("end_date"))
	}

	if startStr != "" && endStr != "" {
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

	start, end, _ := service.PeriodoMesAtual()
	return start, end, nil
}
