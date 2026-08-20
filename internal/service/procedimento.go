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
	ID                 string             `db:"id" json:"id"`
	Nome               string             `db:"nome" json:"nome"`
	PrecoBase          float64            `db:"preco_base" json:"preco_base"`
	DuracaoBaseMinutos int                `db:"duracao_base_minutos" json:"duracao_base_minutos"`
	Ativo              bool               `db:"ativo" json:"ativo"`
	CategoriaID        *string            `db:"categoria_id" json:"categoria_id,omitempty"`
	CategoriaNome      *string            `db:"categoria_nome" json:"categoria_nome,omitempty"`
	Adicionais         []ServicoAdicional `json:"adicionais"`
}

type ServicoAdicional struct {
	ID                      string  `db:"id" json:"id"`
	ServicoID               string  `db:"servico_id" json:"servico_id"`
	Nome                    string  `db:"nome" json:"nome"`
	PrecoAdicional          float64 `db:"preco_adicional" json:"preco_adicional"`
	DuracaoAdicionalMinutos int     `db:"duracao_adicional_minutos" json:"duracao_adicional_minutos"`
}

// CreateService cadastra um serviço base vinculado ao estabelecimento.
// categoriaID opcional: se informado, deve pertencer ao mesmo estabelecimento.
func (s *ProcedimentoService) CreateService(
	ctx context.Context,
	establishmentID, nome string,
	precoBase float64,
	duracaoBase int,
	categoriaID *string,
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

	catID, err := s.normalizeCategoriaID(ctx, establishmentID, categoriaID)
	if err != nil {
		return "", err
	}

	const insert = `
INSERT INTO servicos (estabelecimento_id, nome, preco_base, duracao_base_minutos, ativo, categoria_id)
VALUES ($1, $2, $3, $4, TRUE, $5)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, establishmentID, nome, precoBase, duracaoBase, catID); err != nil {
		return "", fmt.Errorf("cadastrar serviço: %w", err)
	}

	return id, nil
}

// UpdateService atualiza serviço do tenant (nome, preço, duração, categoria, ativo).
func (s *ProcedimentoService) UpdateService(
	ctx context.Context,
	establishmentID, serviceID, nome string,
	precoBase float64,
	duracaoBase int,
	categoriaID *string,
	ativo bool,
) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return fmt.Errorf("nome do serviço é obrigatório")
	}
	if precoBase < 0 {
		return fmt.Errorf("preço base inválido")
	}
	if duracaoBase <= 0 {
		return fmt.Errorf("duração base inválida")
	}

	catID, err := s.normalizeCategoriaID(ctx, establishmentID, categoriaID)
	if err != nil {
		return err
	}

	const update = `
UPDATE servicos
SET nome = $3, preco_base = $4, duracao_base_minutos = $5, categoria_id = $6, ativo = $7
WHERE id = $1 AND estabelecimento_id = $2
`
	result, err := s.db.ExecContext(ctx, update, serviceID, establishmentID, nome, precoBase, duracaoBase, catID, ativo)
	if err != nil {
		return fmt.Errorf("atualizar serviço: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrServicoNaoEncontrado
	}
	return nil
}

func (s *ProcedimentoService) normalizeCategoriaID(
	ctx context.Context,
	establishmentID string,
	categoriaID *string,
) (any, error) {
	if categoriaID == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*categoriaID)
	if trimmed == "" {
		return nil, nil
	}
	const q = `SELECT id FROM categorias_servicos WHERE id = $1 AND estabelecimento_id = $2`
	var id string
	if err := s.db.GetContext(ctx, &id, q, trimmed, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoriaServicoNaoEncontrada
		}
		return nil, fmt.Errorf("validar categoria: %w", err)
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
SELECT s.id, s.nome, s.preco_base, s.duracao_base_minutos, s.ativo,
       s.categoria_id, c.nome AS categoria_nome
FROM servicos s
LEFT JOIN categorias_servicos c
  ON c.id = s.categoria_id AND c.estabelecimento_id = s.estabelecimento_id
WHERE s.estabelecimento_id = $1
ORDER BY s.nome ASC
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
