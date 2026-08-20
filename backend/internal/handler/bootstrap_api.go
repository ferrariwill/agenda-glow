package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

// agendaAPI cobre as operações de agenda usadas por BootstrapAPIHandler.
// *service.AgendaService satisfaz a interface; testes podem injetar stubs.
type agendaAPI interface {
	CriarAgendamento(
		ctx context.Context,
		estabelecimentoID, clienteNome, clienteTelefone, profissionalID, servicoID string,
		adicionaisIDs []string,
		inicio time.Time,
		origem service.OrigemAgendamento,
		aceitaAdiantar bool,
	) (service.ResultadoAgendamento, error)
	ValidarAgendamentoDaProfissional(ctx context.Context, establishmentID, professionalID, agendamentoID string) error
	ConcluirAtendimentoProfissional(ctx context.Context, establishmentID, professionalID, agendamentoID string, financeiro *service.FinanceiroService) error
	GetDashboardProfissional(ctx context.Context, establishmentID, professionalID string, selectedDate time.Time) (*service.DashboardProfissional, error)
}

type BootstrapAPIHandler struct {
	bootstrap *service.BootstrapService
	estab     *service.EstabelecimentoService
	agenda    agendaAPI
	esp       *service.EspecialidadeService
	prof      *service.ProfissionalService
	proc      *service.ProcedimentoService
	fin       *service.FinanceiroService
	fila      *service.FilaEsperaService
	insumo    *service.InsumoService
	earlySlot *service.EarlySlotService
}

func NewBootstrapAPIHandler(
	bootstrap *service.BootstrapService,
	estab *service.EstabelecimentoService,
	agenda agendaAPI,
	esp *service.EspecialidadeService,
	prof *service.ProfissionalService,
	proc *service.ProcedimentoService,
	fin *service.FinanceiroService,
	fila *service.FilaEsperaService,
	insumo *service.InsumoService,
	earlySlot *service.EarlySlotService,
) *BootstrapAPIHandler {
	return &BootstrapAPIHandler{
		bootstrap: bootstrap,
		estab:     estab,
		agenda:    agenda,
		esp:       esp,
		prof:      prof,
		proc:      proc,
		fin:       fin,
		fila:      fila,
		insumo:    insumo,
		earlySlot: earlySlot,
	}
}

// TenantBootstrap GET /api/v1/bootstrap
func (h *BootstrapAPIHandler) TenantBootstrap(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	data, err := h.bootstrap.BuildTenantBootstrap(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// AdminBootstrap GET /api/v1/admin/bootstrap
func (h *BootstrapAPIHandler) AdminBootstrap(w http.ResponseWriter, r *http.Request) {
	data, err := h.bootstrap.BuildAdminBootstrap(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// PublicCatalog GET /api/v1/public/{slug}/catalog
func (h *BootstrapAPIHandler) PublicCatalog(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	est, err := h.estab.BuscarPorSlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, service.ErrEstabelecimentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	catalog, err := h.estab.BuscarCatalogoAutoatendimento(r.Context(), est.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, catalog)
}

type createAppointmentRequest struct {
	ClienteNome     string   `json:"cliente_nome"`
	ClienteTelefone string   `json:"cliente_telefone"`
	ProfissionalID  string   `json:"profissional_id"`
	ServicoID       string   `json:"servico_id"`
	AdicionalIDs    []string `json:"adicional_ids"`
	Data            string   `json:"data"`
	HoraInicio      string   `json:"hora_inicio"`
	AceitaAdiantar  bool     `json:"aceita_adiantar"`
}

// CreateAppointment POST /api/v1/appointments (dona/secretaria) ou /api/v1/public/{slug}/appointments
func (h *BootstrapAPIHandler) CreateAppointment(w http.ResponseWriter, r *http.Request) {
	var establishmentID string
	slug := r.PathValue("slug")
	if slug != "" {
		est, err := h.estab.BuscarPorSlug(r.Context(), slug)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		establishmentID = est.ID
	} else {
		id, ok := security.EstablishmentIDFromContext(r.Context())
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
			return
		}
		establishmentID = id
	}

	var req createAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	inicio, err := time.ParseInLocation("2006-01-02 15:04", req.Data+" "+req.HoraInicio, time.Local)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_datetime")
		return
	}

	origem := service.OrigemInterno
	if slug != "" {
		origem = service.OrigemExterno
	}

	result, err := h.agenda.CriarAgendamento(
		r.Context(),
		establishmentID,
		req.ClienteNome,
		req.ClienteTelefone,
		req.ProfissionalID,
		req.ServicoID,
		req.AdicionalIDs,
		inicio,
		origem,
		req.AceitaAdiantar,
	)
	if err != nil {
		if errors.Is(err, service.ErrColisaoHorario) {
			writeJSONError(w, http.StatusConflict, "slot_unavailable")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	if req.AceitaAdiantar {
		_, _ = h.fila.Inscrever(r.Context(), service.InscreverFilaInput{
			EstabelecimentoID: establishmentID,
			ProfissionalID:    req.ProfissionalID,
			ClienteNome:       req.ClienteNome,
			ClienteTelefone:   req.ClienteTelefone,
			AgendamentoID:     result.ID,
			Data:              req.Data,
			Hora:              req.HoraInicio,
		})
	}

	writeJSON(w, http.StatusCreated, result)
}

type createProfessionalAppointmentRequest struct {
	ClienteNome     string   `json:"cliente_nome"`
	ClienteTelefone string   `json:"cliente_telefone"`
	ProfissionalID  string   `json:"profissional_id"` // ignorado; token manda
	ServicoID       string   `json:"servico_id"`
	AdicionalIDs    []string `json:"adicional_ids"`
	Data            string   `json:"data"`
	HoraInicio      string   `json:"hora_inicio"`
	AceitaAdiantar  bool     `json:"aceita_adiantar"`
}

// CreateProfessionalAppointment POST /api/v1/professional/appointments
// Profissional agenda apenas para si (profissional_id do JWT). Não amplia RequireTenantStaff.
func (h *BootstrapAPIHandler) CreateProfessionalAppointment(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	profID, ok := security.ProfessionalIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_professional_context")
		return
	}

	var req createProfessionalAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	// Escopo: rejeitar tentativa explícita de agendar para outra profissional.
	if req.ProfissionalID != "" && req.ProfissionalID != profID {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return
	}

	inicio, err := time.ParseInLocation("2006-01-02 15:04", req.Data+" "+req.HoraInicio, time.Local)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_datetime")
		return
	}

	result, err := h.agenda.CriarAgendamento(
		r.Context(),
		establishmentID,
		req.ClienteNome,
		req.ClienteTelefone,
		profID,
		req.ServicoID,
		req.AdicionalIDs,
		inicio,
		service.OrigemInterno,
		req.AceitaAdiantar,
	)
	if err != nil {
		if errors.Is(err, service.ErrColisaoHorario) {
			writeJSONError(w, http.StatusConflict, "slot_unavailable")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}

	if req.AceitaAdiantar && h.fila != nil {
		_, _ = h.fila.Inscrever(r.Context(), service.InscreverFilaInput{
			EstabelecimentoID: establishmentID,
			ProfissionalID:    profID,
			ClienteNome:       req.ClienteNome,
			ClienteTelefone:   req.ClienteTelefone,
			AgendamentoID:     result.ID,
			Data:              req.Data,
			Hora:              req.HoraInicio,
		})
	}

	writeJSON(w, http.StatusCreated, result)
}

type updateProfessionalRequest struct {
	Nome            string  `json:"nome"`
	EspecialidadeID string  `json:"especialidade_id"`
	Comissao        float64 `json:"comissao_porcentagem"`
	Ativo           bool    `json:"ativo"`
	DataNascimento  *string `json:"data_nascimento"`
	FotoURL         *string `json:"foto_url"`
}

// UpdateProfessional PUT /api/v1/professionals/{id}
// data_nascimento / foto_url omitidos ou null → não alteram; string válida → define; data futura → 400 birthdate_in_future.
func (h *BootstrapAPIHandler) UpdateProfessional(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	profID := r.PathValue("id")
	var req updateProfessionalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := h.prof.UpdateProfessional(
		r.Context(), establishmentID, profID,
		req.Nome, req.EspecialidadeID, req.Comissao, req.Ativo,
		req.DataNascimento, req.FotoURL,
	); err != nil {
		if errors.Is(err, service.ErrPlanLimitExceeded) {
			writeJSON(w, http.StatusForbidden, limitReachedResponse{
				Error:   "limit_reached",
				Message: "Seu plano atingiu o limite de profissionais parceiras permitidas. Faça um upgrade no painel.",
			})
			return
		}
		if errors.Is(err, service.ErrProfissionalNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "professional_not_found")
			return
		}
		if errors.Is(err, service.ErrDataNascimentoFutura) {
			writeJSONError(w, http.StatusBadRequest, "birthdate_in_future")
			return
		}
		if errors.Is(err, service.ErrDataNascimentoInvalida) {
			writeJSONError(w, http.StatusBadRequest, "invalid_birthdate")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type specialtyRequest struct {
	Nome  string `json:"nome"`
	Ativo *bool  `json:"ativo,omitempty"`
}

// CreateSpecialty POST /api/v1/specialties
func (h *BootstrapAPIHandler) CreateSpecialty(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	var req specialtyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	id, err := h.esp.CreateEspecialidade(r.Context(), establishmentID, req.Nome)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

// UpdateSpecialty PUT /api/v1/specialties/{id}
func (h *BootstrapAPIHandler) UpdateSpecialty(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	id := r.PathValue("id")
	var req specialtyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	ativo := true
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	if err := h.esp.UpdateEspecialidade(r.Context(), establishmentID, id, req.Nome, ativo); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListSpecialties GET /api/v1/specialties
func (h *BootstrapAPIHandler) ListSpecialties(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	list, err := h.esp.ListEspecialidades(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type createClientRequest struct {
	Nome     string `json:"nome"`
	Telefone string `json:"telefone"`
	Email    string `json:"email,omitempty"`
}

// CreateClient POST /api/v1/clients
func (h *BootstrapAPIHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	var req createClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	// Reutiliza resolver de cliente via agendamento dummy não — inserir direto
	const insert = `
INSERT INTO clientes (estabelecimento_id, nome, telefone, email)
VALUES ($1, $2, $3, NULLIF($4, ''))
ON CONFLICT (estabelecimento_id, telefone) DO UPDATE SET nome = EXCLUDED.nome
RETURNING id
`
	var id string
	if err := h.bootstrap.DB().GetContext(r.Context(), &id, insert, establishmentID, req.Nome, req.Telefone, req.Email); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

// CompleteAppointment POST /api/v1/professional/appointments/{id}/complete
func (h *BootstrapAPIHandler) CompleteAppointment(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	profID, ok := security.ProfessionalIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_professional_context")
		return
	}
	agID := r.PathValue("id")
	if err := h.agenda.ValidarAgendamentoDaProfissional(r.Context(), establishmentID, profID, agID); err != nil {
		writeJSONError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := h.agenda.ConcluirAtendimentoProfissional(r.Context(), establishmentID, profID, agID, h.fin); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

// ProfessionalDashboard GET /api/v1/professional/dashboard?date=YYYY-MM-DD
func (h *BootstrapAPIHandler) ProfessionalDashboard(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	profID, ok := security.ProfessionalIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_professional_context")
		return
	}
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	dt, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_date")
		return
	}
	dash, err := h.agenda.GetDashboardProfissional(r.Context(), establishmentID, profID, dt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

type lancamentoRequest struct {
	Tipo      string  `json:"tipo"`
	Descricao string  `json:"descricao"`
	Valor     float64 `json:"valor"`
}

// CreateLancamento POST /api/v1/cash-flow
func (h *BootstrapAPIHandler) CreateLancamento(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	var req lancamentoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := h.fin.RegistrarLancamentoCaixa(r.Context(), establishmentID, req.Descricao, req.Tipo, req.Valor); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// DashboardGerencial GET /api/v1/dashboard/gerencial
func (h *BootstrapAPIHandler) DashboardGerencial(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	est, err := h.estab.BuscarPorID(r.Context(), establishmentID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	start, end, _ := service.PeriodoMesAtual()
	dash, err := h.fin.GetDashboardGerencial(r.Context(), establishmentID, est.NomeComercial, start, end)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, dash)
}

// ChargeAppointment POST /api/v1/appointments/{id}/charge
func (h *BootstrapAPIHandler) ChargeAppointment(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	agID := r.PathValue("id")

	var req struct {
		Metodo   string   `json:"metodo"`
		Valor    *float64 `json:"valor"`
		Concluir *bool    `json:"concluir"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	concluir := true
	if req.Concluir != nil {
		concluir = *req.Concluir
	}

	if err := h.fin.CobrarAtendimento(r.Context(), establishmentID, agID, service.CobrancaInput{
		Metodo:   req.Metodo,
		Valor:    req.Valor,
		Concluir: concluir,
	}); err != nil {
		switch {
		case errors.Is(err, service.ErrAgendamentoJaCobrado):
			writeJSONError(w, http.StatusConflict, "already_charged")
		case errors.Is(err, service.ErrAgendamentoJaConcluido):
			writeJSONError(w, http.StatusConflict, "already_completed")
		case errors.Is(err, service.ErrAgendamentoCancelado):
			writeJSONError(w, http.StatusBadRequest, "cancelled")
		case errors.Is(err, service.ErrAgendamentoNaoEncontrado):
			writeJSONError(w, http.StatusNotFound, "not_found")
		default:
			writeJSONError(w, http.StatusBadRequest, "invalid_status")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "charged"})
}

// CancelAppointment POST /api/v1/appointments/{id}/cancel
func (h *BootstrapAPIHandler) CancelAppointment(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	agID := r.PathValue("id")
	if err := h.earlySlot.CancelAppointment(r.Context(), establishmentID, agID); err != nil {
		if errors.Is(err, service.ErrAgendamentoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// MarkFilaNotificada POST /api/v1/waitlist/{id}/notify
func (h *BootstrapAPIHandler) MarkFilaNotificada(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.fila.MarcarNotificada(r.Context(), id); err != nil {
		if errors.Is(err, service.ErrFilaNaoEncontrada) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "notified"})
}

// ListInsumos GET /api/v1/supplies?q=&limit=20
func (h *BootstrapAPIHandler) ListInsumos(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	q := r.URL.Query().Get("q")
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.insumo.Search(r.Context(), establishmentID, q, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type insumoRequest struct {
	Nome                 string   `json:"nome"`
	Marca                string   `json:"marca"`
	Categoria            string   `json:"categoria"`
	Quantidade           *float64 `json:"quantidade"`
	QuantidadeEmbalagens *float64 `json:"quantidade_embalagens"`
	ConteudoPorEmbalagem *float64 `json:"conteudo_por_embalagem"`
	EstoqueMinimo        float64  `json:"estoque_minimo"`
	EstoqueIdeal         float64  `json:"estoque_ideal"`
	ValorUnitario        float64  `json:"valor_unitario"`
	Unidade              string   `json:"unidade"`
	ImagemURL            string   `json:"imagem_url"`
	InstrucoesUso        string   `json:"instrucoes_uso"`
	Ativo                *bool    `json:"ativo,omitempty"`
}

func (req insumoRequest) toInput(ativo bool) service.InsumoInput {
	in := service.InsumoInput{
		Nome:                 req.Nome,
		Marca:                req.Marca,
		Categoria:            req.Categoria,
		Unidade:              req.Unidade,
		Quantidade:           req.Quantidade,
		QuantidadeEmbalagens: req.QuantidadeEmbalagens,
		EstoqueMinimo:        req.EstoqueMinimo,
		EstoqueIdeal:         req.EstoqueIdeal,
		ValorUnitario:        req.ValorUnitario,
		ImagemURL:            req.ImagemURL,
		InstrucoesUso:        req.InstrucoesUso,
		Ativo:                ativo,
	}
	if req.ConteudoPorEmbalagem != nil {
		in.ConteudoPorEmbalagem = *req.ConteudoPorEmbalagem
	}
	return in
}

// CreateInsumo POST /api/v1/supplies
func (h *BootstrapAPIHandler) CreateInsumo(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	var req insumoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	id, err := h.insumo.Create(r.Context(), establishmentID, req.toInput(true))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusCreated, idResponse{ID: id})
}

// UpdateInsumo PUT /api/v1/supplies/{id}
func (h *BootstrapAPIHandler) UpdateInsumo(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	id := r.PathValue("id")
	var req insumoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	ativo := true
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	if err := h.insumo.Update(r.Context(), establishmentID, id, req.toInput(ativo)); err != nil {
		if errors.Is(err, service.ErrInsumoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type adjustEstoqueRequest struct {
	Delta           *float64 `json:"delta"`
	DeltaEmbalagens *float64 `json:"delta_embalagens"`
}

// AdjustInsumoEstoque POST /api/v1/supplies/{id}/adjust
func (h *BootstrapAPIHandler) AdjustInsumoEstoque(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	id := r.PathValue("id")
	var req adjustEstoqueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := h.insumo.AjustarEstoque(r.Context(), establishmentID, id, service.AjusteEstoqueInput{
		Delta:           req.Delta,
		DeltaEmbalagens: req.DeltaEmbalagens,
	}); err != nil {
		if errors.Is(err, service.ErrInsumoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeleteInsumo DELETE /api/v1/supplies/{id}
func (h *BootstrapAPIHandler) DeleteInsumo(w http.ResponseWriter, r *http.Request) {
	establishmentID, ok := security.EstablishmentIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing_establishment_context")
		return
	}
	id := r.PathValue("id")
	if err := h.insumo.Delete(r.Context(), establishmentID, id); err != nil {
		if errors.Is(err, service.ErrInsumoNaoEncontrado) {
			writeJSONError(w, http.StatusNotFound, "not_found")
			return
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_payload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
