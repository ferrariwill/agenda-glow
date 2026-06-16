package handler

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/agendaglow/agendaglow/frontend"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type AdminConfigHandler struct {
	procedimentos    *service.ProcedimentoService
	profissionais    *service.ProfissionalService
	financeiro       *service.FinanceiroService
	estabelecimentos *service.EstabelecimentoService
	tmpl             *template.Template
}

func NewAdminConfigHandler(
	procedimentos *service.ProcedimentoService,
	profissionais *service.ProfissionalService,
	financeiro *service.FinanceiroService,
	estabelecimentos *service.EstabelecimentoService,
) (*AdminConfigHandler, error) {
	tmpl, err := frontend.LoadAdminConfigTemplates()
	if err != nil {
		return nil, err
	}
	return &AdminConfigHandler{
		procedimentos:    procedimentos,
		profissionais:    profissionais,
		financeiro:       financeiro,
		estabelecimentos: estabelecimentos,
		tmpl:             tmpl,
	}, nil
}

type adminShellData struct {
	EstabelecimentoNome string
	NavActive           string
	estID               string
}

func (s adminShellData) establishmentID() string { return s.estID }

type servicosPageData struct {
	adminShellData
	Servicos []service.Servico
}

type equipePageData struct {
	adminShellData
	Profissionais []service.Profissional
	Limite        service.StatusLimiteEquipe
}

type caixaPageData struct {
	adminShellData
	Caixa *service.CaixaFluxoPage
}

// Servicos GET /admin/servicos
func (h *AdminConfigHandler) Servicos(w http.ResponseWriter, r *http.Request) {
	shell, err := h.shellData(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	shell.NavActive = "servicos"

	servicos, err := h.procedimentos.ListServices(r.Context(), shell.establishmentID())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "config_servicos_page", servicosPageData{
		adminShellData: shell,
		Servicos:       servicos,
	})
}

// CreateServico POST /admin/servicos
func (h *AdminConfigHandler) CreateServico(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	preco, duracao, err := parsePrecoDuracao(r.FormValue("preco_base"), r.FormValue("duracao_base_minutos"))
	if err != nil {
		http.Error(w, "Preço ou duração inválidos", http.StatusBadRequest)
		return
	}

	id, err := h.procedimentos.CreateService(r.Context(), estID, r.FormValue("nome"), preco, duracao)
	if err != nil {
		http.Error(w, "Erro ao cadastrar serviço", http.StatusBadRequest)
		return
	}

	serv, err := h.procedimentos.BuscarServicoPorID(r.Context(), estID, id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "servico_card", serv)
}

// CreateAdicional POST /admin/servicos/{id}/adicionais
func (h *AdminConfigHandler) CreateAdicional(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	serviceID := strings.TrimSpace(r.PathValue("id"))
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	preco, duracao, err := parsePrecoDuracao(r.FormValue("preco_adicional"), r.FormValue("duracao_adicional_minutos"))
	if err != nil {
		http.Error(w, "Preço ou duração inválidos", http.StatusBadRequest)
		return
	}

	id, err := h.procedimentos.CreateServiceAdditional(
		r.Context(), estID, serviceID,
		r.FormValue("nome"), preco, duracao,
	)
	if err != nil {
		http.Error(w, "Erro ao cadastrar adicional", http.StatusBadRequest)
		return
	}

	ad, err := h.procedimentos.BuscarAdicionalPorID(r.Context(), estID, id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "adicional_item", ad)
}

// Equipe GET /admin/equipe
func (h *AdminConfigHandler) Equipe(w http.ResponseWriter, r *http.Request) {
	shell, err := h.shellData(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	shell.NavActive = "equipe"

	lista, err := h.profissionais.ListProfessionals(r.Context(), shell.establishmentID())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	limite, err := h.profissionais.GetStatusLimiteEquipe(r.Context(), shell.establishmentID())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "config_equipe_page", equipePageData{
		adminShellData: shell,
		Profissionais:  lista,
		Limite:         *limite,
	})
}

// CreateProfissional POST /admin/equipe
func (h *AdminConfigHandler) CreateProfissional(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	comissao, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("comissao_porcentagem")), ",", "."), 64)
	if err != nil {
		http.Error(w, "Comissão inválida", http.StatusBadRequest)
		return
	}

	id, err := h.profissionais.CreateProfessional(
		r.Context(), estID,
		r.FormValue("nome"),
		r.FormValue("especialidade"),
		comissao,
	)
	if err != nil {
		if errors.Is(err, service.ErrPlanLimitExceeded) {
			http.Error(w, "Limite do plano atingido", http.StatusForbidden)
			return
		}
		http.Error(w, "Erro ao cadastrar profissional", http.StatusBadRequest)
		return
	}

	prof, err := h.profissionais.BuscarProfissionalPorID(r.Context(), estID, id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	limite, _ := h.profissionais.GetStatusLimiteEquipe(r.Context(), estID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "equipe_row", prof)
	if limite != nil {
		_ = h.tmpl.ExecuteTemplate(w, "limite_alert_oob", *limite)
	}
}

// Caixa GET /admin/caixa
func (h *AdminConfigHandler) Caixa(w http.ResponseWriter, r *http.Request) {
	shell, err := h.shellData(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	shell.NavActive = "caixa"

	caixa, err := h.financeiro.GetCaixaFluxoPage(r.Context(), shell.establishmentID())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "caixa_fluxo_page", caixaPageData{
		adminShellData: shell,
		Caixa:          caixa,
	})
}

// CreateLancamento POST /admin/caixa/lancamento
func (h *AdminConfigHandler) CreateLancamento(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	valorStr := strings.ReplaceAll(strings.TrimSpace(r.FormValue("valor")), ",", ".")
	valor, err := strconv.ParseFloat(valorStr, 64)
	if err != nil {
		http.Error(w, "Valor inválido", http.StatusBadRequest)
		return
	}

	if err := h.financeiro.RegistrarLancamentoCaixa(r.Context(), estID, r.FormValue("descricao"), r.FormValue("tipo"), valor); err != nil {
		http.Error(w, "Erro ao registrar lançamento", http.StatusBadRequest)
		return
	}

	caixa, err := h.financeiro.GetCaixaFluxoPage(r.Context(), estID)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(caixa.Lancamentos) > 0 {
		_ = h.tmpl.ExecuteTemplate(w, "lancamento_row_oob", caixa.Lancamentos[0])
	}
	_ = h.tmpl.ExecuteTemplate(w, "caixa_resumo_oob", caixa)
}

func (h *AdminConfigHandler) shellData(ctx context.Context) (adminShellData, error) {
	estID, ok := security.EstablishmentIDFromContext(ctx)
	if !ok {
		return adminShellData{}, errors.New("missing establishment")
	}
	nome := "Meu Salão"
	if est, err := h.estabelecimentos.BuscarPorID(ctx, estID); err == nil {
		nome = est.NomeComercial
	}
	return adminShellData{EstabelecimentoNome: nome, estID: estID}, nil
}

func (h *AdminConfigHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Erro ao renderizar", http.StatusInternalServerError)
	}
}

func parsePrecoDuracao(precoStr, duracaoStr string) (float64, int, error) {
	preco, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(precoStr), ",", "."), 64)
	if err != nil {
		return 0, 0, err
	}
	duracao, err := strconv.Atoi(strings.TrimSpace(duracaoStr))
	if err != nil {
		return 0, 0, err
	}
	return preco, duracao, nil
}
