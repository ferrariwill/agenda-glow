package service

import "math"

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

// roundQty arredonda quantidades de estoque em 3 casas (NUMERIC(12,3)).
func roundQty(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func calcularComissao(preco, porcentagem float64) float64 {
	return roundMoney(preco * porcentagem / 100)
}
