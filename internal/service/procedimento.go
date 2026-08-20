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
	ErrInvalidProfessional         = errors.New("invalid_professional")
	ErrDonaNotActingAsProfessional = errors.New("dona_not_acting_as_professional")
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
	ProfissionalIDs    []string           `json:"profissional_ids"`
	Adicionais         []ServicoAdicional `json:"adicionais,omitempty"`
}

type ServicoAdicional struct {
	ID                      string  `db:"id" json:"id"`
	ServicoID               string  `db:"servico_id" json:"servico_id"`
	Nome                    string  `db:"nome" json:"nome"`
	PrecoAdicional          float64 `db:"preco_adicional" json:"preco_adicional"`
	DuracaoAdicionalMinutos int     `db:"duracao_adicional_minutos" json:"duracao_adicional_minutos"`
}

// CreateService cadastra um serviço base vinculado ao estabelecimento.
// profissionalIDs opcional: omitido/vazio = sem linhas em servico_profissionais (semântica “todos”).
func (s *ProcedimentoService) CreateService(
	ctx context.Context,
	establishmentID, nome string,
	precoBase float64,
	duracaoBase int,
	profissionalIDs []string,
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

	ids := dedupeIDs(profissionalIDs)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const insert = `
INSERT INTO servicos (estabelecimento_id, nome, preco_base, duracao_base_minutos, ativo)
VALUES ($1, $2, $3, $4, TRUE)
RETURNING id
`
	var id string
	if err := tx.GetContext(ctx, &id, insert, establishmentID, nome, precoBase, duracaoBase); err != nil {
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

// UpdateService atualiza campos do serviço e substitui o set de vínculos em servico_profissionais.
func (s *ProcedimentoService) UpdateService(
	ctx context.Context,
	establishmentID, serviceID, nome string,
	precoBase float64,
	duracaoBase int,
	ativo bool,
	profissionalIDs []string,
) error {
	nome = strings.TrimSpace(nome)
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return ErrServicoNaoEncontrado
	}
	if nome == "" {
		return fmt.Errorf("nome do serviço é obrigatório")
	}
	if precoBase < 0 {
		return fmt.Errorf("preço base inválido")
	}
	if duracaoBase <= 0 {
		return fmt.Errorf("duração base inválida")
	}

	ids := dedupeIDs(profissionalIDs)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
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
			return ErrServicoNaoEncontrado
		}
		return fmt.Errorf("bloquear serviço: %w", err)
	}

	const update = `
UPDATE servicos
SET nome = $3, preco_base = $4, duracao_base_minutos = $5, ativo = $6
WHERE id = $1 AND estabelecimento_id = $2
`
	if _, err := tx.ExecContext(ctx, update, serviceID, establishmentID, nome, precoBase, duracaoBase, ativo); err != nil {
		return fmt.Errorf("atualizar serviço: %w", err)
	}

	if err := s.replaceServicoProfissionais(ctx, tx, establishmentID, serviceID, ids); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar atualização: %w", err)
	}
	return nil
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

// replaceServicoProfissionais faz DELETE + INSERT atômico do set de vínculos do serviço.
func (s *ProcedimentoService) replaceServicoProfissionais(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID, serviceID string,
	profissionalIDs []string,
) error {
	if err := s.validarProfissionaisParaVinculo(ctx, tx, establishmentID, profissionalIDs); err != nil {
		return err
	}

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
	for _, profID := range profissionalIDs {
		if _, err := tx.ExecContext(ctx, insert, establishmentID, serviceID, profID); err != nil {
			return fmt.Errorf("gravar vínculo serviço-profissional: %w", err)
		}
	}
	return nil
}

func (s *ProcedimentoService) validarProfissionaisParaVinculo(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID string,
	profissionalIDs []string,
) error {
	if len(profissionalIDs) == 0 {
		return nil
	}

	const flagQuery = `
SELECT dona_atua_como_profissional
FROM estabelecimentos
WHERE id = $1
`
	var donaAtua bool
	if err := tx.GetContext(ctx, &donaAtua, flagQuery, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEstabelecimentoNaoEncontrado
		}
		return fmt.Errorf("consultar flag dona: %w", err)
	}

	const query = `
SELECT id, ativo, eh_dona
FROM profissionais
WHERE estabelecimento_id = $1 AND id = ANY($2)
`
	type row struct {
		ID     string `db:"id"`
		Ativo  bool   `db:"ativo"`
		EhDona bool   `db:"eh_dona"`
	}
	var rows []row
	if err := tx.SelectContext(ctx, &rows, query, establishmentID, pq.Array(profissionalIDs)); err != nil {
		return fmt.Errorf("validar profissionais: %w", err)
	}
	byID := make(map[string]row, len(rows))
	for _, r := range rows {
		byID[r.ID] = r
	}

	for _, id := range profissionalIDs {
		r, ok := byID[id]
		if !ok {
			return ErrInvalidProfessional
		}
		if !r.Ativo {
			return ErrInvalidProfessional
		}
		if r.EhDona && !donaAtua {
			return ErrDonaNotActingAsProfessional
		}
	}
	return nil
}

func dedupeIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
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

	const queryVinculos = `
SELECT servico_id, profissional_id
FROM servico_profissionais
WHERE estabelecimento_id = $1 AND servico_id = ANY($2)
ORDER BY profissional_id ASC
`
	type vinculo struct {
		ServicoID      string `db:"servico_id"`
		ProfissionalID string `db:"profissional_id"`
	}
	var vinculos []vinculo
	if err := s.db.SelectContext(ctx, &vinculos, queryVinculos, establishmentID, pq.Array(ids)); err != nil {
		return nil, fmt.Errorf("listar vínculos serviço-profissional: %w", err)
	}
	for _, v := range vinculos {
		if idx, ok := index[v.ServicoID]; ok {
			servicos[idx].ProfissionalIDs = append(servicos[idx].ProfissionalIDs, v.ProfissionalID)
		}
	}

	return servicos, nil
}
