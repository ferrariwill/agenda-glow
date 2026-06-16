package handler

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/agendaglow/agendaglow/frontend"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type DashboardProfissionalHandler struct {
	agenda     *service.AgendaService
	financeiro *service.FinanceiroService
	tmpl       *template.Template
}

func NewDashboardProfissionalHandler(
	agenda *service.AgendaService,
	financeiro *service.FinanceiroService,
) (*DashboardProfissionalHandler, error) {
	tmpl, err := frontend.LoadProfessionalDashboardTemplates()
	if err != nil {
		return nil, err
	}
	return &DashboardProfissionalHandler{
		agenda:     agenda,
		financeiro: financeiro,
		tmpl:       tmpl,
	}, nil
}

// ServeHTTP renderiza GET /dashboard/profissional
func (h *DashboardProfissionalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildDashboard(r)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "profissional_page", data); err != nil {
		http.Error(w, "Erro ao renderizar painel", http.StatusInternalServerError)
	}
}

// Timeline retorna partial HTMX da agenda do dia.
func (h *DashboardProfissionalHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildDashboard(r)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "timeline_section", data); err != nil {
		http.Error(w, "Erro ao renderizar agenda", http.StatusInternalServerError)
	}
}

// CompleteAppointment conclui atendimento confirmado via HTMX.
func (h *DashboardProfissionalHandler) CompleteAppointment(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	professionalID, ok := security.ProfessionalIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}

	agendamentoID := strings.TrimSpace(r.PathValue("id"))
	if agendamentoID == "" {
		http.Error(w, "Agendamento inválido", http.StatusBadRequest)
		return
	}

	if err := h.agenda.ConcluirAtendimentoProfissional(
		r.Context(),
		establishmentID,
		professionalID,
		agendamentoID,
		h.financeiro,
	); err != nil {
		switch {
		case errors.Is(err, service.ErrAgendamentoNaoPertenceProfissional):
			http.Error(w, "Agendamento não encontrado", http.StatusNotFound)
		case errors.Is(err, service.ErrAgendamentoStatusInvalido):
			http.Error(w, "Atendimento não está confirmado", http.StatusConflict)
		case errors.Is(err, service.ErrAgendamentoJaConcluido):
			http.Error(w, "Atendimento já concluído", http.StatusConflict)
		case errors.Is(err, service.ErrAgendamentoCancelado):
			http.Error(w, "Atendimento cancelado", http.StatusConflict)
		default:
			http.Error(w, "Erro ao concluir atendimento", http.StatusInternalServerError)
		}
		return
	}

	data, err := h.buildDashboard(r)
	if err != nil {
		http.Error(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "resumo_semana", data); err != nil {
		return
	}
	if err := h.tmpl.ExecuteTemplate(w, "agenda_card_delete", agendamentoID); err != nil {
		return
	}
}

func (h *DashboardProfissionalHandler) buildDashboard(r *http.Request) (*service.DashboardProfissional, error) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		return nil, errors.New("missing establishment")
	}
	professionalID, ok := security.ProfessionalIDFromContext(r.Context())
	if !ok {
		return nil, errors.New("missing professional")
	}

	selectedDate, err := service.ParseDataQuery(r.URL.Query().Get("data"))
	if err != nil {
		return nil, err
	}

	return h.agenda.GetDashboardProfissional(r.Context(), establishmentID, professionalID, selectedDate)
}
