package service

import "math"

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func calcularComissao(preco, porcentagem float64) float64 {
	return roundMoney(preco * porcentagem / 100)
}
