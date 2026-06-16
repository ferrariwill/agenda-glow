package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrEstabelecimentoNaoEncontrado = errors.New("estabelecimento não encontrado")
	ErrSlugInvalido                 = errors.New("slug inválido")
	ErrSlugEmUso                    = errors.New("slug já está em uso")
	ErrSlugAlreadyExists            = errors.New("slug_already_exists")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type EstabelecimentoService struct {
	db *sqlx.DB
}

func NewEstabelecimentoService(db *sqlx.DB) *EstabelecimentoService {
	return &EstabelecimentoService{db: db}
}

type Estabelecimento struct {
	ID            string  `db:"id" json:"id"`
	NomeComercial string  `db:"nome_comercial" json:"nome_comercial"`
	Slug          string  `db:"slug" json:"slug"`
	LogoURL       *string `db:"logo_url" json:"logo_url,omitempty"`
}

type ConfigEstabelecimentoInput struct {
	EstabelecimentoID string
	NomeComercial     string
	Slug              string
	LogoURL           *string // nil = mantém logo atual
}

type CreateEstablishmentInput struct {
	NomeComercial string
	Slug          string
}

type EstablishmentAdminView struct {
	ID            string    `db:"id" json:"id"`
	NomeComercial string    `db:"nome_comercial" json:"nome_comercial"`
	Slug          string    `db:"slug" json:"slug"`
	Ativo         bool      `db:"ativo" json:"ativo"`
	DataCadastro  time.Time `db:"data_cadastro" json:"data_cadastro"`
}

type ProfissionalCatalogo struct {
	ID            string `db:"id" json:"id"`
	Nome          string `db:"nome" json:"nome"`
	Especialidade string `db:"especialidade" json:"especialidade"`
}

type ServicoAdicionalCatalogo struct {
	ID                        string  `db:"id" json:"id"`
	Nome                      string  `db:"nome" json:"nome"`
	PrecoAdicional            float64 `db:"preco_adicional" json:"preco_adicional"`
	DuracaoAdicionalMinutos   int     `db:"duracao_adicional_minutos" json:"duracao_adicional_minutos"`
}

type ServicoCatalogo struct {
	ID                 string                     `db:"id" json:"id"`
	Nome               string                     `db:"nome" json:"nome"`
	PrecoBase          float64                    `db:"preco_base" json:"preco_base"`
	DuracaoBaseMinutos int                        `db:"duracao_base_minutos" json:"duracao_base_minutos"`
	Adicionais         []ServicoAdicionalCatalogo `json:"adicionais"`
}

type CatalogoAutoatendimento struct {
	Estabelecimento Estabelecimento        `json:"estabelecimento"`
	Profissionais   []ProfissionalCatalogo `json:"profissionais"`
	Servicos        []ServicoCatalogo      `json:"servicos"`
}

func ValidarSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return ErrSlugInvalido
	}
	return nil
}

func resolverSlugFinal(nome, suggestedSlug string) (string, error) {
	candidato := strings.TrimSpace(suggestedSlug)
	if candidato == "" {
		candidato = nome
	}

	slug := NormalizarSlug(candidato)
	if slug == "" {
		return "", ErrSlugInvalido
	}

	if err := ValidarSlug(slug); err != nil {
		return "", err
	}

	return slug, nil
}

// RegisterEstablishment cadastra um novo estabelecimento com slug normalizado.
func (s *EstabelecimentoService) RegisterEstablishment(
	ctx context.Context,
	name, suggestedSlug string,
) (id string, slugFinal string, err error) {
	nome := strings.TrimSpace(name)
	if nome == "" {
		return "", "", fmt.Errorf("nome comercial é obrigatório")
	}

	slug, err := resolverSlugFinal(nome, suggestedSlug)
	if err != nil {
		return "", "", err
	}

	const verificarSlug = `SELECT id FROM estabelecimentos WHERE slug = $1 LIMIT 1`
	var existente string
	err = s.db.GetContext(ctx, &existente, verificarSlug, slug)
	if err == nil {
		return "", "", ErrSlugAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("verificar slug: %w", err)
	}

	const insert = `
INSERT INTO estabelecimentos (nome_comercial, slug, ativo)
VALUES ($1, $2, TRUE)
RETURNING id
`
	if err := s.db.GetContext(ctx, &id, insert, nome, slug); err != nil {
		return "", "", fmt.Errorf("cadastrar estabelecimento: %w", err)
	}

	return id, slug, nil
}

// ListAllEstablishments retorna todos os estabelecimentos para o painel Super Admin.
func (s *EstabelecimentoService) ListAllEstablishments(ctx context.Context) ([]EstablishmentAdminView, error) {
	const query = `
SELECT id, nome_comercial, slug, ativo, data_cadastro
FROM estabelecimentos
ORDER BY data_cadastro DESC
`
	var lista []EstablishmentAdminView
	if err := s.db.SelectContext(ctx, &lista, query); err != nil {
		return nil, fmt.Errorf("listar estabelecimentos: %w", err)
	}
	if lista == nil {
		lista = []EstablishmentAdminView{}
	}
	return lista, nil
}

// ToggleEstablishmentStatus ativa ou suspende um estabelecimento.
func (s *EstabelecimentoService) ToggleEstablishmentStatus(ctx context.Context, id string, active bool) error {
	const update = `
UPDATE estabelecimentos
SET ativo = $2
WHERE id = $1
`
	result, err := s.db.ExecContext(ctx, update, id, active)
	if err != nil {
		return fmt.Errorf("atualizar status do estabelecimento: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar linhas afetadas: %w", err)
	}
	if rows == 0 {
		return ErrEstabelecimentoNaoEncontrado
	}

	return nil
}

// BuscarPorSlug localiza o estabelecimento ativo pelo slug da URL pública.
func (s *EstabelecimentoService) BuscarPorSlug(ctx context.Context, slug string) (*Estabelecimento, error) {
	if err := ValidarSlug(slug); err != nil {
		return nil, err
	}

	const query = `
SELECT id, nome_comercial, slug, logo_url
FROM estabelecimentos
WHERE slug = $1 AND ativo = TRUE
`
	var est Estabelecimento
	if err := s.db.GetContext(ctx, &est, query, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar estabelecimento por slug: %w", err)
	}

	return &est, nil
}

// BuscarPorID retorna dados básicos do estabelecimento pelo identificador interno.
func (s *EstabelecimentoService) BuscarPorID(ctx context.Context, id string) (*Estabelecimento, error) {
	const query = `
SELECT id, nome_comercial, slug, logo_url
FROM estabelecimentos
WHERE id = $1
`
	var est Estabelecimento
	if err := s.db.GetContext(ctx, &est, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar estabelecimento por id: %w", err)
	}
	return &est, nil
}

// BuscarCatalogoAutoatendimento retorna profissionais e serviços isolados por estabelecimento.
func (s *EstabelecimentoService) BuscarCatalogoAutoatendimento(
	ctx context.Context,
	estabelecimentoID string,
) (*CatalogoAutoatendimento, error) {
	const queryEstabelecimento = `
SELECT id, nome_comercial, slug, logo_url
FROM estabelecimentos
WHERE id = $1 AND ativo = TRUE
`
	var est Estabelecimento
	if err := s.db.GetContext(ctx, &est, queryEstabelecimento, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar estabelecimento: %w", err)
	}

	const queryProfissionais = `
SELECT id, nome, especialidade
FROM profissionais
WHERE estabelecimento_id = $1 AND ativo = TRUE
ORDER BY nome
`
	var profissionais []ProfissionalCatalogo
	if err := s.db.SelectContext(ctx, &profissionais, queryProfissionais, estabelecimentoID); err != nil {
		return nil, fmt.Errorf("listar profissionais: %w", err)
	}

	const queryServicos = `
SELECT id, nome, preco_base, duracao_base_minutos
FROM servicos
WHERE estabelecimento_id = $1 AND ativo = TRUE
ORDER BY nome
`
	var servicos []ServicoCatalogo
	if err := s.db.SelectContext(ctx, &servicos, queryServicos, estabelecimentoID); err != nil {
		return nil, fmt.Errorf("listar serviços: %w", err)
	}

	if len(servicos) > 0 {
		servicoIDs := make([]string, len(servicos))
		servicoIndex := make(map[string]int, len(servicos))
		for i, serv := range servicos {
			servicoIDs[i] = serv.ID
			servicoIndex[serv.ID] = i
			servicos[i].Adicionais = []ServicoAdicionalCatalogo{}
		}

		const queryAdicionais = `
SELECT sa.id, sa.servico_id, sa.nome, sa.preco_adicional, sa.duracao_adicional_minutos
FROM servico_adicionais sa
INNER JOIN servicos s ON s.id = sa.servico_id
WHERE s.estabelecimento_id = $1
  AND sa.servico_id = ANY($2)
ORDER BY sa.nome
`
		type adicionalRow struct {
			ID                      string  `db:"id"`
			ServicoID               string  `db:"servico_id"`
			Nome                    string  `db:"nome"`
			PrecoAdicional          float64 `db:"preco_adicional"`
			DuracaoAdicionalMinutos int     `db:"duracao_adicional_minutos"`
		}

		var adicionais []adicionalRow
		if err := s.db.SelectContext(ctx, &adicionais, queryAdicionais, estabelecimentoID, pq.Array(servicoIDs)); err != nil {
			return nil, fmt.Errorf("listar adicionais: %w", err)
		}

		for _, ad := range adicionais {
			idx, ok := servicoIndex[ad.ServicoID]
			if !ok {
				continue
			}
			servicos[idx].Adicionais = append(servicos[idx].Adicionais, ServicoAdicionalCatalogo{
				ID:                      ad.ID,
				Nome:                    ad.Nome,
				PrecoAdicional:          ad.PrecoAdicional,
				DuracaoAdicionalMinutos: ad.DuracaoAdicionalMinutos,
			})
		}
	}

	if profissionais == nil {
		profissionais = []ProfissionalCatalogo{}
	}
	if servicos == nil {
		servicos = []ServicoCatalogo{}
	}

	return &CatalogoAutoatendimento{
		Estabelecimento: est,
		Profissionais:   profissionais,
		Servicos:        servicos,
	}, nil
}

// AtualizarConfig persiste nome, slug e logo do estabelecimento em transação segura.
func (s *EstabelecimentoService) AtualizarConfig(
	ctx context.Context,
	input ConfigEstabelecimentoInput,
) (*Estabelecimento, error) {
	if err := ValidarSlug(input.Slug); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lockEstabelecimento = `
SELECT id FROM estabelecimentos WHERE id = $1 AND ativo = TRUE FOR UPDATE
`
	var id string
	if err := tx.GetContext(ctx, &id, lockEstabelecimento, input.EstabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("bloquear estabelecimento: %w", err)
	}

	const verificarSlug = `
SELECT id FROM estabelecimentos WHERE slug = $1 AND id <> $2 LIMIT 1
`
	var outroID string
	err = tx.GetContext(ctx, &outroID, verificarSlug, input.Slug, input.EstabelecimentoID)
	if err == nil {
		return nil, ErrSlugEmUso
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("verificar slug: %w", err)
	}

	var atualizado Estabelecimento
	if input.LogoURL != nil {
		const updateComLogo = `
UPDATE estabelecimentos
SET nome_comercial = $2, slug = $3, logo_url = $4
WHERE id = $1
RETURNING id, nome_comercial, slug, logo_url
`
		if err := tx.GetContext(ctx, &atualizado, updateComLogo,
			input.EstabelecimentoID, input.NomeComercial, input.Slug, *input.LogoURL,
		); err != nil {
			return nil, fmt.Errorf("atualizar estabelecimento: %w", err)
		}
	} else {
		const updateSemLogo = `
UPDATE estabelecimentos
SET nome_comercial = $2, slug = $3
WHERE id = $1
RETURNING id, nome_comercial, slug, logo_url
`
		if err := tx.GetContext(ctx, &atualizado, updateSemLogo,
			input.EstabelecimentoID, input.NomeComercial, input.Slug,
		); err != nil {
			return nil, fmt.Errorf("atualizar estabelecimento: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmar transação: %w", err)
	}

	return &atualizado, nil
}
