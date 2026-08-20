package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

var ErrInsumoNaoEncontrado = errors.New("insumo não encontrado")

type InsumoService struct {
	db *sqlx.DB
}

func NewInsumoService(db *sqlx.DB) *InsumoService {
	return &InsumoService{db: db}
}

type Insumo struct {
	ID                string  `db:"id" json:"id"`
	EstabelecimentoID string  `db:"estabelecimento_id" json:"tenant_id"`
	Nome              string  `db:"nome" json:"nome"`
	Marca             *string `db:"marca" json:"marca,omitempty"`
	Categoria         string  `db:"categoria" json:"categoria"`
	Quantidade        float64 `db:"quantidade" json:"quantidade"`
	EstoqueMinimo     float64 `db:"estoque_minimo" json:"estoque_minimo"`
	EstoqueIdeal      float64 `db:"estoque_ideal" json:"estoque_ideal"`
	ValorUnitario     float64 `db:"valor_unitario" json:"valor_unitario"`
	Unidade           string  `db:"unidade" json:"unidade"`
	ImagemURL         *string `db:"imagem_url" json:"imagem_url,omitempty"`
	InstrucoesUso     *string `db:"instrucoes_uso" json:"instrucoes_uso,omitempty"`
	Ativo             bool    `db:"ativo" json:"ativo"`
}

func (s *InsumoService) List(ctx context.Context, establishmentID string) ([]Insumo, error) {
	return s.Search(ctx, establishmentID, "", 0)
}

// Search lista insumos ativos do tenant. Com q não vazio, filtra por ILIKE em nome/marca.
// limit 0 usa o default 20; máximo 50. Sem q, listagem completa (compatível) — limit ignorado.
func (s *InsumoService) Search(ctx context.Context, establishmentID, q string, limit int) ([]Insumo, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		const query = `
SELECT id, estabelecimento_id, nome, marca, categoria, quantidade, estoque_minimo,
       estoque_ideal, valor_unitario, unidade, imagem_url, instrucoes_uso, ativo
FROM insumos
WHERE estabelecimento_id = $1 AND ativo = TRUE
ORDER BY nome
`
		var list []Insumo
		if err := s.db.SelectContext(ctx, &list, query, establishmentID); err != nil {
			return nil, fmt.Errorf("listar insumos: %w", err)
		}
		if list == nil {
			list = []Insumo{}
		}
		return list, nil
	}

	limit = clampSupplySearchLimit(limit)
	pattern := "%" + q + "%"
	const query = `
SELECT id, estabelecimento_id, nome, marca, categoria, quantidade, estoque_minimo,
       estoque_ideal, valor_unitario, unidade, imagem_url, instrucoes_uso, ativo
FROM insumos
WHERE estabelecimento_id = $1
  AND ativo = TRUE
  AND (nome ILIKE $2 OR COALESCE(marca, '') ILIKE $2)
ORDER BY nome
LIMIT $3
`
	var list []Insumo
	if err := s.db.SelectContext(ctx, &list, query, establishmentID, pattern, limit); err != nil {
		return nil, fmt.Errorf("buscar insumos: %w", err)
	}
	if list == nil {
		list = []Insumo{}
	}
	return list, nil
}

func clampSupplySearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func (s *InsumoService) Create(
	ctx context.Context,
	establishmentID, nome, categoria, unidade string,
	quantidade, estoqueMin, estoqueIdeal, valorUnitario float64,
	marca, imagemURL, instrucoes string,
) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", fmt.Errorf("nome do insumo é obrigatório")
	}
	if categoria == "" {
		categoria = "GERAL"
	}
	if unidade == "" {
		unidade = "un"
	}
	const insert = `
INSERT INTO insumos (
    estabelecimento_id, nome, marca, categoria, quantidade, estoque_minimo,
    estoque_ideal, valor_unitario, unidade, imagem_url, instrucoes_uso, ativo
) VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7, $8, $9, NULLIF($10,''), NULLIF($11,''), TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert,
		establishmentID, nome, marca, categoria, quantidade, estoqueMin, estoqueIdeal,
		valorUnitario, unidade, imagemURL, instrucoes,
	); err != nil {
		return "", fmt.Errorf("criar insumo: %w", err)
	}
	return id, nil
}

func (s *InsumoService) Update(
	ctx context.Context,
	establishmentID, id string,
	nome, categoria, unidade string,
	quantidade, estoqueMin, estoqueIdeal, valorUnitario float64,
	marca, imagemURL, instrucoes string,
	ativo bool,
) error {
	const q = `
UPDATE insumos SET
    nome = $3, marca = NULLIF($4,''), categoria = $5, quantidade = $6,
    estoque_minimo = $7, estoque_ideal = $8, valor_unitario = $9,
    unidade = $10, imagem_url = NULLIF($11,''), instrucoes_uso = NULLIF($12,''), ativo = $13
WHERE id = $1 AND estabelecimento_id = $2
`
	res, err := s.db.ExecContext(ctx, q,
		id, establishmentID, nome, marca, categoria, quantidade,
		estoqueMin, estoqueIdeal, valorUnitario, unidade, imagemURL, instrucoes, ativo,
	)
	if err != nil {
		return fmt.Errorf("atualizar insumo: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInsumoNaoEncontrado
	}
	return nil
}

func (s *InsumoService) AjustarEstoque(ctx context.Context, establishmentID, id string, delta float64) error {
	const q = `
UPDATE insumos
SET quantidade = GREATEST(0, quantidade + $3)
WHERE id = $1 AND estabelecimento_id = $2
`
	res, err := s.db.ExecContext(ctx, q, id, establishmentID, delta)
	if err != nil {
		return fmt.Errorf("ajustar estoque: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInsumoNaoEncontrado
	}
	return nil
}

func (s *InsumoService) Delete(ctx context.Context, establishmentID, id string) error {
	const q = `UPDATE insumos SET ativo = FALSE WHERE id = $1 AND estabelecimento_id = $2`
	res, err := s.db.ExecContext(ctx, q, id, establishmentID)
	if err != nil {
		return fmt.Errorf("excluir insumo: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInsumoNaoEncontrado
	}
	return nil
}

func (s *InsumoService) GetByID(ctx context.Context, establishmentID, id string) (*Insumo, error) {
	const query = `
SELECT id, estabelecimento_id, nome, marca, categoria, quantidade, estoque_minimo,
       estoque_ideal, valor_unitario, unidade, imagem_url, instrucoes_uso, ativo
FROM insumos
WHERE id = $1 AND estabelecimento_id = $2
`
	var item Insumo
	if err := s.db.GetContext(ctx, &item, query, id, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInsumoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar insumo: %w", err)
	}
	return &item, nil
}
