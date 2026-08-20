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

var (
	ErrPlanLimitExceeded     = errors.New("plan_limit_exceeded")
	ErrDataNascimentoFutura  = errors.New("birthdate_in_future")
	ErrDataNascimentoInvalida = errors.New("invalid_birthdate")
)

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
	EhDona              bool    `db:"eh_dona" json:"eh_dona"`
	FotoURL             *string `db:"foto_url" json:"foto_url,omitempty"`
	DataNascimento      *string `db:"data_nascimento" json:"data_nascimento,omitempty"`
}

type planoLimite struct {
	LimiteProfissionais int `db:"limite_profissionais"`
}

// CreateProfessional cadastra profissional respeitando o limite do plano SaaS contratado.
// dataNascimento e fotoURL são opcionais (nil = permanece null). Data futura → ErrDataNascimentoFutura.
func (s *ProfissionalService) CreateProfessional(
	ctx context.Context,
	establishmentID, nome, especialidadeID string,
	comissao float64,
	dataNascimento *string,
	fotoURL *string,
) (string, error) {
	nome = strings.TrimSpace(nome)
	especialidadeID = strings.TrimSpace(especialidadeID)
	if nome == "" || especialidadeID == "" {
		return "", fmt.Errorf("nome e especialidade são obrigatórios")
	}
	if comissao < 0 || comissao > 100 {
		return "", fmt.Errorf("comissão deve estar entre 0 e 100")
	}

	dataSQL, err := normalizeOptionalDataNascimento(dataNascimento)
	if err != nil {
		return "", err
	}
	fotoSQL := normalizeOptionalFotoURL(fotoURL)

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
INSERT INTO profissionais (
    estabelecimento_id, nome, especialidade_id, comissao_porcentagem, ativo,
    data_nascimento, foto_url
)
VALUES ($1, $2, $3, $4, TRUE, $5::DATE, $6)
RETURNING id
`
	var id string
	if err := tx.GetContext(ctx, &id, insert, establishmentID, nome, especialidadeID, comissao, dataSQL, fotoSQL); err != nil {
		return "", fmt.Errorf("cadastrar profissional: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar transação: %w", err)
	}

	return id, nil
}

// UpdateProfessional altera nome, especialidade, comissão e status da profissional.
// dataNascimento/fotoURL nil = não altera; string válida = define; data futura → ErrDataNascimentoFutura.
func (s *ProfissionalService) UpdateProfessional(
	ctx context.Context,
	establishmentID, professionalID string,
	nome, especialidadeID string,
	comissao float64,
	ativo bool,
	dataNascimento *string,
	fotoURL *string,
) error {
	nome = strings.TrimSpace(nome)
	especialidadeID = strings.TrimSpace(especialidadeID)
	if nome == "" || especialidadeID == "" {
		return fmt.Errorf("nome e especialidade são obrigatórios")
	}
	if comissao < 0 || comissao > 100 {
		return fmt.Errorf("comissão deve estar entre 0 e 100")
	}

	setData := false
	var dataSQL interface{}
	if dataNascimento != nil {
		normalized, err := normalizeOptionalDataNascimento(dataNascimento)
		if err != nil {
			return err
		}
		setData = true
		dataSQL = normalized
	}
	setFoto := false
	var fotoSQL interface{}
	if fotoURL != nil {
		setFoto = true
		fotoSQL = normalizeOptionalFotoURL(fotoURL)
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
    pendente_aprovacao = CASE WHEN $6 = TRUE THEN FALSE ELSE pendente_aprovacao END,
    data_nascimento = CASE WHEN $7 THEN $8::DATE ELSE data_nascimento END,
    foto_url = CASE WHEN $9 THEN $10 ELSE foto_url END
WHERE id = $1 AND estabelecimento_id = $2
`
	if _, err := tx.ExecContext(
		ctx, update,
		professionalID, establishmentID, nome, especialidadeID, comissao, ativo,
		setData, dataSQL, setFoto, fotoSQL,
	); err != nil {
		return fmt.Errorf("atualizar profissional: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar atualização: %w", err)
	}

	return nil
}

// SetFotoURL atualiza foto_url somente se id + estabelecimento_id baterem.
func (s *ProfissionalService) SetFotoURL(ctx context.Context, establishmentID, professionalID, fotoURL string) error {
	professionalID = strings.TrimSpace(professionalID)
	establishmentID = strings.TrimSpace(establishmentID)
	fotoURL = strings.TrimSpace(fotoURL)
	if professionalID == "" || establishmentID == "" || fotoURL == "" {
		return ErrProfissionalNaoEncontrado
	}

	const q = `
UPDATE profissionais
SET foto_url = $3
WHERE id = $1 AND estabelecimento_id = $2
`
	res, err := s.db.ExecContext(ctx, q, professionalID, establishmentID, fotoURL)
	if err != nil {
		return fmt.Errorf("atualizar foto_url: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar atualização de foto: %w", err)
	}
	if n == 0 {
		return ErrProfissionalNaoEncontrado
	}
	return nil
}

// normalizeOptionalDataNascimento: nil ou string vazia → null SQL; válida YYYY-MM-DD; futura → ErrDataNascimentoFutura.
func normalizeOptionalDataNascimento(raw *string) (interface{}, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	if err := ValidarDataNascimento(trimmed); err != nil {
		return nil, err
	}
	return trimmed, nil
}

func normalizeOptionalFotoURL(raw *string) interface{} {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// ValidarDataNascimento exige YYYY-MM-DD e rejeita datas civis futuras em America/Sao_Paulo.
func ValidarDataNascimento(raw string) error {
	raw = strings.TrimSpace(raw)
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("America/Sao_Paulo", -3*60*60)
	}
	parsed, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return ErrDataNascimentoInvalida
	}
	agora := time.Now().In(loc)
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, loc)
	if parsed.After(hoje) {
		return ErrDataNascimentoFutura
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
       p.comissao_porcentagem, p.ativo, p.pendente_aprovacao, p.eh_dona,
       NULLIF(TRIM(p.foto_url), '') AS foto_url,
       CASE WHEN p.data_nascimento IS NULL THEN NULL
            ELSE to_char(p.data_nascimento, 'YYYY-MM-DD') END AS data_nascimento
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

