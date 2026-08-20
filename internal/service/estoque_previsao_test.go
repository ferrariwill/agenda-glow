package service

import (
	"database/sql"
	"errors"
	"testing"
)

func TestClassificarNivelEstoque(t *testing.T) {
	t.Parallel()

	dias5 := 5.0
	dias10 := 10.0

	casos := []struct {
		nome                         string
		atual, min, ideal, projetado float64
		dias                         *float64
		horizonte                    int
		esperado                     string
	}{
		{
			nome: "critico por estoque atual no minimo",
			atual: 50, min: 50, ideal: 200, projetado: 100,
			horizonte: 7, esperado: NivelCritico,
		},
		{
			nome: "critico por projetado abaixo do minimo",
			atual: 120, min: 50, ideal: 200, projetado: 34.5,
			horizonte: 7, esperado: NivelCritico,
		},
		{
			nome: "critico por dias ate critico no horizonte",
			atual: 120, min: 50, ideal: 200, projetado: 100,
			dias: &dias5, horizonte: 7, esperado: NivelCritico,
		},
		{
			nome: "atencao quando projetado <= 50% do ideal",
			atual: 200, min: 50, ideal: 200, projetado: 90,
			dias: &dias10, horizonte: 7, esperado: NivelAtencao,
		},
		{
			nome: "ok quando acima dos limiares",
			atual: 200, min: 50, ideal: 200, projetado: 180,
			dias: &dias10, horizonte: 7, esperado: NivelOK,
		},
		{
			nome: "sem media diaria nao usa dias",
			atual: 200, min: 50, ideal: 200, projetado: 180,
			dias: nil, horizonte: 7, esperado: NivelOK,
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nome, func(t *testing.T) {
			t.Parallel()
			got := ClassificarNivelEstoque(caso.atual, caso.min, caso.ideal, caso.projetado, caso.dias, caso.horizonte)
			if got != caso.esperado {
				t.Fatalf("nivel = %s; esperado %s", got, caso.esperado)
			}
		})
	}
}

func TestCalcularQuantidadeSugeridaCompra(t *testing.T) {
	t.Parallel()

	if got := CalcularQuantidadeSugeridaCompra(200, 34.5); got != 165.5 {
		t.Fatalf("quantidade sugerida = %.3f; esperado 165.500", got)
	}
	if got := CalcularQuantidadeSugeridaCompra(100, 150); got != 0 {
		t.Fatalf("quando projetado > ideal, esperado 0; got %.3f", got)
	}
}

func TestBuildPrevisaoItemMedias(t *testing.T) {
	t.Parallel()

	row := previsaoAggRow{
		InsumoID:              "i1",
		Nome:                  "Tintura 7.0",
		Unidade:               "ml",
		QuantidadeAtual:       120,
		EstoqueMinimo:         50,
		EstoqueIdeal:          200,
		ValorUnitario:         0.85,
		ConsumoHistoricoTotal: sql.NullFloat64{Float64: 1140, Valid: true},
		AtendimentosHistorico: sql.NullInt64{Int64: 40, Valid: true},
		DemandaFutura:         sql.NullFloat64{Float64: 85.5, Valid: true},
		AgendamentosFuturos:   sql.NullInt64{Int64: 3, Valid: true},
	}

	item := buildPrevisaoItem(row, 30, 7)
	if item.MediaConsumoPorAtendimento != 28.5 {
		t.Fatalf("media por atendimento = %.3f; esperado 28.500", item.MediaConsumoPorAtendimento)
	}
	if item.MediaConsumoDiaria != 38 {
		t.Fatalf("media diaria = %.3f; esperado 38.000 (1140/30)", item.MediaConsumoDiaria)
	}
	if item.EstoqueProjetado != 34.5 {
		t.Fatalf("estoque projetado = %.3f; esperado 34.500", item.EstoqueProjetado)
	}
	if !item.SugerirCompra || item.Nivel != NivelCritico {
		t.Fatalf("esperado CRITICO com sugestão; nivel=%s sugerir=%v", item.Nivel, item.SugerirCompra)
	}
	if item.QuantidadeSugeridaCompra != 165.5 {
		t.Fatalf("qtd compra = %.3f; esperado 165.500", item.QuantidadeSugeridaCompra)
	}
	if item.CustoEstimadoCompra != 140.68 {
		t.Fatalf("custo = %.2f; esperado 140.68", item.CustoEstimadoCompra)
	}
	if item.DiasAteCritico == nil {
		t.Fatal("dias_ate_critico não deveria ser nil")
	}
	// (120-50)/38 ≈ 1.8
	if *item.DiasAteCritico != 1.8 {
		t.Fatalf("dias_ate_critico = %.1f; esperado 1.8", *item.DiasAteCritico)
	}
}

func TestBuildPrevisaoItemSemHistoricoNemBOM(t *testing.T) {
	t.Parallel()

	row := previsaoAggRow{
		InsumoID:        "i2",
		Nome:            "Água oxigenada",
		Unidade:         "ml",
		QuantidadeAtual: 80,
		EstoqueMinimo:   20,
		EstoqueIdeal:    100,
		ValorUnitario:   0.5,
	}
	item := buildPrevisaoItem(row, 30, 7)
	if item.MediaConsumoPorAtendimento != 0 || item.MediaConsumoDiaria != 0 || item.DemandaFutura != 0 {
		t.Fatalf("medias/demanda deveriam ser 0: %+v", item)
	}
	if item.DiasAteCritico != nil {
		t.Fatalf("dias_ate_critico deveria ser null sem media diaria")
	}
	if item.Nivel != NivelOK {
		t.Fatalf("nivel = %s; esperado OK (acima do minimo e ideal/2)", item.Nivel)
	}
	if item.SugerirCompra {
		t.Fatal("não deveria sugerir compra")
	}
}

func TestBuildPrevisaoItemZeroAtendimentos(t *testing.T) {
	t.Parallel()

	row := previsaoAggRow{
		InsumoID:              "i3",
		Nome:                  "X",
		Unidade:               "un",
		QuantidadeAtual:       10,
		EstoqueMinimo:         5,
		EstoqueIdeal:          20,
		ValorUnitario:         1,
		ConsumoHistoricoTotal: sql.NullFloat64{Float64: 0, Valid: true},
		AtendimentosHistorico: sql.NullInt64{Int64: 0, Valid: true},
	}
	item := buildPrevisaoItem(row, 30, 7)
	if item.MediaConsumoPorAtendimento != 0 {
		t.Fatalf("com 0 atendimentos media deve ser 0; got %.3f", item.MediaConsumoPorAtendimento)
	}
}

func TestNormalizePrevisaoParams(t *testing.T) {
	t.Parallel()

	p, err := NormalizePrevisaoParams(DefaultJanelaDias, DefaultHorizonteCriticoDias, false)
	if err != nil {
		t.Fatalf("defaults: %v", err)
	}
	if p.JanelaDias != 30 || p.HorizonteCriticoDias != 7 {
		t.Fatalf("defaults incorretos: %+v", p)
	}

	_, err = NormalizePrevisaoParams(0, 7, false)
	if !errors.Is(err, ErrPrevisaoParamsInvalidos) {
		t.Fatalf("janela 0: esperado ErrPrevisaoParamsInvalidos; got %v", err)
	}

	_, err = NormalizePrevisaoParams(-1, 7, false)
	if !errors.Is(err, ErrPrevisaoParamsInvalidos) {
		t.Fatalf("esperado ErrPrevisaoParamsInvalidos; got %v", err)
	}
	_, err = NormalizePrevisaoParams(30, 400, false)
	if !errors.Is(err, ErrPrevisaoParamsInvalidos) {
		t.Fatalf("horizonte alto: esperado ErrPrevisaoParamsInvalidos; got %v", err)
	}
	_, err = NormalizePrevisaoParams(366, 7, false)
	if !errors.Is(err, ErrPrevisaoParamsInvalidos) {
		t.Fatalf("janela alta: esperado ErrPrevisaoParamsInvalidos; got %v", err)
	}
}

func TestValidateBOMItens(t *testing.T) {
	t.Parallel()

	if err := validateBOMItens([]ServicoInsumoInput{{InsumoID: "a", QuantidadeUso: 1}}); err != nil {
		t.Fatalf("válido: %v", err)
	}
	if err := validateBOMItens([]ServicoInsumoInput{{InsumoID: "a", QuantidadeUso: 0}}); !errors.Is(err, ErrServicoInsumoInvalido) {
		t.Fatalf("qtd 0: %v", err)
	}
	if err := validateBOMItens([]ServicoInsumoInput{
		{InsumoID: "a", QuantidadeUso: 1},
		{InsumoID: "a", QuantidadeUso: 2},
	}); !errors.Is(err, ErrServicoInsumoInvalido) {
		t.Fatalf("duplicado: %v", err)
	}
}

func TestApenasCriticosFilterLogic(t *testing.T) {
	t.Parallel()

	itemOK := buildPrevisaoItem(previsaoAggRow{
		InsumoID: "ok", Nome: "OK", Unidade: "un",
		QuantidadeAtual: 100, EstoqueMinimo: 10, EstoqueIdeal: 100, ValorUnitario: 1,
	}, 30, 7)
	itemCrit := buildPrevisaoItem(previsaoAggRow{
		InsumoID: "c", Nome: "C", Unidade: "un",
		QuantidadeAtual: 5, EstoqueMinimo: 10, EstoqueIdeal: 100, ValorUnitario: 1,
	}, 30, 7)

	if itemOK.SugerirCompra {
		t.Fatal("OK não deve sugerir compra")
	}
	if !itemCrit.SugerirCompra {
		t.Fatal("CRITICO deve sugerir compra")
	}
}
