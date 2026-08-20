package service

import "math"

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

// roundQty arredonda quantidades de estoque em 3 casas (NUMERIC(12,3)).
func roundQty(value float64) float64 {
	return math.Round(value*1000) / 1000
}

// mulMoneyByQty multiplica quantidade (3 casas) × valor unitário (2 casas)
// em escala inteira, evitando artefatos float64 (ex.: 165.5×0.85 → 140.68).
func mulMoneyByQty(qty, unitPrice float64) float64 {
	qMilli := int64(math.Round(qty * 1000))
	pCents := int64(math.Round(unitPrice * 100))
	cents := int64(math.Round(float64(qMilli*pCents) / 1000.0))
	return float64(cents) / 100.0
}

func calcularComissao(preco, porcentagem float64) float64 {
	return roundMoney(preco * porcentagem / 100)
}
