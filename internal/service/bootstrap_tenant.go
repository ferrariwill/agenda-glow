package service

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// TenantBootstrapPayload agrega dados do salão para o front-end React.
type TenantBootstrapPayload struct {
	Tenant         TenantBootstrapView     `json:"tenant"`
	Plano          *PlanoSaas              `json:"plano,omitempty"`
	Especialidades []Especialidade         `json:"especialidades"`
	Profissionais  []ProfissionalBootstrap `json:"profissionais"`
	Servicos       []Servico               `json:"servicos"`
	Clientes       []ClienteBootstrap      `json:"clientes"`
	Agendamentos   []AgendamentoBootstrap  `json:"agendamentos"`
	Lancamentos    []LancamentoBootstrap   `json:"lancamentos"`
	FilaEspera     []FilaEsperaEntry       `json:"fila_espera"`
	Insumos        []Insumo                `json:"insumos"`
}

type TenantBootstrapView struct {
	ID             string  `json:"id"`
	Nome           string  `json:"nome"`
	Slug           string  `json:"slug"`
	Status         string  `json:"status"`
	LogoURL        *string `json:"logo_url,omitempty"`
	PlanoID        *string `json:"plano_id,omitempty"`
	DataVencimento *string `json:"data_vencimento,omitempty"`
}

type ProfissionalBootstrap struct {
	Profissional
	Expedientes []ExpedienteProfissional `json:"expedientes"`
}

type ClienteBootstrap struct {
	ID       string  `db:"id" json:"id"`
	Nome     string  `db:"nome" json:"nome"`
	Telefone string  `db:"telefone" json:"telefone"`
	Email    *string `db:"email" json:"email,omitempty"`
	Ativo    bool    `db:"ativo" json:"ativo"`
	CriadoEm string  `db:"criado_em" json:"criado_em"`
}

type AgendamentoBootstrap struct {
	ID                      string   `json:"id"`
	TenantID                string   `json:"tenant_id"`
	ProfissionalID          string   `json:"profissional_id"`
	ServicoID               string   `json:"servico_id"`
	ServicoIDs              []string `json:"servico_ids"`
	AdicionalIDs            []string `json:"adicional_ids"`
	ClienteNome             string   `json:"cliente_nome"`
	ClienteTelefone         string   `json:"cliente_telefone"`
	Data                    string   `json:"data"`
	HoraInicio              string   `json:"hora_inicio"`
	Status                  string   `json:"status"`
	MinutosInvadidos        int      `json:"minutos_invadidos,omitempty"`
	AceitaAdiantar          bool     `json:"aceita_adiantar,omitempty"`
	ValorCobrado            *float64 `json:"valor_cobrado,omitempty"`
	MetodoPagamento         *string  `json:"metodo_pagamento,omitempty"`
	CobradoEm               *string  `json:"cobrado_em,omitempty"`
	ConfirmacaoCliente      string   `json:"confirmacao_cliente"`
	UltimoLembreteEnviadoEm *string  `json:"ultimo_lembrete_enviado_em,omitempty"`
}

type LancamentoBootstrap struct {
	ID             string  `json:"id"`
	TenantID       string  `json:"tenant_id"`
	Tipo           string  `json:"tipo"`
	Valor          float64 `json:"valor"`
	Descricao      string  `json:"descricao"`
	Data           string  `json:"data"`
	ProfissionalID *string `json:"profissional_id,omitempty"`
}

// AdminBootstrapPayload dados globais do Super Admin.
type AdminBootstrapPayload struct {
	Tenants []EstablishmentSuperAdminView `json:"tenants"`
	Planos  []PlanoSaas                   `json:"planos"`
}

type BootstrapService struct {
	db *sqlx.DB
}

func NewBootstrapService(db *sqlx.DB) *BootstrapService {
	return &BootstrapService{db: db}
}

// DB expõe o pool para operações pontuais nos handlers.
func (s *BootstrapService) DB() *sqlx.DB {
	return s.db
}

// BuildTenantBootstrap carrega snapshot do estabelecimento para o React.
func (s *BootstrapService) BuildTenantBootstrap(ctx context.Context, establishmentID string) (*TenantBootstrapPayload, error) {
	est, err := s.loadTenantView(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	espSvc := NewEspecialidadeService(s.db)
	especialidades, err := espSvc.ListEspecialidades(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	procSvc := NewProcedimentoService(s.db)
	servicos, err := procSvc.ListServices(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	profSvc := NewProfissionalService(s.db)
	profissionais, err := profSvc.ListProfessionals(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	expedientes, err := s.listExpedientesEstablishment(ctx, establishmentID)
	if err != nil {
		return nil, err
	}
	expByProf := map[string][]ExpedienteProfissional{}
	for _, e := range expedientes {
		expByProf[e.ProfissionalID] = append(expByProf[e.ProfissionalID], e)
	}

	profBootstrap := make([]ProfissionalBootstrap, len(profissionais))
	for i, p := range profissionais {
		profBootstrap[i] = ProfissionalBootstrap{
			Profissional: p,
			Expedientes:  expByProf[p.ID],
		}
		if profBootstrap[i].Expedientes == nil {
			profBootstrap[i].Expedientes = []ExpedienteProfissional{}
		}
	}

	clientes, err := s.listClientes(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	agendamentos, err := s.listAgendamentos(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	lancamentos, err := s.listLancamentos(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	filaSvc := NewFilaEsperaService(s.db)
	fila, err := filaSvc.ListByEstablishment(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	insumoSvc := NewInsumoService(s.db)
	insumos, err := insumoSvc.List(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	var plano *PlanoSaas
	if est.PlanoID != nil && *est.PlanoID != "" {
		planoSvc := NewPlanoSaasService(s.db)
		plano, _ = planoSvc.BuscarPlanoSaasPorID(ctx, *est.PlanoID)
	}

	return &TenantBootstrapPayload{
		Tenant:         *est,
		Plano:          plano,
		Especialidades: especialidades,
		Profissionais:  profBootstrap,
		Servicos:       servicos,
		Clientes:       clientes,
		Agendamentos:   agendamentos,
		Lancamentos:    lancamentos,
		FilaEspera:     fila,
		Insumos:        insumos,
	}, nil
}

// BuildAdminBootstrap lista salões e planos para Super Admin.
func (s *BootstrapService) BuildAdminBootstrap(ctx context.Context) (*AdminBootstrapPayload, error) {
	estSvc := NewEstabelecimentoService(s.db)
	tenants, err := estSvc.ListEstablishmentsSuperAdmin(ctx)
	if err != nil {
		return nil, err
	}
	planoSvc := NewPlanoSaasService(s.db)
	planos, err := planoSvc.ListSaasPlans(ctx)
	if err != nil {
		return nil, err
	}
	return &AdminBootstrapPayload{Tenants: tenants, Planos: planos}, nil
}

func (s *BootstrapService) loadTenantView(ctx context.Context, establishmentID string) (*TenantBootstrapView, error) {
	const query = `
SELECT
    e.id,
    e.nome_comercial,
    e.slug,
    e.logo_url,
    ae.plano_id,
    ae.data_vencimento,
    ae.status AS assinatura_status
FROM estabelecimentos e
LEFT JOIN assinaturas_estabelecimentos ae ON ae.estabelecimento_id = e.id
WHERE e.id = $1 AND e.ativo = TRUE
`
	var row struct {
		ID               string     `db:"id"`
		NomeComercial    string     `db:"nome_comercial"`
		Slug             string     `db:"slug"`
		LogoURL          *string    `db:"logo_url"`
		PlanoID          *string    `db:"plano_id"`
		DataVencimento   *time.Time `db:"data_vencimento"`
		AssinaturaStatus *string    `db:"assinatura_status"`
	}
	if err := s.db.GetContext(ctx, &row, query, establishmentID); err != nil {
		return nil, fmt.Errorf("carregar estabelecimento: %w", err)
	}
	status := "ATIVO"
	if row.AssinaturaStatus != nil {
		switch *row.AssinaturaStatus {
		case "VENCIDO", "SUSPENSO":
			status = "VENCIDO"
		}
	}
	var venc *string
	if row.DataVencimento != nil {
		s := row.DataVencimento.Format("2006-01-02")
		venc = &s
	}
	return &TenantBootstrapView{
		ID:             row.ID,
		Nome:           row.NomeComercial,
		Slug:           row.Slug,
		Status:         status,
		LogoURL:        row.LogoURL,
		PlanoID:        row.PlanoID,
		DataVencimento: venc,
	}, nil
}

func (s *BootstrapService) listExpedientesEstablishment(ctx context.Context, establishmentID string) ([]ExpedienteProfissional, error) {
	const query = `
SELECT ep.id, ep.profissional_id, ep.dia_semana,
       ep.horario_entrada::text, ep.inicio_almoco::text, ep.fim_almoco::text, ep.horario_saida::text
FROM expedientes_profissionais ep
INNER JOIN profissionais p ON p.id = ep.profissional_id
WHERE p.estabelecimento_id = $1
ORDER BY ep.profissional_id, ep.dia_semana
`
	var lista []ExpedienteProfissional
	if err := s.db.SelectContext(ctx, &lista, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar expedientes: %w", err)
	}
	return lista, nil
}

func (s *BootstrapService) listClientes(ctx context.Context, establishmentID string) ([]ClienteBootstrap, error) {
	const query = `
SELECT id, nome, telefone, email, TRUE AS ativo,
       to_char(data_cadastro, 'YYYY-MM-DD') AS criado_em
FROM clientes
WHERE estabelecimento_id = $1
ORDER BY nome
`
	var lista []ClienteBootstrap
	if err := s.db.SelectContext(ctx, &lista, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar clientes: %w", err)
	}
	if lista == nil {
		lista = []ClienteBootstrap{}
	}
	return lista, nil
}

func (s *BootstrapService) listAgendamentos(ctx context.Context, establishmentID string) ([]AgendamentoBootstrap, error) {
	const query = `
SELECT
    a.id,
    a.estabelecimento_id,
    a.profissional_id,
    a.servico_id,
    a.status,
    a.data_hora_inicio,
    a.minutos_invadidos,
    a.aceita_adiantar,
    a.valor_cobrado,
    a.metodo_pagamento,
    a.cobrado_em,
    a.confirmacao_cliente,
    (
        SELECT MAX(n.enviado_em)
        FROM agendamento_notificacoes n
        WHERE n.estabelecimento_id = a.estabelecimento_id
          AND n.agendamento_id = a.id
          AND n.tipo = 'LEMBRETE'
          AND n.status_envio = 'ENVIADO'
    ) AS ultimo_lembrete_enviado_em,
    c.nome AS cliente_nome,
    c.telefone AS cliente_telefone
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
WHERE a.estabelecimento_id = $1
  AND a.data_hora_inicio >= NOW() - INTERVAL '30 days'
  AND a.data_hora_inicio < NOW() + INTERVAL '90 days'
ORDER BY a.data_hora_inicio
`
	type row struct {
		ID                      string     `db:"id"`
		EstabelecimentoID       string     `db:"estabelecimento_id"`
		ProfissionalID          string     `db:"profissional_id"`
		ServicoID               string     `db:"servico_id"`
		Status                  string     `db:"status"`
		DataHoraInicio          time.Time  `db:"data_hora_inicio"`
		MinutosInvadidos        int        `db:"minutos_invadidos"`
		AceitaAdiantar          bool       `db:"aceita_adiantar"`
		ValorCobrado            *float64   `db:"valor_cobrado"`
		MetodoPagamento         *string    `db:"metodo_pagamento"`
		CobradoEm               *time.Time `db:"cobrado_em"`
		ConfirmacaoCliente      string     `db:"confirmacao_cliente"`
		UltimoLembreteEnviadoEm *time.Time `db:"ultimo_lembrete_enviado_em"`
		ClienteNome             string     `db:"cliente_nome"`
		ClienteTelefone         string     `db:"cliente_telefone"`
	}
	var rows []row
	if err := s.db.SelectContext(ctx, &rows, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar agendamentos: %w", err)
	}

	out := make([]AgendamentoBootstrap, 0, len(rows))
	for _, r := range rows {
		item := AgendamentoBootstrap{
			ID:                 r.ID,
			TenantID:           r.EstabelecimentoID,
			ProfissionalID:     r.ProfissionalID,
			ServicoID:          r.ServicoID,
			ServicoIDs:         []string{r.ServicoID},
			AdicionalIDs:       []string{},
			ClienteNome:        r.ClienteNome,
			ClienteTelefone:    r.ClienteTelefone,
			Data:               r.DataHoraInicio.Format("2006-01-02"),
			HoraInicio:         r.DataHoraInicio.Format("15:04"),
			Status:             r.Status,
			MinutosInvadidos:   r.MinutosInvadidos,
			AceitaAdiantar:     r.AceitaAdiantar,
			ValorCobrado:       r.ValorCobrado,
			MetodoPagamento:    r.MetodoPagamento,
			ConfirmacaoCliente: r.ConfirmacaoCliente,
		}
		if r.CobradoEm != nil {
			s := r.CobradoEm.Format("2006-01-02")
			item.CobradoEm = &s
		}
		if r.UltimoLembreteEnviadoEm != nil {
			s := r.UltimoLembreteEnviadoEm.Format(time.RFC3339)
			item.UltimoLembreteEnviadoEm = &s
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *BootstrapService) listLancamentos(ctx context.Context, establishmentID string) ([]LancamentoBootstrap, error) {
	start, end, _ := PeriodoMesAtual()
	inicio, fim := intervaloRelatorio(start, end)
	const query = `
SELECT id, estabelecimento_id, tipo, descricao, valor,
       to_char(data_transacao, 'YYYY-MM-DD') AS data,
       profissional_id
FROM fluxo_caixa
WHERE estabelecimento_id = $1
  AND data_transacao >= $2 AND data_transacao < $3
ORDER BY data_transacao DESC
LIMIT 200
`
	type row struct {
		ID             string  `db:"id"`
		TenantID       string  `db:"estabelecimento_id"`
		Tipo           string  `db:"tipo"`
		Descricao      string  `db:"descricao"`
		Valor          float64 `db:"valor"`
		Data           string  `db:"data"`
		ProfissionalID *string `db:"profissional_id"`
	}
	var rows []row
	if err := s.db.SelectContext(ctx, &rows, query, establishmentID, inicio, fim); err != nil {
		return nil, fmt.Errorf("listar lançamentos: %w", err)
	}
	out := make([]LancamentoBootstrap, len(rows))
	for i, r := range rows {
		out[i] = LancamentoBootstrap{
			ID:             r.ID,
			TenantID:       r.TenantID,
			Tipo:           r.Tipo,
			Valor:          r.Valor,
			Descricao:      r.Descricao,
			Data:           r.Data,
			ProfissionalID: r.ProfissionalID,
		}
	}
	return out, nil
}
