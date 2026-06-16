package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

var ErrAgendamentoNaoPertenceProfissional = errors.New("agendamento não pertence à profissional")
var ErrAgendamentoStatusInvalido = errors.New("agendamento não está confirmado para conclusão")

type ResumoSemanaProfissional struct {
	ComissaoPendente  float64 `json:"comissao_pendente"`
	ServicosRealizados int    `json:"servicos_realizados"`
	PeriodoLabel      string  `json:"periodo_label"`
	StartDate         string  `json:"start_date"`
	EndDate           string  `json:"end_date"`
}

type AgendamentoTimelineItem struct {
	ID            string    `json:"id"`
	HorarioInicio string    `json:"horario_inicio"`
	HorarioFim    string    `json:"horario_fim"`
	Status        string    `json:"status"`
	ClienteNome   string    `json:"cliente_nome"`
	ServicoNome   string    `json:"servico_nome"`
	Adicionais    []string  `json:"adicionais"`
	DataHoraInicio time.Time `json:"-"`
}

type DashboardProfissional struct {
	ProfissionalNome string                    `json:"profissional_nome"`
	DataSelecionada  string                    `json:"data_selecionada"`
	DataLabel        string                    `json:"data_label"`
	DataAnterior     string                    `json:"data_anterior"`
	DataProxima      string                    `json:"data_proxima"`
	ResumoSemana     ResumoSemanaProfissional  `json:"resumo_semana"`
	Agenda           []AgendamentoTimelineItem `json:"agenda"`
}

// GetDashboardProfissional monta extrato semanal e timeline do dia para a profissional logada.
func (s *AgendaService) GetDashboardProfissional(
	ctx context.Context,
	establishmentID, professionalID string,
	selectedDate time.Time,
) (*DashboardProfissional, error) {
	if err := s.validarProfissionalAtivo(ctx, establishmentID, professionalID); err != nil {
		return nil, err
	}

	nome, err := s.buscarNomeProfissional(ctx, establishmentID, professionalID)
	if err != nil {
		return nil, err
	}

	weekStart, weekEnd, weekLabel := PeriodoSemanaAtual()
	resumo, err := s.GetResumoSemanaProfissional(ctx, establishmentID, professionalID, weekStart, weekEnd, weekLabel)
	if err != nil {
		return nil, err
	}

	agenda, err := s.ListarAgendaDiaProfissional(ctx, establishmentID, professionalID, selectedDate)
	if err != nil {
		return nil, err
	}

	prevDay := selectedDate.AddDate(0, 0, -1)
	nextDay := selectedDate.AddDate(0, 0, 1)

	return &DashboardProfissional{
		ProfissionalNome: nome,
		DataSelecionada:  selectedDate.Format("2006-01-02"),
		DataLabel:        formatarDataLabel(selectedDate),
		DataAnterior:     prevDay.Format("2006-01-02"),
		DataProxima:      nextDay.Format("2006-01-02"),
		ResumoSemana:     *resumo,
		Agenda:           agenda,
	}, nil
}

// GetResumoSemanaProfissional retorna comissões pendentes e serviços concluídos na semana.
func (s *AgendaService) GetResumoSemanaProfissional(
	ctx context.Context,
	establishmentID, professionalID string,
	startDate, endDate time.Time,
	periodoLabel string,
) (*ResumoSemanaProfissional, error) {
	inicio, fim := intervaloRelatorio(startDate, endDate)

	const query = `
SELECT
    COALESCE((
        SELECT SUM(fc.valor)
        FROM fluxo_caixa fc
        WHERE fc.estabelecimento_id = $1
          AND fc.profissional_id = $2
          AND fc.tipo = 'CUSTO_VARIAVEL'
          AND fc.status_pagamento = 'PENDENTE'
          AND fc.data_transacao >= $3
          AND fc.data_transacao < $4
    ), 0) AS comissao_pendente,
    COALESCE((
        SELECT COUNT(*)::INTEGER
        FROM agendamentos a
        WHERE a.estabelecimento_id = $1
          AND a.profissional_id = $2
          AND a.status = 'CONCLUIDO'
          AND a.data_hora_inicio >= $3
          AND a.data_hora_inicio < $4
    ), 0) AS servicos_realizados
`
	var row struct {
		ComissaoPendente   float64 `db:"comissao_pendente"`
		ServicosRealizados int     `db:"servicos_realizados"`
	}
	if err := s.db.GetContext(ctx, &row, query, establishmentID, professionalID, inicio, fim); err != nil {
		return nil, fmt.Errorf("buscar resumo semanal: %w", err)
	}

	return &ResumoSemanaProfissional{
		ComissaoPendente:   roundMoney(row.ComissaoPendente),
		ServicosRealizados: row.ServicosRealizados,
		PeriodoLabel:       periodoLabel,
		StartDate:          startDate.Format("2006-01-02"),
		EndDate:            endDate.Format("2006-01-02"),
	}, nil
}

// ListarAgendaDiaProfissional retorna agendamentos da profissional no dia, ordenados por hora.
func (s *AgendaService) ListarAgendaDiaProfissional(
	ctx context.Context,
	establishmentID, professionalID string,
	date time.Time,
) ([]AgendamentoTimelineItem, error) {
	inicioDia, fimDia := limitesDoDia(date)

	const query = `
SELECT
    a.id,
    a.data_hora_inicio,
    a.data_hora_fim,
    a.status,
    c.nome AS cliente_nome,
    s.nome AS servico_nome,
    COALESCE(
        ARRAY_AGG(sa.nome ORDER BY sa.nome) FILTER (WHERE sa.nome IS NOT NULL),
        '{}'
    ) AS adicionais
FROM agendamentos a
INNER JOIN clientes c ON c.id = a.cliente_id AND c.estabelecimento_id = a.estabelecimento_id
INNER JOIN servicos s ON s.id = a.servico_id AND s.estabelecimento_id = a.estabelecimento_id
LEFT JOIN agendamento_adicionais aa ON aa.agendamento_id = a.id
LEFT JOIN servico_adicionais sa ON sa.id = aa.adicional_id
WHERE a.estabelecimento_id = $1
  AND a.profissional_id = $2
  AND a.data_hora_inicio >= $3
  AND a.data_hora_inicio < $4
GROUP BY a.id, a.data_hora_inicio, a.data_hora_fim, a.status, c.nome, s.nome
ORDER BY a.data_hora_inicio ASC
`
	rows, err := s.db.QueryxContext(ctx, query, establishmentID, professionalID, inicioDia, fimDia)
	if err != nil {
		return nil, fmt.Errorf("listar agenda do dia: %w", err)
	}
	defer rows.Close()

	items := make([]AgendamentoTimelineItem, 0, 8)
	for rows.Next() {
		var (
			item        AgendamentoTimelineItem
			adicionais  pq.StringArray
			inicio      time.Time
			fim         time.Time
		)
		if err := rows.Scan(
			&item.ID,
			&inicio,
			&fim,
			&item.Status,
			&item.ClienteNome,
			&item.ServicoNome,
			&adicionais,
		); err != nil {
			return nil, fmt.Errorf("ler agendamento da timeline: %w", err)
		}
		item.DataHoraInicio = inicio
		item.HorarioInicio = inicio.Format("15:04")
		item.HorarioFim = fim.Format("15:04")
		item.Adicionais = []string(adicionais)
		if item.Adicionais == nil {
			item.Adicionais = []string{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar agenda do dia: %w", err)
	}

	return items, nil
}

// ValidarAgendamentoDaProfissional garante isolamento de dados por profissional.
func (s *AgendaService) ValidarAgendamentoDaProfissional(
	ctx context.Context,
	establishmentID, professionalID, agendamentoID string,
) error {
	const query = `
SELECT id FROM agendamentos
WHERE id = $1
  AND estabelecimento_id = $2
  AND profissional_id = $3
`
	var id string
	if err := s.db.GetContext(ctx, &id, query, agendamentoID, establishmentID, professionalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAgendamentoNaoPertenceProfissional
		}
		return fmt.Errorf("validar agendamento da profissional: %w", err)
	}
	return nil
}

// ConcluirAtendimentoProfissional valida ownership e conclui atendimento confirmado.
func (s *AgendaService) ConcluirAtendimentoProfissional(
	ctx context.Context,
	establishmentID, professionalID, agendamentoID string,
	financeiro *FinanceiroService,
) error {
	if err := s.ValidarAgendamentoDaProfissional(ctx, establishmentID, professionalID, agendamentoID); err != nil {
		return err
	}

	const queryStatus = `
SELECT status FROM agendamentos
WHERE id = $1 AND estabelecimento_id = $2 AND profissional_id = $3
`
	var status string
	if err := s.db.GetContext(ctx, &status, queryStatus, agendamentoID, establishmentID, professionalID); err != nil {
		return fmt.Errorf("buscar status do agendamento: %w", err)
	}
	if status != "CONFIRMADO" {
		return ErrAgendamentoStatusInvalido
	}

	return financeiro.ConcluirAtendimento(ctx, agendamentoID)
}

func (s *AgendaService) buscarNomeProfissional(ctx context.Context, establishmentID, professionalID string) (string, error) {
	const query = `
SELECT nome FROM profissionais
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var nome string
	if err := s.db.GetContext(ctx, &nome, query, professionalID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrProfissionalNaoEncontrado
		}
		return "", fmt.Errorf("buscar nome da profissional: %w", err)
	}
	return nome, nil
}

// PeriodoSemanaAtual retorna segunda a domingo da semana corrente.
func PeriodoSemanaAtual() (time.Time, time.Time, string) {
	now := time.Now()
	loc := now.Location()
	daysSinceMonday := (int(now.Weekday()) + 6) % 7
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -daysSinceMonday)
	end := start.AddDate(0, 0, 6)
	label := fmt.Sprintf("%s — %s", formatarDataCurta(start), formatarDataCurta(end))
	return start, end, label
}

func formatarDataLabel(date time.Time) string {
	dias := []string{"Domingo", "Segunda-feira", "Terça-feira", "Quarta-feira", "Quinta-feira", "Sexta-feira", "Sábado"}
	meses := []string{
		"janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro",
	}
	return fmt.Sprintf("%s, %d de %s", dias[date.Weekday()], date.Day(), meses[date.Month()-1])
}

func formatarDataCurta(date time.Time) string {
	return fmt.Sprintf("%02d/%02d", date.Day(), date.Month())
}

func ParseDataQuery(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now(), nil
	}
	return time.ParseInLocation("2006-01-02", raw, time.Local)
}
