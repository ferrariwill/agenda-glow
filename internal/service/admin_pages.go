package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type StatusLimiteEquipe struct {
	LimiteProfissionais int  `json:"limite_profissionais"`
	TotalAtivos         int  `json:"total_ativos"`
	LimiteAtingido      bool `json:"limite_atingido"`
}

// GetStatusLimiteEquipe informa uso do plano SaaS para cadastro de profissionais.
func (s *ProfissionalService) GetStatusLimiteEquipe(ctx context.Context, establishmentID string) (*StatusLimiteEquipe, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	limite, err := s.buscarLimitePlano(ctx, tx, establishmentID)
	if err != nil {
		return nil, err
	}

	const contarAtivos = `
SELECT COUNT(*)::INTEGER FROM profissionais
WHERE estabelecimento_id = $1 AND ativo = TRUE
`
	var totalAtivos int
	if err := tx.GetContext(ctx, &totalAtivos, contarAtivos, establishmentID); err != nil {
		return nil, fmt.Errorf("contar profissionais ativos: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar consulta de limite: %w", err)
	}

	return &StatusLimiteEquipe{
		LimiteProfissionais: limite.LimiteProfissionais,
		TotalAtivos:         totalAtivos,
		LimiteAtingido:      totalAtivos >= limite.LimiteProfissionais,
	}, nil
}

type LancamentoCaixa struct {
	ID              string    `db:"id" json:"id"`
	Tipo            string    `db:"tipo" json:"tipo"`
	Descricao       string    `db:"descricao" json:"descricao"`
	Valor           float64   `db:"valor" json:"valor"`
	DataTransacao   time.Time `db:"data_transacao" json:"data_transacao"`
}

// ListLancamentosMes retorna movimentações recentes do mês corrente.
func (s *FinanceiroService) ListLancamentosMes(ctx context.Context, establishmentID string) ([]LancamentoCaixa, error) {
	start, end, _ := PeriodoMesAtual()
	inicio, fim := intervaloRelatorio(start, end)

	const query = `
SELECT id, tipo, descricao, valor, data_transacao
FROM fluxo_caixa
WHERE estabelecimento_id = $1
  AND data_transacao >= $2
  AND data_transacao < $3
ORDER BY data_transacao DESC
LIMIT 100
`
	var lancamentos []LancamentoCaixa
	if err := s.db.SelectContext(ctx, &lancamentos, query, establishmentID, inicio, fim); err != nil {
		return nil, fmt.Errorf("listar lançamentos do mês: %w", err)
	}
	if lancamentos == nil {
		lancamentos = []LancamentoCaixa{}
	}
	return lancamentos, nil
}

// GetCaixaFluxoPage consolida extrato e resumo financeiro do mês.
func (s *FinanceiroService) GetCaixaFluxoPage(ctx context.Context, establishmentID string) (*CaixaFluxoPage, error) {
	start, end, label := PeriodoMesAtual()

	lancamentos, err := s.ListLancamentosMes(ctx, establishmentID)
	if err != nil {
		return nil, err
	}

	relatorio, err := s.GetEstablishmentFinancialReport(ctx, establishmentID, start, end)
	if err != nil {
		return nil, err
	}

	return &CaixaFluxoPage{
		PeriodoLabel: label,
		StartDate:    start.Format("2006-01-02"),
		EndDate:      end.Format("2006-01-02"),
		Lancamentos:  lancamentos,
		Relatorio:    relatorio,
	}, nil
}

type CaixaFluxoPage struct {
	PeriodoLabel string              `json:"periodo_label"`
	StartDate    string              `json:"start_date"`
	EndDate      string              `json:"end_date"`
	Lancamentos  []LancamentoCaixa   `json:"lancamentos"`
	Relatorio    *RelatorioFinanceiro `json:"relatorio"`
}

// BuscarAdicionalPorID retorna adicional recém-criado para partial HTMX.
func (s *ProcedimentoService) BuscarAdicionalPorID(ctx context.Context, establishmentID, adicionalID string) (*ServicoAdicional, error) {
	const query = `
SELECT sa.id, sa.servico_id, sa.nome, sa.preco_adicional, sa.duracao_adicional_minutos
FROM servico_adicionais sa
INNER JOIN servicos s ON s.id = sa.servico_id
WHERE sa.id = $1 AND s.estabelecimento_id = $2
`
	var ad ServicoAdicional
	if err := s.db.GetContext(ctx, &ad, query, adicionalID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServicoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar adicional: %w", err)
	}
	return &ad, nil
}

// BuscarServicoPorID retorna serviço recém-criado para partial HTMX.
func (s *ProcedimentoService) BuscarServicoPorID(ctx context.Context, establishmentID, serviceID string) (*Servico, error) {
	const query = `
SELECT id, nome, preco_base, duracao_base_minutos, ativo
FROM servicos
WHERE id = $1 AND estabelecimento_id = $2
`
	var serv Servico
	if err := s.db.GetContext(ctx, &serv, query, serviceID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServicoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar serviço: %w", err)
	}
	serv.Adicionais = []ServicoAdicional{}
	return &serv, nil
}

// BuscarProfissionalPorID retorna profissional recém-cadastrada para partial HTMX.
func (s *ProfissionalService) BuscarProfissionalPorID(ctx context.Context, establishmentID, professionalID string) (*Profissional, error) {
	const query = `
SELECT id, nome, especialidade, comissao_porcentagem, ativo
FROM profissionais
WHERE id = $1 AND estabelecimento_id = $2
`
	var p Profissional
	if err := s.db.GetContext(ctx, &p, query, professionalID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfissionalNaoEncontrado
		}
		return nil, fmt.Errorf("buscar profissional: %w", err)
	}
	return &p, nil
}
