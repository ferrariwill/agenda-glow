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
	ErrServicoInsumoInvalido      = errors.New("ficha técnica inválida")
	ErrInsumoBOMNaoEncontrado     = errors.New("insumo da ficha técnica não encontrado ou inativo")
	ErrServicoInsumoNaoEncontrado = errors.New("vínculo serviço-insumo não encontrado")
	ErrServicoInsumoDuplicado     = errors.New("insumo já vinculado a este serviço")
)

type ServicoInsumoService struct {
	db *sqlx.DB
}

func NewServicoInsumoService(db *sqlx.DB) *ServicoInsumoService {
	return &ServicoInsumoService{db: db}
}

// ServicoInsumoItem é um vínculo da ficha técnica (BOM) serviço ↔ insumo (shape Replace/forecast).
type ServicoInsumoItem struct {
	InsumoID      string  `db:"insumo_id" json:"insumo_id"`
	Nome          string  `db:"nome" json:"nome"`
	Unidade       string  `db:"unidade" json:"unidade"`
	QuantidadeUso float64 `db:"quantidade_uso" json:"quantidade_uso"`
}

// ServicoInsumo é o vínculo com id (contrato item CRUD DEV-216).
type ServicoInsumo struct {
	ID            string  `db:"id" json:"id"`
	ServicoID     string  `db:"servico_id" json:"servico_id"`
	InsumoID      string  `db:"insumo_id" json:"insumo_id"`
	InsumoNome    string  `db:"insumo_nome" json:"insumo_nome"`
	InsumoUnidade string  `db:"insumo_unidade" json:"insumo_unidade"`
	QuantidadeUso float64 `db:"quantidade_uso" json:"quantidade_uso"`
}

// ServicoInsumoInput é o payload de um item ao substituir a ficha.
type ServicoInsumoInput struct {
	InsumoID      string  `json:"insumo_id"`
	QuantidadeUso float64 `json:"quantidade_uso"`
}

// List retorna a ficha técnica do serviço no tenant (com id do vínculo).
func (s *ServicoInsumoService) List(ctx context.Context, establishmentID, servicoID string) ([]ServicoInsumo, error) {
	if err := s.requireServicoAtivo(ctx, establishmentID, servicoID); err != nil {
		return nil, err
	}

	const query = `
SELECT si.id, si.servico_id, si.insumo_id, i.nome AS insumo_nome, i.unidade AS insumo_unidade,
       si.quantidade_uso
FROM servico_insumos si
INNER JOIN insumos i
    ON i.id = si.insumo_id
   AND i.estabelecimento_id = si.estabelecimento_id
WHERE si.estabelecimento_id = $1
  AND si.servico_id = $2
  AND i.ativo = TRUE
ORDER BY i.nome
`
	var list []ServicoInsumo
	if err := s.db.SelectContext(ctx, &list, query, establishmentID, servicoID); err != nil {
		return nil, fmt.Errorf("listar ficha técnica: %w", err)
	}
	if list == nil {
		list = []ServicoInsumo{}
	}
	return list, nil
}

func (s *ServicoInsumoService) Create(
	ctx context.Context,
	establishmentID, serviceID, insumoID string,
	quantidadeUso float64,
) (string, error) {
	insumoID = strings.TrimSpace(insumoID)
	if insumoID == "" || quantidadeUso <= 0 {
		return "", fmt.Errorf("%w: insumo_id e quantidade_uso válida são obrigatórios", ErrServicoInsumoInvalido)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := s.validarServicoTx(ctx, tx, establishmentID, serviceID); err != nil {
		return "", err
	}
	if err := s.validarInsumoTx(ctx, tx, establishmentID, insumoID); err != nil {
		return "", err
	}

	var id string
	if err := tx.GetContext(ctx, &id, `
INSERT INTO servico_insumos (estabelecimento_id, servico_id, insumo_id, quantidade_uso)
VALUES ($1, $2, $3, $4)
RETURNING id
`, establishmentID, serviceID, insumoID, roundQty(quantidadeUso)); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", ErrServicoInsumoDuplicado
		}
		return "", fmt.Errorf("criar vínculo serviço-insumo: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar transação: %w", err)
	}
	return id, nil
}

func (s *ServicoInsumoService) Update(
	ctx context.Context, establishmentID, serviceID, linkID string, quantidadeUso float64,
) error {
	if quantidadeUso <= 0 {
		return fmt.Errorf("%w: quantidade_uso deve ser > 0", ErrServicoInsumoInvalido)
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE servico_insumos SET quantidade_uso = $4
WHERE id = $1 AND servico_id = $2 AND estabelecimento_id = $3
`, linkID, serviceID, establishmentID, roundQty(quantidadeUso))
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

func (s *ServicoInsumoService) Delete(ctx context.Context, establishmentID, serviceID, linkID string) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM servico_insumos
WHERE id = $1 AND servico_id = $2 AND estabelecimento_id = $3
`, linkID, serviceID, establishmentID)
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

// Replace substitui a ficha técnica completa do serviço (transação delete+insert).
func (s *ServicoInsumoService) Replace(
	ctx context.Context,
	establishmentID, servicoID string,
	itens []ServicoInsumoInput,
) error {
	if err := s.requireServicoAtivo(ctx, establishmentID, servicoID); err != nil {
		return err
	}
	if err := validateBOMItens(itens); err != nil {
		return err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação ficha técnica: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM servico_insumos WHERE estabelecimento_id = $1 AND servico_id = $2`,
		establishmentID, servicoID,
	); err != nil {
		return fmt.Errorf("limpar ficha técnica: %w", err)
	}

	const insert = `
INSERT INTO servico_insumos (estabelecimento_id, servico_id, insumo_id, quantidade_uso)
SELECT $1, $2, i.id, $3
FROM insumos i
WHERE i.id = $4
  AND i.estabelecimento_id = $1
  AND i.ativo = TRUE
`
	for _, item := range itens {
		res, err := tx.ExecContext(ctx, insert,
			establishmentID, servicoID, roundQty(item.QuantidadeUso), strings.TrimSpace(item.InsumoID),
		)
		if err != nil {
			return fmt.Errorf("inserir vínculo ficha técnica: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrInsumoBOMNaoEncontrado
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ficha técnica: %w", err)
	}
	return nil
}

func (s *ServicoInsumoService) requireServicoAtivo(ctx context.Context, establishmentID, servicoID string) error {
	const q = `
SELECT 1
FROM servicos
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var one int
	if err := s.db.GetContext(ctx, &one, q, servicoID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServicoNaoEncontrado
		}
		return fmt.Errorf("validar serviço: %w", err)
	}
	return nil
}

func (s *ServicoInsumoService) validarServicoTx(ctx context.Context, tx *sqlx.Tx, establishmentID, serviceID string) error {
	const query = `SELECT id FROM servicos WHERE id = $1 AND estabelecimento_id = $2 FOR UPDATE`
	var id string
	if err := tx.GetContext(ctx, &id, query, serviceID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrServicoNaoEncontrado
		}
		return fmt.Errorf("validar serviço: %w", err)
	}
	return nil
}

func (s *ServicoInsumoService) validarInsumoTx(ctx context.Context, tx *sqlx.Tx, establishmentID, insumoID string) error {
	const query = `
SELECT id FROM insumos
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
FOR UPDATE
`
	var id string
	if err := tx.GetContext(ctx, &id, query, insumoID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInsumoNaoEncontrado
		}
		return fmt.Errorf("validar insumo: %w", err)
	}
	return nil
}

func validateBOMItens(itens []ServicoInsumoInput) error {
	seen := make(map[string]struct{}, len(itens))
	for _, item := range itens {
		id := strings.TrimSpace(item.InsumoID)
		if id == "" {
			return fmt.Errorf("%w: insumo_id obrigatório", ErrServicoInsumoInvalido)
		}
		if item.QuantidadeUso <= 0 {
			return fmt.Errorf("%w: quantidade_uso deve ser > 0", ErrServicoInsumoInvalido)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%w: insumo_id duplicado", ErrServicoInsumoInvalido)
		}
		seen[id] = struct{}{}
	}
	return nil
}
