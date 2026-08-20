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
	// ErrProfissionalVinculoInvalido: profissional inexistente, outro tenant ou inativo.
	ErrProfissionalVinculoInvalido = errors.New("invalid_professional")
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
	CategoriaID        *string            `db:"categoria_id" json:"categoria_id"`
	CategoriaNome      *string            `db:"categoria_nome" json:"categoria_nome,omitempty"`
	Adicionais         []ServicoAdicional `json:"adicionais"`
	ProfissionalIDs    []string           `json:"profissional_ids"`
}

type ServicoAdicional struct {
	ID                      string  `db:"id" json:"id"`
	ServicoID               string  `db:"servico_id" json:"servico_id"`
	Nome                    string  `db:"nome" json:"nome"`
	PrecoAdicional          float64 `db:"preco_adicional" json:"preco_adicional"`
	DuracaoAdicionalMinutos int     `db:"duracao_adicional_minutos" json:"duracao_adicional_minutos"`
}

// CreateService cadastra um serviço base vinculado ao estabelecimento.
// profissionalIDs opcional: omitido/vazio → sem vínculos em servico_profissionais.
// categoriaID vazio → sem vínculo de categoria (NULL).
func (s *ProcedimentoService) CreateService(
	ctx context.Context,
	establishmentID, nome string,
	precoBase float64,
	duracaoBase int,
	profissionalIDs []string,
	categoriaID string,
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

	ids, err := normalizeProfissionalIDs(profissionalIDs)
	if err != nil {
		return "", err
	}

	categoriaID = strings.TrimSpace(categoriaID)
	var categoriaArg any
	if categoriaID != "" {
		if err := s.validarCategoriaDoEstabelecimento(ctx, nil, establishmentID, categoriaID); err != nil {
			return "", err
		}
		categoriaArg = categoriaID
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := s.validarProfissionaisAtivosDoTenant(ctx, tx, establishmentID, ids); err != nil {
		return "", err
	}

	const insert = `
INSERT INTO servicos (estabelecimento_id, nome, preco_base, duracao_base_minutos, ativo, categoria_id)
VALUES ($1, $2, $3, $4, TRUE, $5)
RETURNING id
`
	var id string
	if err := tx.GetContext(ctx, &id, insert, establishmentID, nome, precoBase, duracaoBase, categoriaArg); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return "", ErrCategoriaServicoNaoEncontrada
		}
		return "", fmt.Errorf("cadastrar serviço: %w", err)
	}

	if err := s.replaceServicoProfissionais(ctx, tx, establishmentID, id, ids); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar transação: %w", err)
	}

	return id, nil
}

// UpdateServiceInput campos para PUT /api/v1/services/{id}.
// ProfissionalIDs nil = mantém vínculos; não-nil = replace atômico do set.
// CategoriaID nil = mantém; não-nil = trim e set (vazio → NULL).
type UpdateServiceInput struct {
	Nome            string
	PrecoBase       float64
	DuracaoBase     int
	Ativo           bool
	ProfissionalIDs *[]string
	CategoriaID     *string
}

// UpdateService atualiza serviço do tenant e, se solicitado, substitui vínculos.
func (s *ProcedimentoService) UpdateService(
	ctx context.Context,
	establishmentID, serviceID string,
	in UpdateServiceInput,
) (*Servico, error) {
	nome := strings.TrimSpace(in.Nome)
	if nome == "" {
		return nil, fmt.Errorf("nome do serviço é obrigatório")
	}
	if in.PrecoBase < 0 {
		return nil, fmt.Errorf("preço base inválido")
	}
	if in.DuracaoBase <= 0 {
		return nil, fmt.Errorf("duração base inválida")
	}

	var replaceIDs []string
	replaceLinks := false
	if in.ProfissionalIDs != nil {
		ids, err := normalizeProfissionalIDs(*in.ProfissionalIDs)
		if err != nil {
			return nil, err
		}
		replaceIDs = ids
		replaceLinks = true
	}

	updateCategoria := false
	var categoriaArg any
	if in.CategoriaID != nil {
		updateCategoria = true
		cat := strings.TrimSpace(*in.CategoriaID)
		if cat != "" {
			if err := s.validarCategoriaDoEstabelecimento(ctx, nil, establishmentID, cat); err != nil {
				return nil, err
			}
			categoriaArg = cat
		}
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lock = `
SELECT id FROM servicos
WHERE id = $1 AND estabelecimento_id = $2
FOR UPDATE
`
	var lockedID string
	if err := tx.GetContext(ctx, &lockedID, lock, serviceID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServicoNaoEncontrado
		}
		return nil, fmt.Errorf("bloquear serviço: %w", err)
	}

	if replaceLinks {
		if err := s.validarProfissionaisAtivosDoTenant(ctx, tx, establishmentID, replaceIDs); err != nil {
			return nil, err
		}
	}

	if updateCategoria {
		const update = `
UPDATE servicos
SET nome = $3, preco_base = $4, duracao_base_minutos = $5, ativo = $6, categoria_id = $7
WHERE id = $1 AND estabelecimento_id = $2
`
		if _, err := tx.ExecContext(
			ctx, update,
			serviceID, establishmentID, nome, in.PrecoBase, in.DuracaoBase, in.Ativo, categoriaArg,
		); err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
				return nil, ErrCategoriaServicoNaoEncontrada
			}
			return nil, fmt.Errorf("atualizar serviço: %w", err)
		}
	} else {
		const update = `
UPDATE servicos
SET nome = $3, preco_base = $4, duracao_base_minutos = $5, ativo = $6
WHERE id = $1 AND estabelecimento_id = $2
`
		if _, err := tx.ExecContext(
			ctx, update,
			serviceID, establishmentID, nome, in.PrecoBase, in.DuracaoBase, in.Ativo,
		); err != nil {
			return nil, fmt.Errorf("atualizar serviço: %w", err)
		}
	}

	if replaceLinks {
		if err := s.replaceServicoProfissionais(ctx, tx, establishmentID, serviceID, replaceIDs); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar atualização: %w", err)
	}

	return s.BuscarServicoPorID(ctx, establishmentID, serviceID)
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

func (s *ProcedimentoService) validarCategoriaDoEstabelecimento(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID, categoriaID string,
) error {
	const query = `SELECT id FROM categorias_servicos WHERE id = $1 AND estabelecimento_id = $2`
	var id string
	var err error
	if tx != nil {
		err = tx.GetContext(ctx, &id, query, categoriaID, establishmentID)
	} else {
		err = s.db.GetContext(ctx, &id, query, categoriaID, establishmentID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCategoriaServicoNaoEncontrada
		}
		return fmt.Errorf("validar categoria: %w", err)
	}
	return nil
}

func normalizeProfissionalIDs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, id := range raw {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, ErrProfissionalVinculoInvalido
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (s *ProcedimentoService) validarProfissionaisAtivosDoTenant(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID string,
	profissionalIDs []string,
) error {
	if len(profissionalIDs) == 0 {
		return nil
	}
	const query = `
SELECT id FROM profissionais
WHERE estabelecimento_id = $1 AND id = ANY($2) AND ativo = TRUE
`
	var found []string
	if err := tx.SelectContext(ctx, &found, query, establishmentID, pq.Array(profissionalIDs)); err != nil {
		return fmt.Errorf("validar profissionais: %w", err)
	}
	if len(found) != len(profissionalIDs) {
		return ErrProfissionalVinculoInvalido
	}
	return nil
}

func (s *ProcedimentoService) replaceServicoProfissionais(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID, serviceID string,
	profissionalIDs []string,
) error {
	const del = `
DELETE FROM servico_profissionais
WHERE estabelecimento_id = $1 AND servico_id = $2
`
	if _, err := tx.ExecContext(ctx, del, establishmentID, serviceID); err != nil {
		return fmt.Errorf("limpar vínculos serviço-profissional: %w", err)
	}
	if len(profissionalIDs) == 0 {
		return nil
	}
	const insert = `
INSERT INTO servico_profissionais (estabelecimento_id, servico_id, profissional_id)
VALUES ($1, $2, $3)
`
	for _, pid := range profissionalIDs {
		if _, err := tx.ExecContext(ctx, insert, establishmentID, serviceID, pid); err != nil {
			return fmt.Errorf("vincular profissional ao serviço: %w", err)
		}
	}
	return nil
}

// ListServices retorna serviços, adicionais e profissional_ids isolados por estabelecimento.
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
		servicos[i].ProfissionalIDs = []string{}
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

	if err := s.attachProfissionalIDs(ctx, establishmentID, servicos); err != nil {
		return nil, err
	}

	return servicos, nil
}

func (s *ProcedimentoService) attachProfissionalIDs(
	ctx context.Context,
	establishmentID string,
	servicos []Servico,
) error {
	if len(servicos) == 0 {
		return nil
	}
	ids := make([]string, len(servicos))
	index := make(map[string]int, len(servicos))
	for i := range servicos {
		ids[i] = servicos[i].ID
		index[servicos[i].ID] = i
	}

	const query = `
SELECT servico_id, profissional_id
FROM servico_profissionais
WHERE estabelecimento_id = $1 AND servico_id = ANY($2)
ORDER BY profissional_id ASC
`
	type row struct {
		ServicoID      string `db:"servico_id"`
		ProfissionalID string `db:"profissional_id"`
	}
	var rows []row
	if err := s.db.SelectContext(ctx, &rows, query, establishmentID, pq.Array(ids)); err != nil {
		return fmt.Errorf("listar vínculos serviço-profissional: %w", err)
	}
	for _, r := range rows {
		if idx, ok := index[r.ServicoID]; ok {
			servicos[idx].ProfissionalIDs = append(servicos[idx].ProfissionalIDs, r.ProfissionalID)
		}
	}
	return nil
}
