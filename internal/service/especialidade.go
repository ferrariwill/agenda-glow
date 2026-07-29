package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrEspecialidadeNaoEncontrada  = errors.New("especialidade não encontrada")
	ErrEspecialidadeNomeDuplicado = errors.New("já existe especialidade com este nome")
	ErrEspecialidadeInativa       = errors.New("especialidade inativa")
)

type Especialidade struct {
	ID    string `db:"id" json:"id"`
	Nome  string `db:"nome" json:"nome"`
	Ativo bool   `db:"ativo" json:"ativo"`
}

type EspecialidadeService struct {
	db *sqlx.DB
}

func NewEspecialidadeService(db *sqlx.DB) *EspecialidadeService {
	return &EspecialidadeService{db: db}
}

func (s *EspecialidadeService) ListEspecialidades(ctx context.Context, establishmentID string) ([]Especialidade, error) {
	const query = `
SELECT id, nome, ativo
FROM especialidades
WHERE estabelecimento_id = $1
ORDER BY nome ASC
`
	var lista []Especialidade
	if err := s.db.SelectContext(ctx, &lista, query, establishmentID); err != nil {
		return nil, fmt.Errorf("listar especialidades: %w", err)
	}
	if lista == nil {
		lista = []Especialidade{}
	}
	return lista, nil
}

func (s *EspecialidadeService) CreateEspecialidade(ctx context.Context, establishmentID, nome string) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", fmt.Errorf("nome da especialidade é obrigatório")
	}

	const insert = `
INSERT INTO especialidades (estabelecimento_id, nome, ativo)
VALUES ($1, $2, TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, establishmentID, nome); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", ErrEspecialidadeNomeDuplicado
		}
		return "", fmt.Errorf("criar especialidade: %w", err)
	}
	return id, nil
}

func (s *EspecialidadeService) UpdateEspecialidade(
	ctx context.Context,
	establishmentID, especialidadeID, nome string,
	ativo bool,
) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return fmt.Errorf("nome da especialidade é obrigatório")
	}

	const update = `
UPDATE especialidades
SET nome = $3, ativo = $4
WHERE id = $1 AND estabelecimento_id = $2
`
	result, err := s.db.ExecContext(ctx, update, especialidadeID, establishmentID, nome, ativo)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return ErrEspecialidadeNomeDuplicado
		}
		return fmt.Errorf("atualizar especialidade: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEspecialidadeNaoEncontrada
	}
	return nil
}

func (s *EspecialidadeService) BuscarPorID(ctx context.Context, establishmentID, especialidadeID string) (*Especialidade, error) {
	const query = `
SELECT id, nome, ativo
FROM especialidades
WHERE id = $1 AND estabelecimento_id = $2
`
	var esp Especialidade
	if err := s.db.GetContext(ctx, &esp, query, especialidadeID, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEspecialidadeNaoEncontrada
		}
		return nil, fmt.Errorf("buscar especialidade: %w", err)
	}
	return &esp, nil
}

func (s *EspecialidadeService) ValidarEspecialidadeAtiva(ctx context.Context, establishmentID, especialidadeID string) error {
	esp, err := s.BuscarPorID(ctx, establishmentID, especialidadeID)
	if err != nil {
		return err
	}
	if !esp.Ativo {
		return ErrEspecialidadeInativa
	}
	return nil
}
