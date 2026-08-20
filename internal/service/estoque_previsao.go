package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrPrevisaoParamsInvalidos = errors.New("parâmetros de previsão inválidos")

const (
	NivelCritico = "CRITICO"
	NivelAtencao = "ATENCAO"
	NivelOK      = "OK"

	DefaultJanelaDias           = 30
	DefaultHorizonteCriticoDias = 7
	maxJanelaDias               = 365
	maxHorizonteCriticoDias     = 365
)

type EstoquePrevisaoService struct {
	db *sqlx.DB
}

func NewEstoquePrevisaoService(db *sqlx.DB) *EstoquePrevisaoService {
	return &EstoquePrevisaoService{db: db}
}

type PrevisaoParams struct {
	JanelaDias           int
	HorizonteCriticoDias int
	ApenasCriticos       bool
}

type PrevisaoResumo struct {
	TotalInsumos int `json:"total_insumos"`
	Criticos     int `json:"criticos"`
	Atencao      int `json:"atencao"`
	OK           int `json:"ok"`
}

type PrevisaoItem struct {
	InsumoID                   string   `json:"insumo_id"`
	Nome                       string   `json:"nome"`
	Unidade                    string   `json:"unidade"`
	QuantidadeAtual            float64  `json:"quantidade_atual"`
	EstoqueMinimo              float64  `json:"estoque_minimo"`
	EstoqueIdeal               float64  `json:"estoque_ideal"`
	ValorUnitario              float64  `json:"valor_unitario"`
	MediaConsumoPorAtendimento float64  `json:"media_consumo_por_atendimento"`
	MediaConsumoDiaria         float64  `json:"media_consumo_diaria"`
	AtendimentosHistorico      int      `json:"atendimentos_historico"`
	ConsumoHistoricoTotal      float64  `json:"consumo_historico_total"`
	DemandaFutura              float64  `json:"demanda_futura"`
	AgendamentosFuturos        int      `json:"agendamentos_futuros"`
	EstoqueProjetado           float64  `json:"estoque_projetado"`
	DiasAteCritico             *float64 `json:"dias_ate_critico"`
	Nivel                      string   `json:"nivel"`
	SugerirCompra              bool     `json:"sugerir_compra"`
	QuantidadeSugeridaCompra   float64  `json:"quantidade_sugerida_compra"`
	CustoEstimadoCompra        float64  `json:"custo_estimado_compra"`
}

type PrevisaoEstoqueResponse struct {
	GeradoEm             time.Time      `json:"gerado_em"`
	JanelaDias           int            `json:"janela_dias"`
	HorizonteCriticoDias int            `json:"horizonte_critico_dias"`
	Resumo               PrevisaoResumo `json:"resumo"`
	Itens                []PrevisaoItem `json:"itens"`
}

type previsaoAggRow struct {
	InsumoID              string         `db:"insumo_id"`
	Nome                  string         `db:"nome"`
	Unidade               string         `db:"unidade"`
	QuantidadeAtual       float64        `db:"quantidade_atual"`
	EstoqueMinimo         float64        `db:"estoque_minimo"`
	EstoqueIdeal          float64        `db:"estoque_ideal"`
	ValorUnitario         float64        `db:"valor_unitario"`
	ConsumoHistoricoTotal sql.NullFloat64 `db:"consumo_historico_total"`
	AtendimentosHistorico sql.NullInt64  `db:"atendimentos_historico"`
	DemandaFutura         sql.NullFloat64 `db:"demanda_futura"`
	AgendamentosFuturos   sql.NullInt64  `db:"agendamentos_futuros"`
}

// NormalizePrevisaoParams valida ranges (1..365). Defaults devem ser aplicados
// pelo handler quando o query param está ausente.
func NormalizePrevisaoParams(janela, horizonte int, apenasCriticos bool) (PrevisaoParams, error) {
	if janela < 1 || janela > maxJanelaDias {
		return PrevisaoParams{}, fmt.Errorf("%w: janela_dias deve estar entre 1 e %d", ErrPrevisaoParamsInvalidos, maxJanelaDias)
	}
	if horizonte < 1 || horizonte > maxHorizonteCriticoDias {
		return PrevisaoParams{}, fmt.Errorf("%w: horizonte_critico_dias deve estar entre 1 e %d", ErrPrevisaoParamsInvalidos, maxHorizonteCriticoDias)
	}
	return PrevisaoParams{
		JanelaDias:           janela,
		HorizonteCriticoDias: horizonte,
		ApenasCriticos:       apenasCriticos,
	}, nil
}

// Forecast calcula previsão de estoque e sugestão de compra por insumo ativo do tenant.
func (s *EstoquePrevisaoService) Forecast(
	ctx context.Context,
	establishmentID string,
	params PrevisaoParams,
) (*PrevisaoEstoqueResponse, error) {
	now := time.Now().UTC()
	windowStart := now.AddDate(0, 0, -params.JanelaDias)

	const query = `
WITH hist_ag AS (
    SELECT a.id, a.servico_id AS fallback_servico_id
    FROM agendamentos a
    WHERE a.estabelecimento_id = $1
      AND a.status = 'CONCLUIDO'
      AND a.data_hora_inicio >= $2
      AND a.data_hora_inicio < $3
),
hist_svc AS (
    SELECT h.id AS agendamento_id, ags.servico_id
    FROM hist_ag h
    INNER JOIN agendamento_servicos ags ON ags.agendamento_id = h.id
    UNION ALL
    SELECT h.id, h.fallback_servico_id
    FROM hist_ag h
    WHERE NOT EXISTS (
        SELECT 1 FROM agendamento_servicos ags WHERE ags.agendamento_id = h.id
    )
),
hist_consumo AS (
    SELECT
        si.insumo_id,
        COALESCE(SUM(si.quantidade_uso), 0) AS consumo_historico_total,
        COUNT(DISTINCT hs.agendamento_id) AS atendimentos_historico
    FROM hist_svc hs
    INNER JOIN servico_insumos si
        ON si.servico_id = hs.servico_id
       AND si.estabelecimento_id = $1
    GROUP BY si.insumo_id
),
fut_ag AS (
    SELECT a.id, a.servico_id AS fallback_servico_id
    FROM agendamentos a
    WHERE a.estabelecimento_id = $1
      AND a.status IN ('AGENDADO', 'CONFIRMADO')
      AND a.data_hora_inicio >= $3
),
fut_svc AS (
    SELECT f.id AS agendamento_id, ags.servico_id
    FROM fut_ag f
    INNER JOIN agendamento_servicos ags ON ags.agendamento_id = f.id
    UNION ALL
    SELECT f.id, f.fallback_servico_id
    FROM fut_ag f
    WHERE NOT EXISTS (
        SELECT 1 FROM agendamento_servicos ags WHERE ags.agendamento_id = f.id
    )
),
fut_demanda AS (
    SELECT
        si.insumo_id,
        COALESCE(SUM(si.quantidade_uso), 0) AS demanda_futura,
        COUNT(DISTINCT fs.agendamento_id) AS agendamentos_futuros
    FROM fut_svc fs
    INNER JOIN servico_insumos si
        ON si.servico_id = fs.servico_id
       AND si.estabelecimento_id = $1
    GROUP BY si.insumo_id
)
SELECT
    i.id AS insumo_id,
    i.nome,
    i.unidade,
    i.quantidade AS quantidade_atual,
    i.estoque_minimo,
    i.estoque_ideal,
    i.valor_unitario,
    hc.consumo_historico_total,
    hc.atendimentos_historico,
    fd.demanda_futura,
    fd.agendamentos_futuros
FROM insumos i
LEFT JOIN hist_consumo hc ON hc.insumo_id = i.id
LEFT JOIN fut_demanda fd ON fd.insumo_id = i.id
WHERE i.estabelecimento_id = $1
  AND i.ativo = TRUE
ORDER BY i.nome
`

	var rows []previsaoAggRow
	if err := s.db.SelectContext(ctx, &rows, query, establishmentID, windowStart, now); err != nil {
		return nil, fmt.Errorf("calcular previsão de estoque: %w", err)
	}

	itens := make([]PrevisaoItem, 0, len(rows))
	resumo := PrevisaoResumo{}
	for _, row := range rows {
		item := buildPrevisaoItem(row, params.JanelaDias, params.HorizonteCriticoDias)
		resumo.TotalInsumos++
		switch item.Nivel {
		case NivelCritico:
			resumo.Criticos++
		case NivelAtencao:
			resumo.Atencao++
		default:
			resumo.OK++
		}
		if params.ApenasCriticos && !item.SugerirCompra {
			continue
		}
		itens = append(itens, item)
	}

	return &PrevisaoEstoqueResponse{
		GeradoEm:             now,
		JanelaDias:           params.JanelaDias,
		HorizonteCriticoDias: params.HorizonteCriticoDias,
		Resumo:               resumo,
		Itens:                itens,
	}, nil
}

func buildPrevisaoItem(row previsaoAggRow, janelaDias, horizonteCritico int) PrevisaoItem {
	consumoTotal := nullFloat(row.ConsumoHistoricoTotal)
	atendimentos := int(nullInt(row.AtendimentosHistorico))
	demandaFutura := nullFloat(row.DemandaFutura)
	agFuturos := int(nullInt(row.AgendamentosFuturos))

	mediaPorAtendimento := 0.0
	if atendimentos > 0 {
		mediaPorAtendimento = roundQty(consumoTotal / float64(atendimentos))
	}
	mediaDiaria := 0.0
	if janelaDias > 0 {
		mediaDiaria = roundQty(consumoTotal / float64(janelaDias))
	}

	estoqueProjetado := roundQty(row.QuantidadeAtual - demandaFutura)
	diasAteCritico := calcularDiasAteCritico(row.QuantidadeAtual, row.EstoqueMinimo, mediaDiaria)
	nivel := ClassificarNivelEstoque(
		row.QuantidadeAtual,
		row.EstoqueMinimo,
		row.EstoqueIdeal,
		estoqueProjetado,
		diasAteCritico,
		horizonteCritico,
	)
	sugerir := nivel == NivelCritico
	qtdCompra := CalcularQuantidadeSugeridaCompra(row.EstoqueIdeal, estoqueProjetado)
	custo := roundMoney(qtdCompra * row.ValorUnitario)

	return PrevisaoItem{
		InsumoID:                   row.InsumoID,
		Nome:                       row.Nome,
		Unidade:                    row.Unidade,
		QuantidadeAtual:            roundQty(row.QuantidadeAtual),
		EstoqueMinimo:              roundQty(row.EstoqueMinimo),
		EstoqueIdeal:               roundQty(row.EstoqueIdeal),
		ValorUnitario:              roundMoney(row.ValorUnitario),
		MediaConsumoPorAtendimento: mediaPorAtendimento,
		MediaConsumoDiaria:         mediaDiaria,
		AtendimentosHistorico:      atendimentos,
		ConsumoHistoricoTotal:      roundQty(consumoTotal),
		DemandaFutura:              roundQty(demandaFutura),
		AgendamentosFuturos:        agFuturos,
		EstoqueProjetado:           estoqueProjetado,
		DiasAteCritico:             diasAteCritico,
		Nivel:                      nivel,
		SugerirCompra:              sugerir,
		QuantidadeSugeridaCompra:   qtdCompra,
		CustoEstimadoCompra:        custo,
	}
}

// ClassificarNivelEstoque aplica as regras CRITICO / ATENCAO / OK.
func ClassificarNivelEstoque(
	quantidadeAtual, estoqueMinimo, estoqueIdeal, estoqueProjetado float64,
	diasAteCritico *float64,
	horizonteCriticoDias int,
) string {
	if quantidadeAtual <= estoqueMinimo || estoqueProjetado <= estoqueMinimo {
		return NivelCritico
	}
	if diasAteCritico != nil && *diasAteCritico <= float64(horizonteCriticoDias) {
		return NivelCritico
	}
	if estoqueProjetado <= estoqueIdeal*0.5 {
		return NivelAtencao
	}
	return NivelOK
}

// CalcularQuantidadeSugeridaCompra = max(0, ideal - projetado), 3 casas.
func CalcularQuantidadeSugeridaCompra(estoqueIdeal, estoqueProjetado float64) float64 {
	diff := estoqueIdeal - estoqueProjetado
	if diff < 0 {
		return 0
	}
	return roundQty(diff)
}

func calcularDiasAteCritico(quantidadeAtual, estoqueMinimo, mediaConsumoDiaria float64) *float64 {
	if mediaConsumoDiaria <= 0 {
		return nil
	}
	dias := (quantidadeAtual - estoqueMinimo) / mediaConsumoDiaria
	dias = math.Round(dias*10) / 10
	return &dias
}

func nullFloat(v sql.NullFloat64) float64 {
	if !v.Valid {
		return 0
	}
	return v.Float64
}

func nullInt(v sql.NullInt64) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}
