package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

const intervaloSlotMinutos = 30

type intervaloAgendado struct {
	Inicio time.Time `db:"data_hora_inicio"`
	Fim    time.Time `db:"data_hora_fim"`
}

type expedienteDia struct {
	HorarioEntrada string  `db:"horario_entrada"`
	InicioAlmoco   *string `db:"inicio_almoco"`
	FimAlmoco      *string `db:"fim_almoco"`
	HorarioSaida   string  `db:"horario_saida"`
}

// GetAvailableSlots retorna horários livres da profissional no dia informado,
// respeitando a jornada de trabalho, pausa de almoço, duração do procedimento
// e agendamentos já confirmados.
func (s *AgendaService) GetAvailableSlots(
	ctx context.Context,
	establishmentID, professionalID string,
	date time.Time,
	procedureID string,
	additionalIDs []string,
) ([]string, error) {
	if err := s.validarProfissionalAtivo(ctx, establishmentID, professionalID); err != nil {
		return nil, err
	}

	expediente, err := s.buscarExpedienteDoDia(ctx, professionalID, int(date.Weekday()))
	if err != nil {
		return nil, err
	}
	if expediente == nil {
		return []string{}, nil
	}

	duracaoTotal, err := s.calcularDuracaoTotal(ctx, establishmentID, procedureID, additionalIDs)
	if err != nil {
		return nil, err
	}

	ocupados, err := s.buscarAgendamentosDoDia(ctx, establishmentID, professionalID, date)
	if err != nil {
		return nil, err
	}

	slots, err := filtrarSlotsDisponiveis(date, duracaoTotal, ocupados, expediente)
	if err != nil {
		return nil, err
	}
	if slots == nil {
		slots = []string{}
	}
	return slots, nil
}

func (s *AgendaService) buscarExpedienteDoDia(
	ctx context.Context,
	professionalID string,
	diaSemana int,
) (*expedienteDia, error) {
	const query = `
SELECT horario_entrada, inicio_almoco, fim_almoco, horario_saida
FROM expedientes_profissionais
WHERE profissional_id = $1
  AND dia_semana = $2
`
	var exp expedienteDia
	err := s.db.GetContext(ctx, &exp, query, professionalID, diaSemana)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("buscar expediente da profissional: %w", err)
	}
	return &exp, nil
}

func (s *AgendaService) calcularDuracaoTotal(
	ctx context.Context,
	establishmentID, procedureID string,
	additionalIDs []string,
) (int, error) {
	const queryServico = `
SELECT duracao_base_minutos
FROM servicos
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var duracaoBase int
	if err := s.db.GetContext(ctx, &duracaoBase, queryServico, procedureID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrServicoNaoEncontrado
		}
		return 0, fmt.Errorf("buscar duração do serviço: %w", err)
	}

	if len(additionalIDs) == 0 {
		return duracaoBase, nil
	}

	const queryAdicionais = `
SELECT
    COALESCE(SUM(sa.duracao_adicional_minutos), 0)::INTEGER AS duracao_minutos,
    COUNT(sa.id)::INTEGER AS quantidade
FROM servico_adicionais sa
INNER JOIN servicos s ON s.id = sa.servico_id
WHERE sa.servico_id = $1
  AND s.estabelecimento_id = $2
  AND sa.id = ANY($3)
`
	var resultado struct {
		DuracaoMinutos int `db:"duracao_minutos"`
		Quantidade     int `db:"quantidade"`
	}
	if err := s.db.GetContext(ctx, &resultado, queryAdicionais, procedureID, establishmentID, pq.Array(additionalIDs)); err != nil {
		return 0, fmt.Errorf("somar duração dos adicionais: %w", err)
	}
	if resultado.Quantidade != len(additionalIDs) {
		return 0, ErrAdicionalInvalido
	}

	return duracaoBase + resultado.DuracaoMinutos, nil
}

func (s *AgendaService) validarProfissionalAtivo(ctx context.Context, establishmentID, professionalID string) error {
	const query = `
SELECT id FROM profissionais
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var id string
	if err := s.db.GetContext(ctx, &id, query, professionalID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfissionalNaoEncontrado
		}
		return fmt.Errorf("validar profissional: %w", err)
	}
	return nil
}

func (s *AgendaService) buscarAgendamentosDoDia(
	ctx context.Context,
	establishmentID, professionalID string,
	date time.Time,
) ([]intervaloAgendado, error) {
	inicioDia, fimDia := limitesDoDia(date)

	const query = `
SELECT data_hora_inicio, data_hora_fim
FROM agendamentos
WHERE estabelecimento_id = $1
  AND profissional_id = $2
  AND status IN ('AGENDADO', 'CONFIRMADO')
  AND data_hora_inicio >= $3
  AND data_hora_inicio < $4
ORDER BY data_hora_inicio ASC
`
	var ocupados []intervaloAgendado
	if err := s.db.SelectContext(ctx, &ocupados, query, establishmentID, professionalID, inicioDia, fimDia); err != nil {
		return nil, fmt.Errorf("buscar agendamentos do dia: %w", err)
	}
	if ocupados == nil {
		ocupados = []intervaloAgendado{}
	}
	return ocupados, nil
}

func limitesDoDia(date time.Time) (time.Time, time.Time) {
	y, m, d := date.Date()
	loc := date.Location()
	inicio := time.Date(y, m, d, 0, 0, 0, 0, loc)
	fim := inicio.Add(24 * time.Hour)
	return inicio, fim
}

func filtrarSlotsDisponiveis(
	date time.Time,
	duracaoMinutos int,
	ocupados []intervaloAgendado,
	expediente *expedienteDia,
) ([]string, error) {
	abertura, err := horarioDoDia(date, expediente.HorarioEntrada)
	if err != nil {
		return nil, fmt.Errorf("horario_entrada: %w", err)
	}
	fechamento, err := horarioDoDia(date, expediente.HorarioSaida)
	if err != nil {
		return nil, fmt.Errorf("horario_saida: %w", err)
	}

	bloqueios := make([]intervaloAgendado, len(ocupados))
	copy(bloqueios, ocupados)

	if expediente.InicioAlmoco != nil && expediente.FimAlmoco != nil {
		inicioAlmoco, err := horarioDoDia(date, *expediente.InicioAlmoco)
		if err != nil {
			return nil, fmt.Errorf("inicio_almoco: %w", err)
		}
		fimAlmoco, err := horarioDoDia(date, *expediente.FimAlmoco)
		if err != nil {
			return nil, fmt.Errorf("fim_almoco: %w", err)
		}
		bloqueios = append(bloqueios, intervaloAgendado{
			Inicio: inicioAlmoco,
			Fim:    fimAlmoco,
		})
	}

	duracao := time.Duration(duracaoMinutos) * time.Minute
	intervalo := time.Duration(intervaloSlotMinutos) * time.Minute

	slots := make([]string, 0, 16)
	for candidato := abertura; candidato.Add(duracao).Compare(fechamento) <= 0; candidato = candidato.Add(intervalo) {
		fimProposto := candidato.Add(duracao)
		if !intervaloLivre(candidato, fimProposto, bloqueios) {
			continue
		}
		slots = append(slots, candidato.Format("15:04"))
	}

	return slots, nil
}

func horarioDoDia(date time.Time, hora string) (time.Time, error) {
	parsed, err := parseHorarioExpediente(hora)
	if err != nil {
		return time.Time{}, err
	}
	y, m, d := date.Date()
	loc := date.Location()
	return time.Date(y, m, d, parsed.Hour(), parsed.Minute(), parsed.Second(), 0, loc), nil
}

func intervaloLivre(inicio, fim time.Time, bloqueios []intervaloAgendado) bool {
	for _, bloco := range bloqueios {
		if intervalosSobrepostos(inicio, fim, bloco.Inicio, bloco.Fim) {
			return false
		}
	}
	return true
}

func intervalosSobrepostos(inicioA, fimA, inicioB, fimB time.Time) bool {
	return inicioA.Before(fimB) && fimA.After(inicioB)
}
