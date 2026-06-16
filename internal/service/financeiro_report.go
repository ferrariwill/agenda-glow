package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrProfissionalFinanceiroNaoEncontrado = errors.New("profissional não encontrado")
	ErrNenhumaComissaoPendente             = errors.New("nenhuma comissão pendente no período")
)

type RelatorioFinanceiro struct {
	TotalEntradas           float64 `json:"total_entradas"`
	TotalCustosFixos        float64 `json:"total_custos_fixos"`
	TotalComissoesPendentes float64 `json:"total_comissoes_pendentes"`
	LucroLiquidoEstimado    float64 `json:"lucro_liquido_estimado"`
}

type ComissaoPendenteItem struct {
	ID             string    `db:"id" json:"id"`
	Descricao      string    `db:"descricao" json:"descricao"`
	Valor          float64   `db:"valor" json:"valor"`
	DataTransacao  time.Time `db:"data_transacao" json:"data_transacao"`
}

type ComissaoPendenteProfissional struct {
	ProfissionalID string                 `json:"profissional_id"`
	TotalPendente  float64                `json:"total_pendente"`
	Itens          []ComissaoPendenteItem `json:"itens"`
}

// GetPartnerCommissionsReport soma comissões pendentes da profissional no período.
func (s *FinanceiroService) GetPartnerCommissionsReport(
	ctx context.Context,
	establishmentID, professionalID string,
	startDate, endDate time.Time,
) (float64, error) {
	inicio, fim := intervaloRelatorio(startDate, endDate)

	const query = `
SELECT COALESCE(SUM(valor), 0)
FROM fluxo_caixa
WHERE estabelecimento_id = $1
  AND profissional_id = $2
  AND tipo = 'CUSTO_VARIAVEL'
  AND status_pagamento = 'PENDENTE'
  AND data_transacao >= $3
  AND data_transacao < $4
`
	var total float64
	if err := s.db.GetContext(ctx, &total, query, establishmentID, professionalID, inicio, fim); err != nil {
		return 0, fmt.Errorf("somar comissões pendentes: %w", err)
	}

	return total, nil
}

// GetPartnerPendingCommissions retorna total e detalhamento de comissões pendentes.
func (s *FinanceiroService) GetPartnerPendingCommissions(
	ctx context.Context,
	establishmentID, professionalID string,
	startDate, endDate time.Time,
) (*ComissaoPendenteProfissional, error) {
	if err := s.validarProfissionalEstabelecimento(ctx, establishmentID, professionalID); err != nil {
		return nil, err
	}

	inicio, fim := intervaloRelatorio(startDate, endDate)

	const query = `
SELECT id, descricao, valor, data_transacao
FROM fluxo_caixa
WHERE estabelecimento_id = $1
  AND profissional_id = $2
  AND tipo = 'CUSTO_VARIAVEL'
  AND status_pagamento = 'PENDENTE'
  AND data_transacao >= $3
  AND data_transacao < $4
ORDER BY data_transacao ASC
`
	var itens []ComissaoPendenteItem
	if err := s.db.SelectContext(ctx, &itens, query, establishmentID, professionalID, inicio, fim); err != nil {
		return nil, fmt.Errorf("listar comissões pendentes: %w", err)
	}
	if itens == nil {
		itens = []ComissaoPendenteItem{}
	}

	var total float64
	for _, item := range itens {
		total = roundMoney(total + item.Valor)
	}

	return &ComissaoPendenteProfissional{
		ProfissionalID: professionalID,
		TotalPendente:  total,
		Itens:          itens,
	}, nil
}

// GetEstablishmentFinancialReport consolida entradas, custos e lucro do período.
func (s *FinanceiroService) GetEstablishmentFinancialReport(
	ctx context.Context,
	establishmentID string,
	startDate, endDate time.Time,
) (*RelatorioFinanceiro, error) {
	inicio, fim := intervaloRelatorio(startDate, endDate)

	const query = `
SELECT
    COALESCE(SUM(CASE WHEN tipo = 'ENTRADA' THEN valor ELSE 0 END), 0) AS total_entradas,
    COALESCE(SUM(CASE WHEN tipo = 'CUSTO_FIXO' THEN valor ELSE 0 END), 0) AS total_custos_fixos,
    COALESCE(SUM(CASE
        WHEN tipo = 'CUSTO_VARIAVEL' AND status_pagamento = 'PENDENTE' THEN valor
        ELSE 0
    END), 0) AS total_comissoes_pendentes,
    COALESCE(SUM(CASE WHEN tipo = 'CUSTO_VARIAVEL' THEN valor ELSE 0 END), 0) AS total_comissoes
FROM fluxo_caixa
WHERE estabelecimento_id = $1
  AND data_transacao >= $2
  AND data_transacao < $3
`
	var row struct {
		TotalEntradas           float64 `db:"total_entradas"`
		TotalCustosFixos        float64 `db:"total_custos_fixos"`
		TotalComissoesPendentes float64 `db:"total_comissoes_pendentes"`
		TotalComissoes          float64 `db:"total_comissoes"`
	}
	if err := s.db.GetContext(ctx, &row, query, establishmentID, inicio, fim); err != nil {
		return nil, fmt.Errorf("gerar relatório financeiro: %w", err)
	}

	lucro := roundMoney(row.TotalEntradas - row.TotalCustosFixos - row.TotalComissoes)

	return &RelatorioFinanceiro{
		TotalEntradas:           row.TotalEntradas,
		TotalCustosFixos:        row.TotalCustosFixos,
		TotalComissoesPendentes: row.TotalComissoesPendentes,
		LucroLiquidoEstimado:    lucro,
	}, nil
}

// PayPartnerCommissions quita comissões pendentes da profissional no período.
func (s *FinanceiroService) PayPartnerCommissions(
	ctx context.Context,
	establishmentID, professionalID string,
	startDate, endDate time.Time,
) error {
	if err := s.validarProfissionalEstabelecimento(ctx, establishmentID, professionalID); err != nil {
		return err
	}

	inicio, fim := intervaloRelatorio(startDate, endDate)
	observacao := fmt.Sprintf(" | LIQUIDADO em %s", time.Now().Format("02/01/2006 15:04"))

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lockPendentes = `
SELECT id
FROM fluxo_caixa
WHERE estabelecimento_id = $1
  AND profissional_id = $2
  AND tipo = 'CUSTO_VARIAVEL'
  AND status_pagamento = 'PENDENTE'
  AND data_transacao >= $3
  AND data_transacao < $4
FOR UPDATE
`
	var ids []string
	if err := tx.SelectContext(ctx, &ids, lockPendentes, establishmentID, professionalID, inicio, fim); err != nil {
		return fmt.Errorf("bloquear comissões pendentes: %w", err)
	}
	if len(ids) == 0 {
		return ErrNenhumaComissaoPendente
	}

	const update = `
UPDATE fluxo_caixa
SET status_pagamento = 'PAGO',
    descricao = descricao || $5
WHERE estabelecimento_id = $1
  AND profissional_id = $2
  AND tipo = 'CUSTO_VARIAVEL'
  AND status_pagamento = 'PENDENTE'
  AND data_transacao >= $3
  AND data_transacao < $4
`
	result, err := tx.ExecContext(ctx, update, establishmentID, professionalID, inicio, fim, observacao)
	if err != nil {
		return fmt.Errorf("quitar comissões: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar comissões quitadas: %w", err)
	}
	if rows == 0 {
		return ErrNenhumaComissaoPendente
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar transação: %w", err)
	}

	return nil
}

func (s *FinanceiroService) validarProfissionalEstabelecimento(ctx context.Context, establishmentID, professionalID string) error {
	const query = `
SELECT id FROM profissionais
WHERE id = $1 AND estabelecimento_id = $2
`
	var id string
	if err := s.db.GetContext(ctx, &id, query, professionalID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfissionalFinanceiroNaoEncontrado
		}
		return fmt.Errorf("validar profissional: %w", err)
	}
	return nil
}

func intervaloRelatorio(startDate, endDate time.Time) (time.Time, time.Time) {
	loc := startDate.Location()
	inicio := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc)
	fimDia := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, loc)
	fim := fimDia.Add(24 * time.Hour)
	return inicio, fim
}
