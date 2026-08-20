package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrServicoInsumoNaoEncontrado = errors.New("vínculo serviço-insumo não encontrado")
	ErrServicoInsumoDuplicado     = errors.New("insumo já vinculado a este serviço")
)

type ServicoInsumo struct {
	ID            string  `db:"id" json:"id"`
	ServicoID     string  `db:"servico_id" json:"servico_id"`
	InsumoID      string  `db:"insumo_id" json:"insumo_id"`
	InsumoNome    string  `db:"insumo_nome" json:"insumo_nome"`
	InsumoUnidade string  `db:"insumo_unidade" json:"insumo_unidade"`
	QuantidadeUso float64 `db:"quantidade_uso" json:"quantidade_uso"`
}

type ServicoInsumoService struct {
	db *sqlx.DB
}

func NewServicoInsumoService(db *sqlx.DB) *ServicoInsumoService {
	return &ServicoInsumoService{db: db}
}

func (s *ServicoInsumoService) ListByServico(ctx context.Context, establishmentID, servicoID string) ([]ServicoInsumo, error) {
	if err := s.validarServico(ctx, establishmentID, servicoID); err != nil {
		return nil, err
	}

	const query = `
SELECT si.id, si.servico_id, si.insumo_id, i.nome AS insumo_nome, i.unidade AS insumo_unidade, si.quantidade_uso
FROM servico_insumos si
INNER JOIN insumos i ON i.id = si.insumo_id AND i.estabelecimento_id = si.estabelecimento_id
WHERE si.estabelecimento_id = $1 AND si.servico_id = $2
ORDER BY i.nome ASC
`
	var lista []ServicoInsumo
	if err := s.db.SelectContext(ctx, &lista, query, establishmentID, servicoID); err != nil {
		return nil, fmt.Errorf("listar insumos do serviço: %w", err)
	}
	if lista == nil {
		lista = []ServicoInsumo{}
	}
	return lista, nil
}

func (s *ServicoInsumoService) Create(
	ctx context.Context,
	establishmentID, servicoID, insumoID string,
	quantidadeUso float64,
) (string, error) {
	if quantidadeUso <= 0 {
		return "", fmt.Errorf("quantidade_uso deve ser maior que zero")
	}
	if err := s.validarServico(ctx, establishmentID, servicoID); err != nil {
		return "", err
	}
	if err := s.validarInsumo(ctx, establishmentID, insumoID); err != nil {
		return "", err
	}

	const insert = `
INSERT INTO servico_insumos (estabelecimento_id, servico_id, insumo_id, quantidade_uso)
VALUES ($1, $2, $3, $4)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, establishmentID, servicoID, insumoID, quantidadeUso); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", ErrServicoInsumoDuplicado
		}
		return "", fmt.Errorf("vincular insumo ao serviço: %w", err)
	}
	return id, nil
}

func (s *ServicoInsumoService) Update(
	ctx context.Context,
	establishmentID, servicoID, linkID string,
	quantidadeUso float64,
) error {
	if quantidadeUso <= 0 {
		return fmt.Errorf("quantidade_uso deve ser maior que zero")
	}
	if err := s.validarServico(ctx, establishmentID, servicoID); err != nil {
		return err
	}

	const update = `
UPDATE servico_insumos
SET quantidade_uso = $4
WHERE id = $1 AND estabelecimento_id = $2 AND servico_id = $3
`
	result, err := s.db.ExecContext(ctx, update, linkID, establishmentID, servicoID, quantidadeUso)
	if err != nil {
		return fmt.Errorf("atualizar vínculo serviço-insumo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrServicoInsumoNaoEncontrado
	}
	return nil
}

func (s *ServicoInsumoService) Delete(ctx context.Context, establishmentID, servicoID, linkID string) error {
	if err := s.validarServico(ctx, establishmentID, servicoID); err != nil {
		return err
	}

	const q = `
DELETE FROM servico_insumos
WHERE id = $1 AND estabelecimento_id = $2 AND servico_id = $3
`
	result, err := s.db.ExecContext(ctx, q, linkID, establishmentID, servicoID)
	if err != nil {
		return fmt.Errorf("excluir vínculo serviço-insumo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrServicoInsumoNaoEncontrado
	}
	return nil
}

func (s *ServicoInsumoService) validarServico(ctx context.Context, establishmentID, servicoID string) error {
	const query = `SELECT id FROM servicos WHERE id = $1 AND estabelecimento_id = $2`
	var id string
	if err := s.db.GetContext(ctx, &id, query, servicoID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServicoNaoEncontrado
		}
		return fmt.Errorf("validar serviço: %w", err)
	}
	return nil
}

func (s *ServicoInsumoService) validarInsumo(ctx context.Context, establishmentID, insumoID string) error {
	const query = `SELECT id FROM insumos WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE`
	var id string
	if err := s.db.GetContext(ctx, &id, query, insumoID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInsumoNaoEncontrado
		}
		return fmt.Errorf("validar insumo: %w", err)
	}
	return nil
}
