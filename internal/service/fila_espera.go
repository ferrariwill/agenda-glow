package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

var ErrFilaNaoEncontrada = errors.New("entrada da fila não encontrada")

// FilaEsperaService gerencia a fila de clientes que aceitam adiantar horário.
type FilaEsperaService struct {
	db *sqlx.DB
}

func NewFilaEsperaService(db *sqlx.DB) *FilaEsperaService {
	return &FilaEsperaService{db: db}
}

type FilaEsperaEntry struct {
	ID               string  `db:"id" json:"id"`
	EstabelecimentoID string `db:"estabelecimento_id" json:"tenant_id"`
	ProfissionalID   string  `db:"profissional_id" json:"profissional_id"`
	ClienteNome      string  `db:"cliente_nome" json:"cliente_nome"`
	ClienteTelefone  string  `db:"cliente_telefone" json:"cliente_telefone"`
	Status           string  `db:"status" json:"status"`
	AgendamentoID    *string `db:"agendamento_id" json:"agendamento_id,omitempty"`
	DataDesejada     *string `db:"data_desejada" json:"data,omitempty"`
	HoraDesejada     *string `db:"hora_desejada" json:"hora_desejada,omitempty"`
}

type InscreverFilaInput struct {
	EstabelecimentoID string
	ProfissionalID    string
	ClienteNome       string
	ClienteTelefone   string
	AgendamentoID     string
	Data              string
	Hora              string
}

func normalizeTelefoneFila(tel string) string {
	var b strings.Builder
	for _, r := range tel {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Inscrever remove entradas anteriores do mesmo telefone/profissional e cria nova inscrição AGUARDANDO.
func (s *FilaEsperaService) Inscrever(ctx context.Context, in InscreverFilaInput) (string, error) {
	tel := normalizeTelefoneFila(in.ClienteTelefone)
	if tel == "" {
		return "", fmt.Errorf("telefone inválido")
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const del = `
DELETE FROM fila_espera
WHERE profissional_id = $1
  AND cliente_telefone = $2
  AND status = 'AGUARDANDO'
`
	if _, err := tx.ExecContext(ctx, del, in.ProfissionalID, tel); err != nil {
		return "", fmt.Errorf("limpar fila anterior: %w", err)
	}

	var dataPtr, horaPtr *string
	if strings.TrimSpace(in.Data) != "" {
		d := strings.TrimSpace(in.Data)
		dataPtr = &d
	}
	if strings.TrimSpace(in.Hora) != "" {
		h := strings.TrimSpace(in.Hora)
		if len(h) == 5 {
			h += ":00"
		}
		horaPtr = &h
	}

	var agID *string
	if strings.TrimSpace(in.AgendamentoID) != "" {
		id := strings.TrimSpace(in.AgendamentoID)
		agID = &id
	}

	const insert = `
INSERT INTO fila_espera (
    estabelecimento_id, profissional_id, cliente_nome, cliente_telefone,
    status, agendamento_id, data_desejada, hora_desejada
) VALUES ($1, $2, $3, $4, 'AGUARDANDO', $5, $6::DATE, $7::TIME)
RETURNING id
`
	var id string
	if err := tx.GetContext(ctx, &id, insert,
		in.EstabelecimentoID, in.ProfissionalID, strings.TrimSpace(in.ClienteNome), tel,
		agID, dataPtr, horaPtr,
	); err != nil {
		return "", fmt.Errorf("inscrever na fila: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar inscrição na fila: %w", err)
	}
	return id, nil
}

// BuscarAguardando retorna a entrada AGUARDANDO da profissional, se existir.
func (s *FilaEsperaService) BuscarAguardando(ctx context.Context, profissionalID string) (*FilaEsperaEntry, error) {
	const query = `
SELECT
    id, estabelecimento_id, profissional_id, cliente_nome, cliente_telefone,
    status, agendamento_id,
    data_desejada::TEXT, hora_desejada::TEXT
FROM fila_espera
WHERE profissional_id = $1 AND status = 'AGUARDANDO'
ORDER BY criado_em ASC
LIMIT 1
`
	var row struct {
		ID               string         `db:"id"`
		EstabelecimentoID string        `db:"estabelecimento_id"`
		ProfissionalID   string         `db:"profissional_id"`
		ClienteNome      string         `db:"cliente_nome"`
		ClienteTelefone  string         `db:"cliente_telefone"`
		Status           string         `db:"status"`
		AgendamentoID    sql.NullString `db:"agendamento_id"`
		DataDesejada     sql.NullString `db:"data_desejada"`
		HoraDesejada     sql.NullString `db:"hora_desejada"`
	}
	if err := s.db.GetContext(ctx, &row, query, profissionalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("buscar fila: %w", err)
	}
	entry := &FilaEsperaEntry{
		ID:               row.ID,
		EstabelecimentoID: row.EstabelecimentoID,
		ProfissionalID:   row.ProfissionalID,
		ClienteNome:      row.ClienteNome,
		ClienteTelefone:  row.ClienteTelefone,
		Status:           row.Status,
	}
	if row.AgendamentoID.Valid {
		entry.AgendamentoID = &row.AgendamentoID.String
	}
	if row.DataDesejada.Valid {
		entry.DataDesejada = &row.DataDesejada.String
	}
	if row.HoraDesejada.Valid {
		h := row.HoraDesejada.String
		if len(h) >= 5 {
			h = h[:5]
		}
		entry.HoraDesejada = &h
	}
	return entry, nil
}

// MarcarNotificada altera status para NOTIFICADO.
func (s *FilaEsperaService) MarcarNotificada(ctx context.Context, id string) error {
	const q = `UPDATE fila_espera SET status = 'NOTIFICADO' WHERE id = $1 AND status = 'AGUARDANDO'`
	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("marcar fila notificada: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrFilaNaoEncontrada
	}
	return nil
}

func (s *FilaEsperaService) ListByEstablishment(ctx context.Context, establishmentID string) ([]FilaEsperaEntry, error) {
	const query = `
SELECT
    id, estabelecimento_id, profissional_id, cliente_nome, cliente_telefone,
    status, agendamento_id,
    data_desejada::TEXT, hora_desejada::TEXT
FROM fila_espera
WHERE estabelecimento_id = $1
  AND status IN ('AGUARDANDO', 'NOTIFICADO')
  AND criado_em >= NOW() - INTERVAL '7 days'
ORDER BY criado_em DESC
`
	rows, err := s.db.QueryxContext(ctx, query, establishmentID)
	if err != nil {
		return nil, fmt.Errorf("listar fila: %w", err)
	}
	defer rows.Close()

	var out []FilaEsperaEntry
	for rows.Next() {
		var row struct {
			ID               string         `db:"id"`
			EstabelecimentoID string        `db:"estabelecimento_id"`
			ProfissionalID   string         `db:"profissional_id"`
			ClienteNome      string         `db:"cliente_nome"`
			ClienteTelefone  string         `db:"cliente_telefone"`
			Status           string         `db:"status"`
			AgendamentoID    sql.NullString `db:"agendamento_id"`
			DataDesejada     sql.NullString `db:"data_desejada"`
			HoraDesejada     sql.NullString `db:"hora_desejada"`
		}
		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}
		e := FilaEsperaEntry{
			ID:               row.ID,
			EstabelecimentoID: row.EstabelecimentoID,
			ProfissionalID:   row.ProfissionalID,
			ClienteNome:      row.ClienteNome,
			ClienteTelefone:  row.ClienteTelefone,
			Status:           row.Status,
		}
		if row.AgendamentoID.Valid {
			e.AgendamentoID = &row.AgendamentoID.String
		}
		if row.DataDesejada.Valid {
			e.DataDesejada = &row.DataDesejada.String
		}
		if row.HoraDesejada.Valid {
			h := row.HoraDesejada.String
			if len(h) >= 5 {
				h = h[:5]
			}
			e.HoraDesejada = &h
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// parseHoraFila removido — hora normalizada no handler HTTP.
