package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTipoLancamentoInvalido = errors.New("tipo de lançamento inválido")
	ErrValorLancamentoInvalido = errors.New("valor do lançamento deve ser maior que zero")
	ErrDescricaoLancamentoObrigatoria = errors.New("descrição do lançamento é obrigatória")
)

type DesempenhoProfissional struct {
	ID               string  `json:"id"`
	Nome             string  `json:"nome"`
	Especialidade    string  `json:"especialidade"`
	ServicosMes      int     `json:"servicos_mes"`
	ComissaoPendente float64 `json:"comissao_pendente"`
}

type DashboardGerencial struct {
	EstabelecimentoNome string                   `json:"estabelecimento_nome"`
	PeriodoLabel        string                   `json:"periodo_label"`
	StartDate           string                   `json:"start_date"`
	EndDate             string                   `json:"end_date"`
	Relatorio           *RelatorioFinanceiro     `json:"relatorio"`
	Equipe              []DesempenhoProfissional `json:"equipe"`
}

// GetDashboardGerencial consolida métricas financeiras e desempenho da equipe no período.
func (s *FinanceiroService) GetDashboardGerencial(
	ctx context.Context,
	establishmentID, estabelecimentoNome string,
	startDate, endDate time.Time,
) (*DashboardGerencial, error) {
	relatorio, err := s.GetEstablishmentFinancialReport(ctx, establishmentID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	equipe, err := s.GetTeamPerformanceReport(ctx, establishmentID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &DashboardGerencial{
		EstabelecimentoNome: estabelecimentoNome,
		PeriodoLabel:        formatarPeriodoLabel(startDate),
		StartDate:           startDate.Format("2006-01-02"),
		EndDate:             endDate.Format("2006-01-02"),
		Relatorio:           relatorio,
		Equipe:              equipe,
	}, nil
}

// GetTeamPerformanceReport lista profissionais com serviços concluídos e comissões pendentes.
func (s *FinanceiroService) GetTeamPerformanceReport(
	ctx context.Context,
	establishmentID string,
	startDate, endDate time.Time,
) ([]DesempenhoProfissional, error) {
	inicio, fim := intervaloRelatorio(startDate, endDate)

	const query = `
SELECT
    p.id,
    p.nome,
    p.especialidade,
    COALESCE((
        SELECT COUNT(*)::INTEGER
        FROM agendamentos a
        WHERE a.estabelecimento_id = p.estabelecimento_id
          AND a.profissional_id = p.id
          AND a.status = 'CONCLUIDO'
          AND a.data_hora_inicio >= $2
          AND a.data_hora_inicio < $3
    ), 0) AS servicos_mes,
    COALESCE((
        SELECT SUM(fc.valor)
        FROM fluxo_caixa fc
        WHERE fc.estabelecimento_id = p.estabelecimento_id
          AND fc.profissional_id = p.id
          AND fc.tipo = 'CUSTO_VARIAVEL'
          AND fc.status_pagamento = 'PENDENTE'
          AND fc.data_transacao >= $2
          AND fc.data_transacao < $3
    ), 0) AS comissao_pendente
FROM profissionais p
WHERE p.estabelecimento_id = $1
ORDER BY p.nome ASC
`
	var equipe []DesempenhoProfissional
	if err := s.db.SelectContext(ctx, &equipe, query, establishmentID, inicio, fim); err != nil {
		return nil, fmt.Errorf("listar desempenho da equipe: %w", err)
	}
	if equipe == nil {
		equipe = []DesempenhoProfissional{}
	}

	for i := range equipe {
		equipe[i].ComissaoPendente = roundMoney(equipe[i].ComissaoPendente)
	}

	return equipe, nil
}

// RegistrarLancamentoCaixa insere uma movimentação rápida no fluxo de caixa.
func (s *FinanceiroService) RegistrarLancamentoCaixa(
	ctx context.Context,
	establishmentID, descricao, tipo string,
	valor float64,
) error {
	descricao = strings.TrimSpace(descricao)
	tipo = strings.ToUpper(strings.TrimSpace(tipo))

	if descricao == "" {
		return ErrDescricaoLancamentoObrigatoria
	}
	if valor <= 0 {
		return ErrValorLancamentoInvalido
	}
	switch tipo {
	case "ENTRADA", "CUSTO_FIXO", "CUSTO_VARIAVEL":
	default:
		return ErrTipoLancamentoInvalido
	}

	const insert = `
INSERT INTO fluxo_caixa (estabelecimento_id, tipo, descricao, valor)
VALUES ($1, $2, $3, $4)
`
	if _, err := s.db.ExecContext(ctx, insert, establishmentID, tipo, descricao, roundMoney(valor)); err != nil {
		return fmt.Errorf("registrar lançamento: %w", err)
	}

	return nil
}

// PeriodoMesAtual retorna início, fim e rótulo do mês corrente.
func PeriodoMesAtual() (time.Time, time.Time, string) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, -1)
	return start, end, formatarPeriodoLabel(start)
}

func formatarPeriodoLabel(ref time.Time) string {
	meses := []string{
		"Janeiro", "Fevereiro", "Março", "Abril", "Maio", "Junho",
		"Julho", "Agosto", "Setembro", "Outubro", "Novembro", "Dezembro",
	}
	return fmt.Sprintf("%s %d", meses[ref.Month()-1], ref.Year())
}
