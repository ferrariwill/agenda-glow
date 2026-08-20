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

var (
	ErrCategoriaServicoNaoEncontrada = errors.New("categoria de serviço não encontrada")
	ErrCategoriaServicoNomeDuplicado = errors.New("já existe categoria com este nome")
)

type CategoriaServico struct {
	ID                string  `db:"id" json:"id"`
	EstabelecimentoID string  `db:"estabelecimento_id" json:"tenant_id"`
	Nome              string  `db:"nome" json:"nome"`
	Icone             *string `db:"icone" json:"icone,omitempty"`
}

type CategoriaServicoService struct {
	db *sqlx.DB
}

func NewCategoriaServicoService(db *sqlx.DB) *CategoriaServicoService {
	return &CategoriaServicoService{db: db}
}

func (s *CategoriaServicoService) List(ctx context.Context, establishmentID string) ([]CategoriaServico, error) {
	const query = `
SELECT id, estabelecimento_id, nome, icone
FROM categorias_servicos
WHERE estabelecimento_id = $1
ORDER BY nome ASC
`
	var lista []CategoriaServico
	if err := s.db.SelectContext(ctx, &lista, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar categorias de serviço: %w", err)
	}
	if lista == nil {
		lista = []CategoriaServico{}
	}
	return lista, nil
}

func (s *CategoriaServicoService) Create(ctx context.Context, establishmentID, nome, icone string) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", fmt.Errorf("nome da categoria é obrigatório")
	}
	icone = strings.TrimSpace(icone)

	const insert = `
INSERT INTO categorias_servicos (estabelecimento_id, nome, icone)
VALUES ($1, $2, NULLIF($3, ''))
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, establishmentID, nome, icone); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", ErrCategoriaServicoNomeDuplicado
		}
		return "", fmt.Errorf("criar categoria de serviço: %w", err)
	}
	return id, nil
}

func (s *CategoriaServicoService) Update(ctx context.Context, establishmentID, id, nome, icone string) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return fmt.Errorf("nome da categoria é obrigatório")
	}
	icone = strings.TrimSpace(icone)

	const update = `
UPDATE categorias_servicos
SET nome = $3, icone = NULLIF($4, '')
WHERE id = $1 AND estabelecimento_id = $2
`
	result, err := s.db.ExecContext(ctx, update, id, establishmentID, nome, icone)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrCategoriaServicoNomeDuplicado
		}
		return fmt.Errorf("atualizar categoria de serviço: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCategoriaServicoNaoEncontrada
	}
	return nil
}

func (s *CategoriaServicoService) Delete(ctx context.Context, establishmentID, id string) error {
	const q = `DELETE FROM categorias_servicos WHERE id = $1 AND estabelecimento_id = $2`
	result, err := s.db.ExecContext(ctx, q, id, establishmentID)
	if err != nil {
		return fmt.Errorf("excluir categoria de serviço: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCategoriaServicoNaoEncontrada
	}
	return nil
}

func (s *CategoriaServicoService) BuscarPorID(ctx context.Context, establishmentID, id string) (*CategoriaServico, error) {
	const query = `
SELECT id, estabelecimento_id, nome, icone
FROM categorias_servicos
WHERE id = $1 AND estabelecimento_id = $2
`
	var cat CategoriaServico
	if err := s.db.GetContext(ctx, &cat, query, id, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCategoriaServicoNaoEncontrada
		}
		return nil, fmt.Errorf("buscar categoria de serviço: %w", err)
	}
	return &cat, nil
}

// ExistsNoTenant valida se a categoria pertence ao estabelecimento.
func (s *CategoriaServicoService) ExistsNoTenant(ctx context.Context, establishmentID, id string) error {
	_, err := s.BuscarPorID(ctx, establishmentID, id)
	return err
}
