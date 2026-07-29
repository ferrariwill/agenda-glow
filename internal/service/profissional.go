package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrPlanLimitExceeded = errors.New("plan_limit_exceeded")

type ProfissionalService struct {
	db *sqlx.DB
}

func NewProfissionalService(db *sqlx.DB) *ProfissionalService {
	return &ProfissionalService{db: db}
}

type Profissional struct {
	ID                  string  `db:"id" json:"id"`
	Nome                string  `db:"nome" json:"nome"`
	EspecialidadeID     string  `db:"especialidade_id" json:"especialidade_id"`
	Especialidade       string  `db:"especialidade_nome" json:"especialidade"`
	ComissaoPorcentagem float64 `db:"comissao_porcentagem" json:"comissao_porcentagem"`
	Ativo               bool    `db:"ativo" json:"ativo"`
	PendenteAprovacao   bool    `db:"pendente_aprovacao" json:"pendente_aprovacao"`
}

type planoLimite struct {
	LimiteProfissionais int `db:"limite_profissionais"`
}

// CreateProfessional cadastra profissional respeitando o limite do plano SaaS contratado.
func (s *ProfissionalService) CreateProfessional(
	ctx context.Context,
	establishmentID, nome, especialidadeID string,
	comissao float64,
) (string, error) {
	nome = strings.TrimSpace(nome)
	especialidadeID = strings.TrimSpace(especialidadeID)
	if nome == "" || especialidadeID == "" {
		return "", fmt.Errorf("nome e especialidade são obrigatórios")
	}
	if comissao < 0 || comissao > 100 {
		return "", fmt.Errorf("comissão deve estar entre 0 e 100")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lockEstabelecimento = `SELECT id FROM estabelecimentos WHERE id = $1 FOR UPDATE`
	var estabelecimentoID string
	if err := tx.GetContext(ctx, &estabelecimentoID, lockEstabelecimento, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrEstabelecimentoNaoEncontrado
		}
		return "", fmt.Errorf("bloquear estabelecimento: %w", err)
	}

	limite, err := s.buscarLimitePlano(ctx, tx, establishmentID)
	if err != nil {
		return "", err
	}

	const contarAtivos = `
SELECT COUNT(*)::INTEGER
FROM profissionais
WHERE estabelecimento_id = $1 AND ativo = TRUE AND pendente_aprovacao = FALSE
`
	var totalAtivos int
	if err := tx.GetContext(ctx, &totalAtivos, contarAtivos, establishmentID); err != nil {
		return "", fmt.Errorf("contar profissionais ativos: %w", err)
	}

	if totalAtivos >= limite.LimiteProfissionais {
		return "", ErrPlanLimitExceeded
	}

	if err := s.validarEspecialidadeAtiva(ctx, tx, establishmentID, especialidadeID); err != nil {
		return "", err
	}

	const insert = `
INSERT INTO profissionais (estabelecimento_id, nome, especialidade_id, comissao_porcentagem, ativo)
VALUES ($1, $2, $3, $4, TRUE)
RETURNING id
`
	var id string
	if err := tx.GetContext(ctx, &id, insert, establishmentID, nome, especialidadeID, comissao); err != nil {
		return "", fmt.Errorf("cadastrar profissional: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar transação: %w", err)
	}

	return id, nil
}

// UpdateProfessional altera nome, especialidade, comissão e status da profissional.
func (s *ProfissionalService) UpdateProfessional(
	ctx context.Context,
	establishmentID, professionalID string,
	nome, especialidadeID string,
	comissao float64,
	ativo bool,
) error {
	nome = strings.TrimSpace(nome)
	especialidadeID = strings.TrimSpace(especialidadeID)
	if nome == "" || especialidadeID == "" {
		return fmt.Errorf("nome e especialidade são obrigatórios")
	}
	if comissao < 0 || comissao > 100 {
		return fmt.Errorf("comissão deve estar entre 0 e 100")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lockProf = `
SELECT id, ativo FROM profissionais
WHERE id = $1 AND estabelecimento_id = $2
FOR UPDATE
`
	var atual struct {
		ID    string `db:"id"`
		Ativo bool   `db:"ativo"`
	}
	if err := tx.GetContext(ctx, &atual, lockProf, professionalID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfissionalNaoEncontrado
		}
		return fmt.Errorf("bloquear profissional: %w", err)
	}

	if ativo && !atual.Ativo {
		const lockEstabelecimento = `SELECT id FROM estabelecimentos WHERE id = $1 FOR UPDATE`
		var estabID string
		if err := tx.GetContext(ctx, &estabID, lockEstabelecimento, establishmentID); err != nil {
			return fmt.Errorf("bloquear estabelecimento: %w", err)
		}

		limite, err := s.buscarLimitePlano(ctx, tx, establishmentID)
		if err != nil {
			return err
		}

		const contarAtivos = `
SELECT COUNT(*)::INTEGER FROM profissionais
WHERE estabelecimento_id = $1 AND ativo = TRUE AND pendente_aprovacao = FALSE
`
		var totalAtivos int
		if err := tx.GetContext(ctx, &totalAtivos, contarAtivos, establishmentID); err != nil {
			return fmt.Errorf("contar profissionais ativos: %w", err)
		}
		if totalAtivos >= limite.LimiteProfissionais {
			return ErrPlanLimitExceeded
		}
	}

	if ativo {
		if err := s.validarEspecialidadeAtiva(ctx, tx, establishmentID, especialidadeID); err != nil {
			return err
		}
	} else if err := s.validarEspecialidadeDoEstabelecimento(ctx, tx, establishmentID, especialidadeID); err != nil {
		return err
	}

	const update = `
UPDATE profissionais
SET nome = $3, especialidade_id = $4, comissao_porcentagem = $5, ativo = $6,
    pendente_aprovacao = CASE WHEN $6 = TRUE THEN FALSE ELSE pendente_aprovacao END
WHERE id = $1 AND estabelecimento_id = $2
`
	if _, err := tx.ExecContext(ctx, update, professionalID, establishmentID, nome, especialidadeID, comissao, ativo); err != nil {
		return fmt.Errorf("atualizar profissional: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar atualização: %w", err)
	}

	return nil
}

func (s *ProfissionalService) buscarLimitePlano(ctx context.Context, tx *sqlx.Tx, establishmentID string) (*planoLimite, error) {
	const query = `
SELECT ps.limite_profissionais
FROM assinaturas_estabelecimentos ae
INNER JOIN planos_saas ps ON ps.id = ae.plano_id
WHERE ae.estabelecimento_id = $1
FOR UPDATE OF ae
`
	var limite planoLimite
	if err := tx.GetContext(ctx, &limite, query, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanoSaasNaoEncontrado
		}
		return nil, fmt.Errorf("consultar limite do plano: %w", err)
	}
	return &limite, nil
}

// ListProfessionals retorna profissionais ativos e inativos do estabelecimento.
func (s *ProfissionalService) ListProfessionals(ctx context.Context, establishmentID string) ([]Profissional, error) {
	const query = `
SELECT p.id, p.nome, p.especialidade_id, e.nome AS especialidade_nome,
       p.comissao_porcentagem, p.ativo, p.pendente_aprovacao
FROM profissionais p
INNER JOIN especialidades e ON e.id = p.especialidade_id AND e.estabelecimento_id = p.estabelecimento_id
WHERE p.estabelecimento_id = $1
ORDER BY p.nome ASC
`
	var lista []Profissional
	if err := s.db.SelectContext(ctx, &lista, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar profissionais: %w", err)
	}
	if lista == nil {
		lista = []Profissional{}
	}
	return lista, nil
}

func (s *ProfissionalService) validarEspecialidadeAtiva(ctx context.Context, tx *sqlx.Tx, establishmentID, especialidadeID string) error {
	const query = `
SELECT id FROM especialidades
WHERE id = $1 AND estabelecimento_id = $2 AND ativo = TRUE
`
	var id string
	if err := tx.GetContext(ctx, &id, query, especialidadeID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEspecialidadeNaoEncontrada
		}
		return fmt.Errorf("validar especialidade: %w", err)
	}
	return nil
}

func (s *ProfissionalService) validarEspecialidadeDoEstabelecimento(ctx context.Context, tx *sqlx.Tx, establishmentID, especialidadeID string) error {
	const query = `SELECT id FROM especialidades WHERE id = $1 AND estabelecimento_id = $2`
	var id string
	if err := tx.GetContext(ctx, &id, query, especialidadeID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEspecialidadeNaoEncontrada
		}
		return fmt.Errorf("validar especialidade: %w", err)
	}
	return nil
}

// ExpedienteInput representa a escala de um dia da semana (0=Domingo … 6=Sábado).
type ExpedienteInput struct {
	DiaSemana      int     `json:"dia_semana"`
	HorarioEntrada string  `json:"horario_entrada"` // HH:MM ou HH:MM:SS
	InicioAlmoco   *string `json:"inicio_almoco,omitempty"`
	FimAlmoco      *string `json:"fim_almoco,omitempty"`
	HorarioSaida   string  `json:"horario_saida"`
}

// ExpedienteProfissional é o registro persistido da jornada de trabalho.
type ExpedienteProfissional struct {
	ID              string  `db:"id" json:"id"`
	ProfissionalID  string  `db:"profissional_id" json:"profissional_id"`
	DiaSemana       int     `db:"dia_semana" json:"dia_semana"`
	HorarioEntrada  string  `db:"horario_entrada" json:"horario_entrada"`
	InicioAlmoco    *string `db:"inicio_almoco" json:"inicio_almoco,omitempty"`
	FimAlmoco       *string `db:"fim_almoco" json:"fim_almoco,omitempty"`
	HorarioSaida    string  `db:"horario_saida" json:"horario_saida"`
}

var (
	ErrExpedienteInvalido = errors.New("expediente inválido")
)

// SetProfessionalHours substitui atomicamente a grade de horários da profissional.
func (s *ProfissionalService) SetProfessionalHours(
	ctx context.Context,
	professionalID string,
	hours []ExpedienteInput,
) error {
	professionalID = strings.TrimSpace(professionalID)
	if professionalID == "" {
		return ErrProfissionalNaoEncontrado
	}

	if err := validarExpedientesInput(hours); err != nil {
		return err
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := s.lockProfissional(ctx, tx, professionalID); err != nil {
		return err
	}

	const deleteExpedientes = `DELETE FROM expedientes_profissionais WHERE profissional_id = $1`
	if _, err := tx.ExecContext(ctx, deleteExpedientes, professionalID); err != nil {
		return fmt.Errorf("limpar expedientes anteriores: %w", err)
	}

	const insert = `
INSERT INTO expedientes_profissionais (
    profissional_id,
    dia_semana,
    horario_entrada,
    inicio_almoco,
    fim_almoco,
    horario_saida
) VALUES ($1, $2, $3::TIME, $4::TIME, $5::TIME, $6::TIME)
`
	for _, h := range hours {
		entrada, err := normalizarHorarioSQL(h.HorarioEntrada)
		if err != nil {
			return fmt.Errorf("horario_entrada dia %d: %w", h.DiaSemana, err)
		}
		saida, err := normalizarHorarioSQL(h.HorarioSaida)
		if err != nil {
			return fmt.Errorf("horario_saida dia %d: %w", h.DiaSemana, err)
		}

		var inicioAlmoco, fimAlmoco interface{}
		if h.InicioAlmoco != nil && h.FimAlmoco != nil {
			inicio, err := normalizarHorarioSQL(*h.InicioAlmoco)
			if err != nil {
				return fmt.Errorf("inicio_almoco dia %d: %w", h.DiaSemana, err)
			}
			fim, err := normalizarHorarioSQL(*h.FimAlmoco)
			if err != nil {
				return fmt.Errorf("fim_almoco dia %d: %w", h.DiaSemana, err)
			}
			inicioAlmoco = inicio
			fimAlmoco = fim
		}

		if _, err := tx.ExecContext(
			ctx, insert,
			professionalID,
			h.DiaSemana,
			entrada,
			inicioAlmoco,
			fimAlmoco,
			saida,
		); err != nil {
			return fmt.Errorf("gravar expediente dia %d: %w", h.DiaSemana, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar expedientes: %w", err)
	}

	return nil
}

func (s *ProfissionalService) lockProfissional(ctx context.Context, tx *sqlx.Tx, professionalID string) error {
	const query = `SELECT id FROM profissionais WHERE id = $1 FOR UPDATE`
	var id string
	if err := tx.GetContext(ctx, &id, query, professionalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrProfissionalNaoEncontrado
		}
		return fmt.Errorf("bloquear profissional: %w", err)
	}
	return nil
}

func validarExpedientesInput(hours []ExpedienteInput) error {
	if len(hours) == 0 {
		return nil
	}

	vistos := make(map[int]struct{}, len(hours))
	for _, h := range hours {
		if h.DiaSemana < 0 || h.DiaSemana > 6 {
			return fmt.Errorf("%w: dia_semana deve estar entre 0 e 6", ErrExpedienteInvalido)
		}
		if _, dup := vistos[h.DiaSemana]; dup {
			return fmt.Errorf("%w: dia_semana %d duplicado na requisição", ErrExpedienteInvalido, h.DiaSemana)
		}
		vistos[h.DiaSemana] = struct{}{}

		entrada, err := parseHorarioExpediente(h.HorarioEntrada)
		if err != nil {
			return fmt.Errorf("%w: horario_entrada inválido no dia %d", ErrExpedienteInvalido, h.DiaSemana)
		}
		saida, err := parseHorarioExpediente(h.HorarioSaida)
		if err != nil {
			return fmt.Errorf("%w: horario_saida inválido no dia %d", ErrExpedienteInvalido, h.DiaSemana)
		}
		if !saida.After(entrada) {
			return fmt.Errorf("%w: horario_saida deve ser posterior à entrada no dia %d", ErrExpedienteInvalido, h.DiaSemana)
		}

		temInicio := h.InicioAlmoco != nil && strings.TrimSpace(*h.InicioAlmoco) != ""
		temFim := h.FimAlmoco != nil && strings.TrimSpace(*h.FimAlmoco) != ""
		switch {
		case temInicio && !temFim, !temInicio && temFim:
			return fmt.Errorf("%w: informe inicio_almoco e fim_almoco juntos no dia %d", ErrExpedienteInvalido, h.DiaSemana)
		case temInicio && temFim:
			inicioAlmoco, err := parseHorarioExpediente(*h.InicioAlmoco)
			if err != nil {
				return fmt.Errorf("%w: inicio_almoco inválido no dia %d", ErrExpedienteInvalido, h.DiaSemana)
			}
			fimAlmoco, err := parseHorarioExpediente(*h.FimAlmoco)
			if err != nil {
				return fmt.Errorf("%w: fim_almoco inválido no dia %d", ErrExpedienteInvalido, h.DiaSemana)
			}
			if !fimAlmoco.After(inicioAlmoco) {
				return fmt.Errorf("%w: fim_almoco deve ser posterior ao inicio_almoco no dia %d", ErrExpedienteInvalido, h.DiaSemana)
			}
			if inicioAlmoco.Before(entrada) || fimAlmoco.After(saida) {
				return fmt.Errorf("%w: intervalo de almoço deve estar dentro da jornada no dia %d", ErrExpedienteInvalido, h.DiaSemana)
			}
		}
	}

	return nil
}

func parseHorarioExpediente(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("formato de horário inválido: %q", raw)
}

func normalizarHorarioSQL(raw string) (string, error) {
	t, err := parseHorarioExpediente(raw)
	if err != nil {
		return "", err
	}
	return t.Format("15:04:05"), nil
}

