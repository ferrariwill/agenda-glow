package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jmoiron/sqlx"
)

var (
	ErrInsumoNaoEncontrado = errors.New("insumo não encontrado")
	ErrInsumoPayload       = errors.New("payload de insumo inválido")
)

var unidadesInsumoPermitidas = map[string]struct{}{
	"ml": {},
	"g":  {},
	"un": {},
	"kg": {},
	"oz": {},
}

type InsumoService struct {
	db *sqlx.DB
}

func NewInsumoService(db *sqlx.DB) *InsumoService {
	return &InsumoService{db: db}
}

// Insumo is the canonical stock row. Quantidade is always in the base unit
// (ml/g/un/…). QuantidadeEmbalagens is derived for API responses.
type Insumo struct {
	ID                   string  `db:"id" json:"id"`
	EstabelecimentoID    string  `db:"estabelecimento_id" json:"tenant_id"`
	Nome                 string  `db:"nome" json:"nome"`
	Marca                *string `db:"marca" json:"marca,omitempty"`
	Categoria            string  `db:"categoria" json:"categoria"`
	Quantidade           float64 `db:"quantidade" json:"quantidade"`
	ConteudoPorEmbalagem float64 `db:"conteudo_por_embalagem" json:"conteudo_por_embalagem"`
	QuantidadeEmbalagens float64 `db:"-" json:"quantidade_embalagens"`
	EstoqueMinimo        float64 `db:"estoque_minimo" json:"estoque_minimo"`
	EstoqueIdeal         float64 `db:"estoque_ideal" json:"estoque_ideal"`
	ValorUnitario        float64 `db:"valor_unitario" json:"valor_unitario"` // custo por 1 unidade base
	Unidade              string  `db:"unidade" json:"unidade"`
	ImagemURL            *string `db:"imagem_url" json:"imagem_url,omitempty"`
	InstrucoesUso        *string `db:"instrucoes_uso" json:"instrucoes_uso,omitempty"`
	Ativo                bool    `db:"ativo" json:"ativo"`
}

// InsumoInput is the create/update payload. Prefer QuantidadeEmbalagens when set;
// otherwise use Quantidade (compat). If neither is set, stock starts at 0.
type InsumoInput struct {
	Nome                 string
	Marca                string
	Categoria            string
	Unidade              string
	ConteudoPorEmbalagem float64
	Quantidade           *float64
	QuantidadeEmbalagens *float64
	EstoqueMinimo        float64
	EstoqueIdeal         float64
	ValorUnitario        float64
	ImagemURL            string
	InstrucoesUso        string
	Ativo                bool
}

// AjusteEstoqueInput adjusts stock in base units. Prefer DeltaEmbalagens when set.
type AjusteEstoqueInput struct {
	Delta           *float64
	DeltaEmbalagens *float64
}

const insumoSelectCols = `
id, estabelecimento_id, nome, marca, categoria, quantidade, conteudo_por_embalagem,
estoque_minimo, estoque_ideal, valor_unitario, unidade, imagem_url, instrucoes_uso, ativo
`

func (s *InsumoService) List(ctx context.Context, establishmentID string) ([]Insumo, error) {
	query := `
SELECT ` + insumoSelectCols + `
FROM insumos
WHERE estabelecimento_id = $1 AND ativo = TRUE
ORDER BY nome
`
	var list []Insumo
	if err := s.db.SelectContext(ctx, &list, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar insumos: %w", err)
	}
	for i := range list {
		list[i].applyDerived()
	}
	return list, nil
}

func (s *InsumoService) Create(ctx context.Context, establishmentID string, in InsumoInput) (string, error) {
	nome, categoria, unidade, conteudo, quantidade, estoqueMin, estoqueIdeal, valor, err := normalizeInsumoInput(in)
	if err != nil {
		return "", err
	}
	const insert = `
INSERT INTO insumos (
    estabelecimento_id, nome, marca, categoria, quantidade, conteudo_por_embalagem,
    estoque_minimo, estoque_ideal, valor_unitario, unidade, imagem_url, instrucoes_uso, ativo
) VALUES ($1, $2, NULLIF($3,''), $4, $5, $6, $7, $8, $9, $10, NULLIF($11,''), NULLIF($12,''), TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert,
		establishmentID, nome, in.Marca, categoria, quantidade, conteudo,
		estoqueMin, estoqueIdeal, valor, unidade, in.ImagemURL, in.InstrucoesUso,
	); err != nil {
		return "", fmt.Errorf("criar insumo: %w", err)
	}
	return id, nil
}

func (s *InsumoService) Update(ctx context.Context, establishmentID, id string, in InsumoInput) error {
	nome, categoria, unidade, conteudo, quantidade, estoqueMin, estoqueIdeal, valor, err := normalizeInsumoInput(in)
	if err != nil {
		return err
	}
	const q = `
UPDATE insumos SET
    nome = $3, marca = NULLIF($4,''), categoria = $5, quantidade = $6,
    conteudo_por_embalagem = $7, estoque_minimo = $8, estoque_ideal = $9,
    valor_unitario = $10, unidade = $11, imagem_url = NULLIF($12,''),
    instrucoes_uso = NULLIF($13,''), ativo = $14
WHERE id = $1 AND estabelecimento_id = $2
`
	res, err := s.db.ExecContext(ctx, q,
		id, establishmentID, nome, in.Marca, categoria, quantidade, conteudo,
		estoqueMin, estoqueIdeal, valor, unidade, in.ImagemURL, in.InstrucoesUso, in.Ativo,
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

func (s *InsumoService) AjustarEstoque(ctx context.Context, establishmentID, id string, in AjusteEstoqueInput) error {
	conteudo := 0.0
	if in.DeltaEmbalagens != nil {
		item, err := s.GetByID(ctx, establishmentID, id)
		if err != nil {
			return err
		}
		conteudo = item.ConteudoPorEmbalagem
	}
	delta, err := resolveAjusteDelta(in, conteudo)
	if err != nil {
		return err
	}
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
	query := `
SELECT ` + insumoSelectCols + `
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
	item.applyDerived()
	return &item, nil
}

func (i *Insumo) applyDerived() {
	i.QuantidadeEmbalagens = calcularQuantidadeEmbalagens(i.Quantidade, i.ConteudoPorEmbalagem)
}

func normalizeInsumoInput(in InsumoInput) (nome, categoria, unidade string, conteudo, quantidade, estoqueMin, estoqueIdeal, valor float64, err error) {
	nome = strings.TrimSpace(in.Nome)
	if nome == "" {
		return "", "", "", 0, 0, 0, 0, 0, fmt.Errorf("%w: nome do insumo é obrigatório", ErrInsumoPayload)
	}
	categoria = strings.TrimSpace(in.Categoria)
	if categoria == "" {
		categoria = "GERAL"
	}
	unidade, err = normalizeUnidadeInsumo(in.Unidade)
	if err != nil {
		return "", "", "", 0, 0, 0, 0, 0, err
	}
	conteudo = in.ConteudoPorEmbalagem
	if conteudo == 0 {
		conteudo = 1
	}
	if conteudo <= 0 {
		return "", "", "", 0, 0, 0, 0, 0, fmt.Errorf("%w: conteudo_por_embalagem deve ser > 0", ErrInsumoPayload)
	}
	quantidade, err = resolverQuantidadeBase(in.Quantidade, in.QuantidadeEmbalagens, conteudo)
	if err != nil {
		return "", "", "", 0, 0, 0, 0, 0, err
	}
	if in.EstoqueMinimo < 0 || in.EstoqueIdeal < 0 || in.ValorUnitario < 0 {
		return "", "", "", 0, 0, 0, 0, 0, fmt.Errorf("%w: estoque_minimo, estoque_ideal e valor_unitario devem ser >= 0", ErrInsumoPayload)
	}
	return nome, categoria, unidade, conteudo, quantidade, in.EstoqueMinimo, in.EstoqueIdeal, in.ValorUnitario, nil
}

func normalizeUnidadeInsumo(unidade string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(unidade))
	if u == "" {
		return "un", nil
	}
	if _, ok := unidadesInsumoPermitidas[u]; !ok {
		return "", fmt.Errorf("%w: unidade inválida %q", ErrInsumoPayload, unidade)
	}
	return u, nil
}

// resolverQuantidadeBase prefers quantidade_embalagens when present (Mode A),
// else quantidade (Mode B). If both are set, embalagens wins. If neither, 0.
func resolverQuantidadeBase(quantidade, quantidadeEmbalagens *float64, conteudoPorEmbalagem float64) (float64, error) {
	if quantidadeEmbalagens != nil {
		if *quantidadeEmbalagens < 0 {
			return 0, fmt.Errorf("%w: quantidade_embalagens deve ser >= 0", ErrInsumoPayload)
		}
		return round3(*quantidadeEmbalagens * conteudoPorEmbalagem), nil
	}
	if quantidade != nil {
		if *quantidade < 0 {
			return 0, fmt.Errorf("%w: quantidade deve ser >= 0", ErrInsumoPayload)
		}
		return *quantidade, nil
	}
	return 0, nil
}

func resolveAjusteDelta(in AjusteEstoqueInput, conteudoPorEmbalagem float64) (float64, error) {
	if in.DeltaEmbalagens != nil {
		if conteudoPorEmbalagem <= 0 {
			return 0, fmt.Errorf("%w: conteudo_por_embalagem inválido para ajuste", ErrInsumoPayload)
		}
		return round3(*in.DeltaEmbalagens * conteudoPorEmbalagem), nil
	}
	if in.Delta != nil {
		return *in.Delta, nil
	}
	return 0, fmt.Errorf("%w: informe delta ou delta_embalagens", ErrInsumoPayload)
}

func calcularQuantidadeEmbalagens(quantidade, conteudoPorEmbalagem float64) float64 {
	if conteudoPorEmbalagem <= 0 {
		return 0
	}
	return round3(quantidade / conteudoPorEmbalagem)
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
