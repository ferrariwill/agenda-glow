package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrAgendamentoNaoEncontrado = errors.New("agendamento não encontrado")
	ErrAgendamentoJaConcluido   = errors.New("agendamento já foi concluído")
	ErrAgendamentoCancelado     = errors.New("agendamento cancelado não pode ser concluído")
	ErrAgendamentoJaCobrado     = errors.New("este procedimento já foi cobrado")
	ErrAssinaturaSemCreditos    = errors.New("assinatura sem visitas restantes")
)

type FinanceiroService struct {
	db *sqlx.DB
}

func NewFinanceiroService(db *sqlx.DB) *FinanceiroService {
	return &FinanceiroService{db: db}
}

type atendimentoFinanceiro struct {
	ID                  string  `db:"id"`
	EstabelecimentoID   string  `db:"estabelecimento_id"`
	ClienteTelefone     string  `db:"cliente_telefone"`
	ProfissionalID      string  `db:"profissional_id"`
	ProfissionalNome    string  `db:"profissional_nome"`
	ServicoNome         string  `db:"servico_nome"`
	ValorTotal          float64 `db:"valor_total"`
	ComissaoPorcentagem float64 `db:"comissao_porcentagem"`
	Status              string  `db:"status"`
	ViaClubeAssinatura  bool    `db:"via_clube_assinatura"`
}

type creditoAssinaturaConsumido struct {
	PlanoNome                string  `db:"plano_nome"`
	ValorRepasseProfissional float64 `db:"valor_repasse_profissional"`
}

const queryAtendimentoFinanceiro = `
SELECT
    a.id,
    a.estabelecimento_id,
    c.telefone AS cliente_telefone,
    a.profissional_id,
    p.nome AS profissional_nome,
    s.nome AS servico_nome,
    s.preco_base + COALESCE(adds.total_adicional, 0) AS valor_total,
    p.comissao_porcentagem,
    a.status,
    a.via_clube_assinatura
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
INNER JOIN profissionais p ON p.id = a.profissional_id AND p.estabelecimento_id = a.estabelecimento_id
INNER JOIN servicos s ON s.id = a.servico_id AND s.estabelecimento_id = a.estabelecimento_id
LEFT JOIN (
    SELECT aa.agendamento_id, SUM(sa.preco_adicional) AS total_adicional
    FROM agendamento_adicionais aa
    INNER JOIN servico_adicionais sa ON sa.id = aa.adicional_id
    GROUP BY aa.agendamento_id
) adds ON adds.agendamento_id = a.id
WHERE a.id = $1
FOR UPDATE OF a
`

func (s *FinanceiroService) ConcluirAtendimento(ctx context.Context, agendamentoID string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	atendimento, err := s.buscarAtendimento(ctx, tx, agendamentoID)
	if err != nil {
		return err
	}

	switch atendimento.Status {
	case "CONCLUIDO":
		return ErrAgendamentoJaConcluido
	case "CANCELADO":
		return ErrAgendamentoCancelado
	}

	var credito *creditoAssinaturaConsumido
	if atendimento.ViaClubeAssinatura {
		credito, err = s.consumirCreditoAssinatura(ctx, tx, atendimento.EstabelecimentoID, atendimento.ClienteTelefone)
		if err != nil {
			return err
		}
	}

	if err := s.marcarAgendamentoConcluido(ctx, tx, agendamentoID); err != nil {
		return err
	}

	if atendimento.ViaClubeAssinatura {
		if credito != nil && credito.ValorRepasseProfissional > 0 {
			if err := s.registrarRepasseClube(ctx, tx, atendimento, credito); err != nil {
				return err
			}
		}
	} else {
		comissao := calcularComissao(atendimento.ValorTotal, atendimento.ComissaoPorcentagem)
		if err := s.registrarLancamentosAvulsos(ctx, tx, atendimento, comissao); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar transação: %w", err)
	}

	return nil
}

func (s *FinanceiroService) buscarAtendimento(
	ctx context.Context,
	tx *sqlx.Tx,
	agendamentoID string,
) (*atendimentoFinanceiro, error) {
	var atendimento atendimentoFinanceiro
	if err := tx.GetContext(ctx, &atendimento, queryAtendimentoFinanceiro, agendamentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgendamentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar agendamento: %w", err)
	}

	return &atendimento, nil
}

func (s *FinanceiroService) marcarAgendamentoConcluido(ctx context.Context, tx *sqlx.Tx, agendamentoID string) error {
	const update = `
UPDATE agendamentos
SET status = 'CONCLUIDO'
WHERE id = $1
`
	if _, err := tx.ExecContext(ctx, update, agendamentoID); err != nil {
		return fmt.Errorf("atualizar status do agendamento: %w", err)
	}

	return nil
}

func (s *FinanceiroService) registrarLancamentosAvulsos(
	ctx context.Context,
	tx *sqlx.Tx,
	atendimento *atendimentoFinanceiro,
	comissao float64,
) error {
	const insertEntrada = `
INSERT INTO fluxo_caixa (estabelecimento_id, tipo, descricao, valor, profissional_id)
VALUES ($1, 'ENTRADA', $2, $3, NULL)
`
	descricaoEntrada := fmt.Sprintf(
		"Atendimento: %s — %s",
		atendimento.ServicoNome,
		atendimento.ProfissionalNome,
	)
	if _, err := tx.ExecContext(ctx, insertEntrada, atendimento.EstabelecimentoID, descricaoEntrada, atendimento.ValorTotal); err != nil {
		return fmt.Errorf("registrar entrada no caixa: %w", err)
	}

	const insertComissao = `
INSERT INTO fluxo_caixa (estabelecimento_id, tipo, descricao, valor, profissional_id, status_pagamento)
VALUES ($1, 'CUSTO_VARIAVEL', $2, $3, $4, 'PENDENTE')
`
	descricaoComissao := fmt.Sprintf(
		"Comissão (%g%%): %s — %s",
		atendimento.ComissaoPorcentagem,
		atendimento.ServicoNome,
		atendimento.ProfissionalNome,
	)
	if _, err := tx.ExecContext(
		ctx,
		insertComissao,
		atendimento.EstabelecimentoID,
		descricaoComissao,
		comissao,
		atendimento.ProfissionalID,
	); err != nil {
		return fmt.Errorf("registrar comissão no caixa: %w", err)
	}

	return nil
}

func (s *FinanceiroService) registrarRepasseClube(
	ctx context.Context,
	tx *sqlx.Tx,
	atendimento *atendimentoFinanceiro,
	credito *creditoAssinaturaConsumido,
) error {
	const insertRepasse = `
INSERT INTO fluxo_caixa (estabelecimento_id, tipo, descricao, valor, profissional_id, status_pagamento)
VALUES ($1, 'CUSTO_VARIAVEL', $2, $3, $4, 'PENDENTE')
`
	descricao := fmt.Sprintf(
		"Repasse clube (%s): %s — %s",
		credito.PlanoNome,
		atendimento.ServicoNome,
		atendimento.ProfissionalNome,
	)
	if _, err := tx.ExecContext(
		ctx,
		insertRepasse,
		atendimento.EstabelecimentoID,
		descricao,
		credito.ValorRepasseProfissional,
		atendimento.ProfissionalID,
	); err != nil {
		return fmt.Errorf("registrar repasse do clube: %w", err)
	}

	return nil
}

func (s *FinanceiroService) consumirCreditoAssinatura(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, telefone string,
) (*creditoAssinaturaConsumido, error) {
	const lockAssinante = `
SELECT ca.id
FROM clientes_assinantes ca
INNER JOIN planos_assinatura pa ON pa.id = ca.plano_id AND pa.estabelecimento_id = ca.estabelecimento_id
WHERE ca.estabelecimento_id = $1
  AND ca.cliente_telefone = $2
  AND ca.status = 'ATIVO'
  AND pa.ativo = TRUE
FOR UPDATE OF ca
`
	var assinanteID string
	if err := tx.GetContext(ctx, &assinanteID, lockAssinante, estabelecimentoID, telefone); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssinaturaSemCreditos
		}
		return nil, fmt.Errorf("bloquear assinatura do cliente: %w", err)
	}

	const debitarCredito = `
UPDATE clientes_assinantes ca
SET visitas_restantes = visitas_restantes - 1
FROM planos_assinatura pa
WHERE ca.id = $1
  AND ca.plano_id = pa.id
  AND ca.estabelecimento_id = pa.estabelecimento_id
  AND ca.status = 'ATIVO'
  AND ca.visitas_restantes > 0
  AND pa.ativo = TRUE
RETURNING pa.nome AS plano_nome, pa.valor_repasse_profissional
`
	var credito creditoAssinaturaConsumido
	if err := tx.GetContext(ctx, &credito, debitarCredito, assinanteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAssinaturaSemCreditos
		}
		return nil, fmt.Errorf("debitar crédito da assinatura: %w", err)
	}

	return &credito, nil
}

// CobrancaInput espelha registrarCobrancaAgendamento do protótipo React.
type CobrancaInput struct {
	Metodo   string
	Valor    *float64
	Concluir bool
}

// CobrarAtendimento registra cobrança (método/valor) e opcionalmente conclui o atendimento.
func (s *FinanceiroService) CobrarAtendimento(
	ctx context.Context,
	establishmentID, agendamentoID string,
	in CobrancaInput,
) error {
	if !in.Concluir {
		return s.registrarCobrancaSemConcluir(ctx, establishmentID, agendamentoID, in)
	}
	return s.cobrarEConcluir(ctx, establishmentID, agendamentoID, in)
}

type atendimentoCobranca struct {
	atendimentoFinanceiro
	CobradoEm *time.Time `db:"cobrado_em"`
	ClienteNome string   `db:"cliente_nome"`
}

func (s *FinanceiroService) buscarAtendimentoDoEstabelecimento(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID, agendamentoID string,
) (*atendimentoCobranca, error) {
	const query = `
SELECT
    a.id,
    a.estabelecimento_id,
    c.telefone AS cliente_telefone,
    c.nome AS cliente_nome,
    a.profissional_id,
    p.nome AS profissional_nome,
    s.nome AS servico_nome,
    s.preco_base + COALESCE(adds.total_adicional, 0) AS valor_total,
    p.comissao_porcentagem,
    a.status,
    a.via_clube_assinatura,
    a.cobrado_em
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
INNER JOIN profissionais p ON p.id = a.profissional_id AND p.estabelecimento_id = a.estabelecimento_id
INNER JOIN servicos s ON s.id = a.servico_id AND s.estabelecimento_id = a.estabelecimento_id
LEFT JOIN (
    SELECT aa.agendamento_id, SUM(sa.preco_adicional) AS total_adicional
    FROM agendamento_adicionais aa
    INNER JOIN servico_adicionais sa ON sa.id = aa.adicional_id
    GROUP BY aa.agendamento_id
) adds ON adds.agendamento_id = a.id
WHERE a.id = $1 AND a.estabelecimento_id = $2
FOR UPDATE OF a
`
	var row atendimentoCobranca
	if err := tx.GetContext(ctx, &row, query, agendamentoID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAgendamentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar atendimento: %w", err)
	}
	return &row, nil
}

func (s *FinanceiroService) registrarCobrancaSemConcluir(
	ctx context.Context,
	establishmentID, agendamentoID string,
	in CobrancaInput,
) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	atendimento, err := s.buscarAtendimentoDoEstabelecimento(ctx, tx, establishmentID, agendamentoID)
	if err != nil {
		return err
	}
	if atendimento.CobradoEm != nil {
		return ErrAgendamentoJaCobrado
	}

	valor := atendimento.ValorTotal
	if in.Valor != nil && *in.Valor > 0 {
		valor = roundMoney(*in.Valor)
	}
	if valor <= 0 {
		return fmt.Errorf("valor inválido para cobrança")
	}

	hoje := time.Now().Format("2006-01-02")
	const insertEntrada = `
INSERT INTO fluxo_caixa (estabelecimento_id, tipo, descricao, valor, profissional_id)
VALUES ($1, 'ENTRADA', $2, $3, NULL)
`
	desc := fmt.Sprintf("Cobrança — %s (%s)", atendimento.ServicoNome, atendimento.ClienteNome)
	if _, err := tx.ExecContext(ctx, insertEntrada, establishmentID, desc, valor); err != nil {
		return fmt.Errorf("registrar entrada: %w", err)
	}

	const upd = `
UPDATE agendamentos
SET valor_cobrado = $3, metodo_pagamento = NULLIF($4,''), cobrado_em = $5::DATE
WHERE id = $1 AND estabelecimento_id = $2
`
	if _, err := tx.ExecContext(ctx, upd, agendamentoID, establishmentID, valor, in.Metodo, hoje); err != nil {
		return fmt.Errorf("marcar cobrança: %w", err)
	}

	return tx.Commit()
}

func (s *FinanceiroService) cobrarEConcluir(
	ctx context.Context,
	establishmentID, agendamentoID string,
	in CobrancaInput,
) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	atendimento, err := s.buscarAtendimentoDoEstabelecimento(ctx, tx, establishmentID, agendamentoID)
	if err != nil {
		return err
	}
	if atendimento.CobradoEm != nil {
		return ErrAgendamentoJaCobrado
	}
	switch atendimento.Status {
	case "CONCLUIDO":
		return ErrAgendamentoJaConcluido
	case "CANCELADO":
		return ErrAgendamentoCancelado
	}

	valor := atendimento.ValorTotal
	if in.Valor != nil && *in.Valor > 0 {
		valor = roundMoney(*in.Valor)
	}

	var credito *creditoAssinaturaConsumido
	if atendimento.ViaClubeAssinatura {
		credito, err = s.consumirCreditoAssinatura(ctx, tx, atendimento.EstabelecimentoID, atendimento.ClienteTelefone)
		if err != nil {
			return err
		}
	}

	hoje := time.Now().Format("2006-01-02")
	const upd = `
UPDATE agendamentos
SET status = 'CONCLUIDO', valor_cobrado = $3, metodo_pagamento = NULLIF($4,''), cobrado_em = $5::DATE
WHERE id = $1 AND estabelecimento_id = $2
`
	if _, err := tx.ExecContext(ctx, upd, agendamentoID, establishmentID, valor, in.Metodo, hoje); err != nil {
		return fmt.Errorf("concluir agendamento: %w", err)
	}

	if atendimento.ViaClubeAssinatura {
		if credito != nil && credito.ValorRepasseProfissional > 0 {
			if err := s.registrarRepasseClube(ctx, tx, &atendimento.atendimentoFinanceiro, credito); err != nil {
				return err
			}
		}
	} else {
		comissao := calcularComissao(valor, atendimento.ComissaoPorcentagem)
		fin := atendimento.atendimentoFinanceiro
		fin.ValorTotal = valor
		if err := s.registrarLancamentosAvulsos(ctx, tx, &fin, comissao); err != nil {
			return err
		}
	}

	return tx.Commit()
}
