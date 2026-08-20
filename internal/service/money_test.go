package service

import "testing"

func TestMulMoneyByQtyArredondaAoCentavoEstavel(t *testing.T) {
	t.Parallel()

	// Caso do contrato DEV-200: float64 direto daria 140.674999… → 140.67.
	if got := mulMoneyByQty(165.5, 0.85); got != 140.68 {
		t.Fatalf("mulMoneyByQty(165.5, 0.85) = %.2f; esperado 140.68", got)
	}
	if got := mulMoneyByQty(100, 1.99); got != 199 {
		t.Fatalf("mulMoneyByQty(100, 1.99) = %.2f; esperado 199.00", got)
	}
	if got := mulMoneyByQty(0, 10); got != 0 {
		t.Fatalf("mulMoneyByQty(0, 10) = %.2f; esperado 0", got)
	}
}

func TestCalcularComissaoArredondaAoCentavo(t *testing.T) {
	t.Parallel()

	casos := []struct {
		nome        string
		valor       float64
		porcentagem float64
		esperado    float64
	}{
		{nome: "valor inteiro", valor: 130, porcentagem: 45, esperado: 58.50},
		{nome: "arredonda para cima", valor: 130, porcentagem: 33.33, esperado: 43.33},
		{nome: "meio centavo para cima", valor: 0.05, porcentagem: 50, esperado: 0.03},
		{nome: "comissão zero", valor: 99.99, porcentagem: 0, esperado: 0},
		{nome: "comissão integral", valor: 99.99, porcentagem: 100, esperado: 99.99},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nome, func(t *testing.T) {
			t.Parallel()

			if obtido := calcularComissao(caso.valor, caso.porcentagem); obtido != caso.esperado {
				t.Fatalf(
					"calcularComissao(%.2f, %.2f) = %.2f; esperado %.2f",
					caso.valor,
					caso.porcentagem,
					obtido,
					caso.esperado,
				)
			}
		})
	}
}
