package handler

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/agendaglow/agendaglow/frontend"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type SuperAdminUIHandler struct {
	estabelecimentos *service.EstabelecimentoService
	planos           *service.PlanoSaasService
	auth             *service.AuthService
	saasGuard        *security.SaaSGuard
	tmpl             *template.Template
	publicBaseURL    string
}

func NewSuperAdminUIHandler(
	estabelecimentos *service.EstabelecimentoService,
	planos *service.PlanoSaasService,
	auth *service.AuthService,
	saasGuard *security.SaaSGuard,
) (*SuperAdminUIHandler, error) {
	tmpl, err := frontend.LoadSuperAdminTemplates()
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSpace(os.Getenv("APP_PUBLIC_URL"))
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("APP_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = "https://agendaglow.com.br"
	}
	return &SuperAdminUIHandler{
		estabelecimentos: estabelecimentos,
		planos:           planos,
		auth:             auth,
		saasGuard:        saasGuard,
		tmpl:             tmpl,
		publicBaseURL:    strings.TrimRight(baseURL, "/"),
	}, nil
}

type establishmentRowData struct {
	Est           service.EstablishmentSuperAdminView
	PublicBaseURL string
	Plans         []service.PlanoSaas
}

type superAdminDashboardData struct {
	PublicBaseURL string
	NavActive     string
	Rows          []establishmentRowData
	Plans         []service.PlanoSaas
	FlashMessage  string
}

type superAdminPlanosData struct {
	NavActive     string
	Plans         []service.PlanoSaas
}

// Dashboard GET /superadmin/dashboard
func (h *SuperAdminUIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	estabelecimentos, err := h.estabelecimentos.ListEstablishmentsSuperAdmin(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	planos, err := h.planos.ListSaasPlans(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	rows := make([]establishmentRowData, len(estabelecimentos))
	for i, est := range estabelecimentos {
		rows[i] = establishmentRowData{
			Est:           est,
			PublicBaseURL: h.publicBaseURL,
			Plans:         planos,
		}
	}

	h.render(w, "superadmin_dashboard_page", superAdminDashboardData{
		PublicBaseURL: h.publicBaseURL,
		NavActive:     "dashboard",
		Rows:          rows,
		Plans:         planos,
	})
}

// CreateEstablishment POST /superadmin/establishments
func (h *SuperAdminUIHandler) CreateEstablishment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	nome := r.FormValue("nome_comercial")
	slug := r.FormValue("slug")
	planoID := strings.TrimSpace(r.FormValue("plano_id"))

	id, _, err := h.estabelecimentos.RegisterEstablishment(r.Context(), nome, slug)
	if err != nil {
		if errors.Is(err, service.ErrSlugAlreadyExists) {
			http.Error(w, "Slug já em uso", http.StatusConflict)
			return
		}
		http.Error(w, "Erro ao cadastrar salão", http.StatusBadRequest)
		return
	}

	if planoID != "" {
		if err := h.planos.AssignPlanToEstablishment(r.Context(), id, planoID, 12); err != nil {
			http.Error(w, "Salão criado, mas falha ao atribuir plano", http.StatusBadRequest)
			return
		}
		h.invalidateCache(id)
	}

	est, err := h.estabelecimentos.GetEstablishmentSuperAdminByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "establishment_row_new_oob", establishmentRowData{
		Est:           *est,
		PublicBaseURL: h.publicBaseURL,
		Plans:         h.loadPlans(r.Context()),
	})
}

// AssignPlanEstablishment POST /superadmin/establishments/{id}/assign-plan
func (h *SuperAdminUIHandler) AssignPlanEstablishment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	planoID := strings.TrimSpace(r.FormValue("plano_id"))
	if planoID == "" {
		http.Error(w, "Selecione um plano", http.StatusBadRequest)
		return
	}

	meses := 12
	if v := strings.TrimSpace(r.FormValue("meses")); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			meses = parsed
		}
	}

	if err := h.planos.AssignPlanToEstablishment(r.Context(), id, planoID, meses); err != nil {
		if errors.Is(err, service.ErrPlanoSaasNaoEncontrado) {
			http.Error(w, "Plano não encontrado", http.StatusBadRequest)
			return
		}
		http.Error(w, "Erro ao atribuir plano", http.StatusInternalServerError)
		return
	}

	h.invalidateCache(id)
	h.renderEstablishmentRowWithFlash(w, r, id, "Plano atribuído com sucesso.")
}

// CreateDonaOwner POST /superadmin/establishments/{id}/create-dona
func (h *SuperAdminUIHandler) CreateDonaOwner(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	est, err := h.estabelecimentos.GetEstablishmentSuperAdminByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			http.Error(w, "Salão não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if email == "" {
		email = est.Slug + "-dona@glow.local"
	}

	const senhaInicial = "AgendaGlow@2026"
	estID := id
	if _, err := h.auth.CreateUser(r.Context(), email, senhaInicial, security.RoleDona, &estID, nil, ""); err != nil {
		if errors.Is(err, service.ErrEmailJaCadastrado) {
			http.Error(w, "E-mail já cadastrado", http.StatusConflict)
			return
		}
		http.Error(w, "Erro ao criar usuária dona", http.StatusInternalServerError)
		return
	}

	msg := "Dona criada: " + email + " · senha: " + senhaInicial + " · login em /login/dona"
	h.renderEstablishmentRowWithFlash(w, r, id, msg)
}

// SuspendEstablishment POST /superadmin/establishments/{id}/suspend
func (h *SuperAdminUIHandler) SuspendEstablishment(w http.ResponseWriter, r *http.Request) {
	h.toggleEstablishment(w, r, false)
}

// ActivateEstablishment POST /superadmin/establishments/{id}/activate
func (h *SuperAdminUIHandler) ActivateEstablishment(w http.ResponseWriter, r *http.Request) {
	h.toggleEstablishment(w, r, true)
}

func (h *SuperAdminUIHandler) toggleEstablishment(w http.ResponseWriter, r *http.Request, activate bool) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	var err error
	var msg string
	if activate {
		err = h.planos.ActivateEstablishment(r.Context(), h.estabelecimentos, id)
		msg = "Salão ativado."
	} else {
		err = h.planos.SuspendEstablishment(r.Context(), h.estabelecimentos, id)
		msg = "Salão suspenso."
	}
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			http.Error(w, "Salão não encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "Erro ao atualizar status", http.StatusInternalServerError)
		return
	}

	h.invalidateCache(id)
	h.renderEstablishmentRowWithFlash(w, r, id, msg)
}

// RenewEstablishment POST /superadmin/establishments/{id}/renew
func (h *SuperAdminUIHandler) RenewEstablishment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := h.planos.RenewEstablishmentSubscription(r.Context(), id, 12); err != nil {
		if errors.Is(err, service.ErrPlanoSaasNaoEncontrado) {
			http.Error(w, "Salão sem plano atribuído", http.StatusBadRequest)
			return
		}
		http.Error(w, "Erro ao renovar assinatura", http.StatusInternalServerError)
		return
	}

	h.invalidateCache(id)
	h.renderEstablishmentRowWithFlash(w, r, id, "Assinatura renovada por +12 meses.")
}

// Planos GET /superadmin/planos
func (h *SuperAdminUIHandler) Planos(w http.ResponseWriter, r *http.Request) {
	planos, err := h.planos.ListSaasPlans(r.Context())
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	h.render(w, "superadmin_planos_page", superAdminPlanosData{
		NavActive: "planos",
		Plans:     planos,
	})
}

// CreatePlan POST /superadmin/planos
func (h *SuperAdminUIHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	preco, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("preco_mensal")), ",", "."), 64)
	if err != nil {
		http.Error(w, "Preço inválido", http.StatusBadRequest)
		return
	}
	limite, err := strconv.Atoi(strings.TrimSpace(r.FormValue("limite_profissionais")))
	if err != nil {
		http.Error(w, "Limite inválido", http.StatusBadRequest)
		return
	}

	id, err := h.planos.CreateSaasPlan(r.Context(), r.FormValue("nome"), preco, limite)
	if err != nil {
		if errors.Is(err, service.ErrPlanoSaasNomeDuplicado) {
			http.Error(w, "Já existe um plano com este nome", http.StatusConflict)
			return
		}
		http.Error(w, "Erro ao cadastrar plano", http.StatusBadRequest)
		return
	}

	plano, err := h.planos.BuscarPlanoSaasPorID(r.Context(), id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "plano_card_new_oob", plano)
	_ = h.tmpl.ExecuteTemplate(w, "superadmin_flash_oob", "Plano criado com sucesso.")
}

// UpdatePlan POST /superadmin/planos/{id}
func (h *SuperAdminUIHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	planID := strings.TrimSpace(r.PathValue("id"))
	if planID == "" {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	preco, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(r.FormValue("preco_mensal")), ",", "."), 64)
	if err != nil {
		http.Error(w, "Preço inválido", http.StatusBadRequest)
		return
	}
	limite, err := strconv.Atoi(strings.TrimSpace(r.FormValue("limite_profissionais")))
	if err != nil {
		http.Error(w, "Limite inválido", http.StatusBadRequest)
		return
	}
	ativo := r.FormValue("ativo") == "on" || r.FormValue("ativo") == "true" || r.FormValue("ativo") == "1"

	if err := h.planos.UpdateSaasPlan(r.Context(), planID, r.FormValue("nome"), preco, limite, ativo); err != nil {
		switch {
		case errors.Is(err, service.ErrPlanoSaasNaoEncontrado):
			http.Error(w, "Plano não encontrado", http.StatusNotFound)
		case errors.Is(err, service.ErrPlanoSaasNomeDuplicado):
			http.Error(w, "Já existe um plano com este nome", http.StatusConflict)
		case errors.Is(err, service.ErrLimiteProfissionaisInvalido):
			http.Error(w, "Limite de profissionais inválido", http.StatusBadRequest)
		default:
			http.Error(w, "Erro ao atualizar plano", http.StatusInternalServerError)
		}
		return
	}

	plano, err := h.planos.BuscarPlanoSaasPorID(r.Context(), planID)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if plano.Ativo {
		_ = h.tmpl.ExecuteTemplate(w, "plano_card_oob", plano)
	} else {
		_ = h.tmpl.ExecuteTemplate(w, "plano_card_remove_oob", planID)
	}
	_ = h.tmpl.ExecuteTemplate(w, "superadmin_flash_oob", "Plano atualizado com sucesso.")
}

func (h *SuperAdminUIHandler) renderEstablishmentRowWithFlash(w http.ResponseWriter, r *http.Request, id, flash string) {
	est, err := h.estabelecimentos.GetEstablishmentSuperAdminByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	row := establishmentRowData{
		Est:           *est,
		PublicBaseURL: h.publicBaseURL,
		Plans:         h.loadPlans(r.Context()),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "establishment_row_oob", row)
	if flash != "" {
		_ = h.tmpl.ExecuteTemplate(w, "superadmin_flash_oob", flash)
	}
}

func (h *SuperAdminUIHandler) loadPlans(ctx context.Context) []service.PlanoSaas {
	planos, err := h.planos.ListSaasPlans(ctx)
	if err != nil {
		return []service.PlanoSaas{}
	}
	return planos
}

func (h *SuperAdminUIHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Erro ao renderizar", http.StatusInternalServerError)
	}
}

func (h *SuperAdminUIHandler) invalidateCache(establishmentID string) {
	if h.saasGuard != nil {
		h.saasGuard.InvalidateCache(establishmentID)
	}
}
