package service

import (
	"errors"
	"testing"
)

func ptrFloat(v float64) *float64 { return &v }

func TestResolverQuantidadeBase_ModoEmbalagens(t *testing.T) {
	t.Parallel()

	// Caso canônico Dona: 10 frascos × 50 ml → 500 ml
	got, err := resolverQuantidadeBase(nil, ptrFloat(10), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 500 {
		t.Fatalf("quantidade = %v; want 500", got)
	}
}

func TestResolverQuantidadeBase_ModoTotalCompat(t *testing.T) {
	t.Parallel()

	got, err := resolverQuantidadeBase(ptrFloat(500), nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 500 {
		t.Fatalf("quantidade = %v; want 500", got)
	}
}

func TestResolverQuantidadeBase_PriorizaEmbalagensQuandoAmbos(t *testing.T) {
	t.Parallel()

	got, err := resolverQuantidadeBase(ptrFloat(999), ptrFloat(10), 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 500 {
		t.Fatalf("quantidade = %v; want 500 (embalagens wins)", got)
	}
}

func TestResolverQuantidadeBase_SemEntradaZero(t *testing.T) {
	t.Parallel()

	got, err := resolverQuantidadeBase(nil, nil, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 0 {
		t.Fatalf("quantidade = %v; want 0", got)
	}
}

func TestResolverQuantidadeBase_Negativo(t *testing.T) {
	t.Parallel()

	_, err := resolverQuantidadeBase(ptrFloat(-1), nil, 1)
	if !errors.Is(err, ErrInsumoPayload) {
		t.Fatalf("err = %v; want ErrInsumoPayload", err)
	}
	_, err = resolverQuantidadeBase(nil, ptrFloat(-1), 1)
	if !errors.Is(err, ErrInsumoPayload) {
		t.Fatalf("err = %v; want ErrInsumoPayload", err)
	}
}

func TestCalcularQuantidadeEmbalagens(t *testing.T) {
	t.Parallel()

	if got := calcularQuantidadeEmbalagens(500, 50); got != 10 {
		t.Fatalf("embalagens = %v; want 10", got)
	}
	// Divisão não exata ainda retorna float com 3 casas
	if got := calcularQuantidadeEmbalagens(100, 30); got != 3.333 {
		t.Fatalf("embalagens = %v; want 3.333", got)
	}
	if got := calcularQuantidadeEmbalagens(10, 0); got != 0 {
		t.Fatalf("embalagens = %v; want 0 when conteudo<=0", got)
	}
}

func TestNormalizeInsumoInput_Canonico10x50(t *testing.T) {
	t.Parallel()

	_, _, unidade, conteudo, quantidade, _, _, _, err := normalizeInsumoInput(InsumoInput{
		Nome:                 "Shampoo X",
		Unidade:              "ML",
		ConteudoPorEmbalagem: 50,
		QuantidadeEmbalagens: ptrFloat(10),
		EstoqueMinimo:        100,
		EstoqueIdeal:         500,
		ValorUnitario:        0.18,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unidade != "ml" {
		t.Fatalf("unidade = %q; want ml", unidade)
	}
	if conteudo != 50 {
		t.Fatalf("conteudo = %v; want 50", conteudo)
	}
	if quantidade != 500 {
		t.Fatalf("quantidade = %v; want 500", quantidade)
	}
}

func TestNormalizeUnidadeInsumo(t *testing.T) {
	t.Parallel()

	u, err := normalizeUnidadeInsumo("")
	if err != nil || u != "un" {
		t.Fatalf("default = (%q, %v); want (un, nil)", u, err)
	}
	_, err = normalizeUnidadeInsumo("litros")
	if !errors.Is(err, ErrInsumoPayload) {
		t.Fatalf("err = %v; want ErrInsumoPayload", err)
	}
}

func TestResolveAjusteDelta(t *testing.T) {
	t.Parallel()

	delta, err := resolveAjusteDelta(AjusteEstoqueInput{Delta: ptrFloat(-30)}, 50)
	if err != nil || delta != -30 {
		t.Fatalf("delta base = (%v, %v); want (-30, nil)", delta, err)
	}

	delta, err = resolveAjusteDelta(AjusteEstoqueInput{DeltaEmbalagens: ptrFloat(-1)}, 50)
	if err != nil || delta != -50 {
		t.Fatalf("delta embalagens = (%v, %v); want (-50, nil)", delta, err)
	}

	// Ambos: prioriza delta_embalagens
	delta, err = resolveAjusteDelta(AjusteEstoqueInput{
		Delta:           ptrFloat(-999),
		DeltaEmbalagens: ptrFloat(-2),
	}, 50)
	if err != nil || delta != -100 {
		t.Fatalf("prioridade embalagens = (%v, %v); want (-100, nil)", delta, err)
	}

	_, err = resolveAjusteDelta(AjusteEstoqueInput{}, 50)
	if !errors.Is(err, ErrInsumoPayload) {
		t.Fatalf("err = %v; want ErrInsumoPayload", err)
	}
}
