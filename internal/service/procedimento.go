package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type ProcedimentoService struct {
	db *sqlx.DB
}

func NewProcedimentoService(db *sqlx.DB) *ProcedimentoService {
	return &ProcedimentoService{db: db}
}

type Servico struct {
	ID                 string              `db:"id" json:"id"`
	Nome               string              `db:"nome" json:"nome"`
	PrecoBase          float64             `db:"preco_base" json:"preco_base"`
	DuracaoBaseMinutos int                 `db:"duracao_base_minutos" json:"duracao_base_minutos"`
	Ativo              bool                `db:"ativo" json:"ativo"`
	Adicionais         []ServicoAdicional  `json:"adicionais,omitempty"`
}

type ServicoAdicional struct {
	ID                        string  `db:"id" json:"id"`
	ServicoID                 string  `db:"servico_id" json:"servico_id"`
	Nome                      string  `db:"nome" json:"nome"`
	PrecoAdicional            float64 `db:"preco_adicional" json:"preco_adicional"`
	DuracaoAdicionalMinutos   int     `db:"duracao_adicional_minutos" json:"duracao_adicional_minutos"`
}

// CreateService cadastra um serviço base vinculado ao estabelecimento.
func (s *ProcedimentoService) CreateService(
	ctx context.Context,
	establishmentID, nome string,
	precoBase float64,
	duracaoBase int,
) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", fmt.Errorf("nome do serviço é obrigatório")
	}
	if precoBase < 0 {
		return "", fmt.Errorf("preço base inválido")
	}
	if duracaoBase <= 0 {
		return "", fmt.Errorf("duração base inválida")
	}

	const insert = `
INSERT INTO servicos (estabelecimento_id, nome, preco_base, duracao_base_minutos, ativo)
VALUES ($1, $2, $3, $4, TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, establishmentID, nome, precoBase, duracaoBase); err != nil {
		return "", fmt.Errorf("cadastrar serviço: %w", err)
	}

	return id, nil
}

// CreateServiceAdditional cadastra variação/adicional em serviço do estabelecimento.
func (s *ProcedimentoService) CreateServiceAdditional(
	ctx context.Context,
	establishmentID, serviceID, nome string,
	precoAdd float64,
	duracaoAdd int,
) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", fmt.Errorf("nome do adicional é obrigatório")
	}
	if precoAdd < 0 {
		return "", fmt.Errorf("preço adicional inválido")
	}
	if duracaoAdd < 0 {
		return "", fmt.Errorf("duração adicional inválida")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := s.validarServicoDoEstabelecimento(ctx, tx, establishmentID, serviceID); err != nil {
		return "", err
	}

	const insert = `
INSERT INTO servico_adicionais (servico_id, nome, preco_adicional, duracao_adicional_minutos)
VALUES ($1, $2, $3, $4)
RETURNING id
`
	var id string
	if err := tx.GetContext(ctx, &id, insert, serviceID, nome, precoAdd, duracaoAdd); err != nil {
		return "", fmt.Errorf("cadastrar adicional: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar transação: %w", err)
	}

	return id, nil
}

func (s *ProcedimentoService) validarServicoDoEstabelecimento(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID, serviceID string,
) error {
	const query = `
SELECT id FROM servicos
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
FOR UPDATE
`
	var id string
	if err := tx.GetContext(ctx, &id, query, serviceID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServicoNaoEncontrado
		}
		return fmt.Errorf("validar serviço: %w", err)
	}
	return nil
}

// ListServices retorna serviços e adicionais isolados por estabelecimento.
func (s *ProcedimentoService) ListServices(ctx context.Context, establishmentID string) ([]Servico, error) {
	const queryServicos = `
SELECT id, nome, preco_base, duracao_base_minutos, ativo
FROM servicos
WHERE estabelecimento_id = $1
ORDER BY nome ASC
`
	var servicos []Servico
	if err := s.db.SelectContext(ctx, &servicos, queryServicos, establishmentID); err != nil {
		return nil, fmt.Errorf("listar serviços: %w", err)
	}
	if len(servicos) == 0 {
		return []Servico{}, nil
	}

	ids := make([]string, len(servicos))
	index := make(map[string]int, len(servicos))
	for i, serv := range servicos {
		ids[i] = serv.ID
		index[serv.ID] = i
		servicos[i].Adicionais = []ServicoAdicional{}
	}

	const queryAdicionais = `
SELECT sa.id, sa.servico_id, sa.nome, sa.preco_adicional, sa.duracao_adicional_minutos
FROM servico_adicionais sa
INNER JOIN servicos s ON s.id = sa.servico_id
WHERE s.estabelecimento_id = $1 AND sa.servico_id = ANY($2)
ORDER BY sa.nome ASC
`
	var adicionais []ServicoAdicional
	if err := s.db.SelectContext(ctx, &adicionais, queryAdicionais, establishmentID, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("listar adicionais: %w", err)
	}

	for _, ad := range adicionais {
		if idx, ok := index[ad.ServicoID]; ok {
			servicos[idx].Adicionais = append(servicos[idx].Adicionais, ad)
		}
	}

	return servicos, nil
}
