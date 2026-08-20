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
	especialidades   *service.EspecialidadeService
	financeiro       *service.FinanceiroService
	estabelecimentos *service.EstabelecimentoService
	tmpl             *template.Template
}

func NewAdminConfigHandler(
	procedimentos *service.ProcedimentoService,
	profissionais *service.ProfissionalService,
	especialidades *service.EspecialidadeService,
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
		especialidades:   especialidades,
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
	Profissionais  []service.Profissional
	Especialidades []service.Especialidade
	Limite         service.StatusLimiteEquipe
}

type especialidadesPageData struct {
	adminShellData
	Especialidades []service.Especialidade
}

type equipeRowRenderData struct {
	Profissional   service.Profissional
	Especialidades []service.Especialidade
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

	id, err := h.procedimentos.CreateService(r.Context(), estID, r.FormValue("nome"), preco, duracao, nil, "")
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

	especialidades, err := h.especialidades.ListEspecialidades(r.Context(), shell.establishmentID())
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
		Especialidades: especialidades,
		Limite:         *limite,
	})
}

// Especialidades GET /admin/especialidades
func (h *AdminConfigHandler) Especialidades(w http.ResponseWriter, r *http.Request) {
	shell, err := h.shellData(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	shell.NavActive = "especialidades"

	lista, err := h.especialidades.ListEspecialidades(r.Context(), shell.establishmentID())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "config_especialidades_page", especialidadesPageData{
		adminShellData: shell,
		Especialidades: lista,
	})
}

// CreateEspecialidade POST /admin/especialidades
func (h *AdminConfigHandler) CreateEspecialidade(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	id, err := h.especialidades.CreateEspecialidade(r.Context(), estID, r.FormValue("nome"))
	if err != nil {
		if errors.Is(err, service.ErrEspecialidadeNomeDuplicado) {
			http.Error(w, "Já existe especialidade com este nome", http.StatusConflict)
			return
		}
		http.Error(w, "Erro ao cadastrar especialidade", http.StatusBadRequest)
		return
	}

	esp, err := h.especialidades.BuscarPorID(r.Context(), estID, id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "especialidade_row_new_oob", esp)
	_ = h.tmpl.ExecuteTemplate(w, "admin_flash_oob", "Especialidade cadastrada.")
}

// UpdateEspecialidade POST /admin/especialidades/{id}
func (h *AdminConfigHandler) UpdateEspecialidade(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	espID := strings.TrimSpace(r.PathValue("id"))
	if espID == "" {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}
	ativo := r.FormValue("ativo") == "on" || r.FormValue("ativo") == "true" || r.FormValue("ativo") == "1"

	if err := h.especialidades.UpdateEspecialidade(r.Context(), estID, espID, r.FormValue("nome"), ativo); err != nil {
		switch {
		case errors.Is(err, service.ErrEspecialidadeNaoEncontrada):
			http.Error(w, "Especialidade não encontrada", http.StatusNotFound)
		case errors.Is(err, service.ErrEspecialidadeNomeDuplicado):
			http.Error(w, "Já existe especialidade com este nome", http.StatusConflict)
		default:
			http.Error(w, "Erro ao atualizar especialidade", http.StatusBadRequest)
		}
		return
	}

	esp, err := h.especialidades.BuscarPorID(r.Context(), estID, espID)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "especialidade_row_oob", esp)
	_ = h.tmpl.ExecuteTemplate(w, "admin_flash_oob", "Especialidade atualizada.")
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
		strings.TrimSpace(r.FormValue("especialidade_id")),
		comissao,
		nil,
		nil,
	)
	if err != nil {
		if errors.Is(err, service.ErrPlanLimitExceeded) {
			http.Error(w, "Limite do plano atingido", http.StatusForbidden)
			return
		}
		if errors.Is(err, service.ErrEspecialidadeNaoEncontrada) {
			http.Error(w, "Selecione uma especialidade válida", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrEspecialidadeInativa) {
			http.Error(w, "Especialidade inativa", http.StatusBadRequest)
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
	especialidades, _ := h.especialidades.ListEspecialidades(r.Context(), estID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "equipe_row_new_oob", equipeRowRenderData{
		Profissional:   *prof,
		Especialidades: especialidades,
	})
	if limite != nil {
		_ = h.tmpl.ExecuteTemplate(w, "limite_alert_oob", *limite)
	}
	_ = h.tmpl.ExecuteTemplate(w, "equipe_empty_remove_oob", nil)
}

// UpdateProfissional POST /admin/equipe/{id}
func (h *AdminConfigHandler) UpdateProfissional(w http.ResponseWriter, r *http.Request) {
	estID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Não autorizado", http.StatusUnauthorized)
		return
	}
	profID := strings.TrimSpace(r.PathValue("id"))
	if profID == "" {
		http.Error(w, "Profissional inválida", http.StatusBadRequest)
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
	ativo := r.FormValue("ativo") == "on" || r.FormValue("ativo") == "true" || r.FormValue("ativo") == "1"

	err = h.profissionais.UpdateProfessional(
		r.Context(), estID, profID,
		r.FormValue("nome"),
		strings.TrimSpace(r.FormValue("especialidade_id")),
		comissao,
		ativo,
		nil,
		nil,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProfissionalNaoEncontrado):
			http.Error(w, "Profissional não encontrada", http.StatusNotFound)
		case errors.Is(err, service.ErrPlanLimitExceeded):
			http.Error(w, "Limite do plano atingido — não é possível reativar", http.StatusForbidden)
		case errors.Is(err, service.ErrEspecialidadeNaoEncontrada):
			http.Error(w, "Selecione uma especialidade válida", http.StatusBadRequest)
		case errors.Is(err, service.ErrEspecialidadeInativa):
			http.Error(w, "Especialidade inativa", http.StatusBadRequest)
		default:
			http.Error(w, "Erro ao atualizar profissional", http.StatusBadRequest)
		}
		return
	}

	prof, err := h.profissionais.BuscarProfissionalPorID(r.Context(), estID, profID)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	especialidades, _ := h.especialidades.ListEspecialidades(r.Context(), estID)
	limite, _ := h.profissionais.GetStatusLimiteEquipe(r.Context(), estID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "equipe_row_oob", equipeRowRenderData{
		Profissional:   *prof,
		Especialidades: especialidades,
	})
	if limite != nil {
		_ = h.tmpl.ExecuteTemplate(w, "limite_alert_oob", *limite)
	}
	_ = h.tmpl.ExecuteTemplate(w, "equipe_flash_oob", "Profissional atualizada com sucesso.")
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
