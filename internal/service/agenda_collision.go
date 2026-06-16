package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type resultadoColisao int

const (
	colisaoNenhuma resultadoColisao = iota
	colisaoTotalInicio
	colisaoSobreposicaoTermino
)

type detalheSobreposicao struct {
	ProximoInicio time.Time
	MinutosInvadidos int
}

// avaliarColisaoHorario distingue colisão total no início (rejeitar) de
// sobreposição parcial de término (aguardar aprovação da profissional).
func (s *AgendaService) avaliarColisaoHorario(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, profissionalID string,
	inicio, fim time.Time,
) (resultadoColisao, *detalheSobreposicao, error) {
	if err := s.verificarInicioOcupado(ctx, tx, estabelecimentoID, profissionalID, inicio); err != nil {
		if errors.Is(err, ErrColisaoHorario) {
			return colisaoTotalInicio, nil, nil
		}
		return 0, nil, err
	}

	detalhe, err := s.buscarSobreposicaoTermino(ctx, tx, estabelecimentoID, profissionalID, inicio, fim)
	if err != nil {
		return 0, nil, err
	}
	if detalhe != nil {
		return colisaoSobreposicaoTermino, detalhe, nil
	}

	return colisaoNenhuma, nil, nil
}

func (s *AgendaService) verificarInicioOcupado(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, profissionalID string,
	inicio time.Time,
) error {
	const query = `
SELECT a.id
FROM agendamentos a
WHERE a.estabelecimento_id = $1
  AND a.profissional_id = $2
  AND a.status IN ('AGENDADO', 'CONFIRMADO')
  AND a.data_hora_inicio <= $3
  AND a.data_hora_fim > $3
LIMIT 1
FOR UPDATE OF a
`
	var conflito string
	err := tx.GetContext(ctx, &conflito, query, estabelecimentoID, profissionalID, inicio)
	if err == nil {
		return ErrColisaoHorario
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("verificar início ocupado: %w", err)
	}
	return nil
}

func (s *AgendaService) buscarSobreposicaoTermino(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, profissionalID string,
	inicio, fim time.Time,
) (*detalheSobreposicao, error) {
	const query = `
SELECT a.data_hora_inicio
FROM agendamentos a
WHERE a.estabelecimento_id = $1
  AND a.profissional_id = $2
  AND a.status IN ('AGENDADO', 'CONFIRMADO')
  AND a.data_hora_inicio >= $3
  AND a.data_hora_inicio < $4
ORDER BY a.data_hora_inicio ASC
LIMIT 1
FOR UPDATE OF a
`
	var proximoInicio time.Time
	err := tx.GetContext(ctx, &proximoInicio, query, estabelecimentoID, profissionalID, inicio, fim)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("verificar sobreposição de término: %w", err)
	}

	minutos := int(fim.Sub(proximoInicio).Minutes())
	if minutos < 1 {
		minutos = 1
	}

	return &detalheSobreposicao{
		ProximoInicio:    proximoInicio,
		MinutosInvadidos: minutos,
	}, nil
}

// intervaloTotalmenteLivre garante que o procedimento cabe sem nenhuma sobreposição.
func (s *AgendaService) intervaloTotalmenteLivre(
	ctx context.Context,
	tx *sqlx.Tx,
	estabelecimentoID, profissionalID string,
	inicio, fim time.Time,
	excluirAgendamentoID string,
) error {
	const query = `
SELECT a.id
FROM agendamentos a
WHERE a.estabelecimento_id = $1
  AND a.profissional_id = $2
  AND a.status IN ('AGENDADO', 'CONFIRMADO')
  AND a.id <> $5
  AND a.data_hora_inicio < $4
  AND a.data_hora_fim > $3
LIMIT 1
FOR UPDATE OF a
`
	var conflito string
	err := tx.GetContext(ctx, &conflito, query, estabelecimentoID, profissionalID, inicio, fim, excluirAgendamentoID)
	if err == nil {
		return ErrHorarioIndisponivel
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("verificar intervalo livre: %w", err)
	}
	return nil
}
