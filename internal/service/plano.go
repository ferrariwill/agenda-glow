package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrPlanoSaasNaoEncontrado     = errors.New("plano SaaS não encontrado")
	ErrPlanoSaasInativo           = errors.New("plano SaaS inativo")
	ErrMesesContratacaoInvalidos  = errors.New("meses de contratação inválidos")
	ErrLimiteProfissionaisInvalido = errors.New("limite de profissionais inválido")
)

type PlanoSaasService struct {
	db *sqlx.DB
}

func NewPlanoSaasService(db *sqlx.DB) *PlanoSaasService {
	return &PlanoSaasService{db: db}
}

type PlanoSaas struct {
	ID                   string  `db:"id" json:"id"`
	Nome                 string  `db:"nome" json:"nome"`
	PrecoMensal          float64 `db:"preco_mensal" json:"preco_mensal"`
	LimiteProfissionais  int     `db:"limite_profissionais" json:"limite_profissionais"`
	Ativo                bool    `db:"ativo" json:"ativo"`
}

type assinaturaExistente struct {
	ID              string    `db:"id"`
	DataVencimento  time.Time `db:"data_vencimento"`
}

// CreateSaasPlan cadastra um novo plano corporativo disponível na plataforma.
func (s *PlanoSaasService) CreateSaasPlan(
	ctx context.Context,
	name string,
	price float64,
	professionalLimit int,
) (string, error) {
	nome := strings.TrimSpace(name)
	if nome == "" {
		return "", fmt.Errorf("nome do plano é obrigatório")
	}
	if price < 0 {
		return "", fmt.Errorf("preço mensal inválido")
	}
	if professionalLimit <= 0 {
		return "", ErrLimiteProfissionaisInvalido
	}

	const insert = `
INSERT INTO planos_saas (nome, preco_mensal, limite_profissionais, ativo)
VALUES ($1, $2, $3, TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, nome, price, professionalLimit); err != nil {
		return "", fmt.Errorf("criar plano SaaS: %w", err)
	}

	return id, nil
}

// ListSaasPlans retorna todos os planos corporativos cadastrados.
func (s *PlanoSaasService) ListSaasPlans(ctx context.Context) ([]PlanoSaas, error) {
	const query = `
SELECT id, nome, preco_mensal, limite_profissionais, ativo
FROM planos_saas
ORDER BY preco_mensal ASC, nome ASC
`
	var planos []PlanoSaas
	if err := s.db.SelectContext(ctx, &planos, query); err != nil {
		return nil, fmt.Errorf("listar planos SaaS: %w", err)
	}
	if planos == nil {
		planos = []PlanoSaas{}
	}
	return planos, nil
}

// AssignPlanToEstablishment vincula ou renova a assinatura SaaS de um estabelecimento.
func (s *PlanoSaasService) AssignPlanToEstablishment(
	ctx context.Context,
	establishmentID, planID string,
	monthsDuration int,
) error {
	if monthsDuration <= 0 {
		return ErrMesesContratacaoInvalidos
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := s.validarEstabelecimento(ctx, tx, establishmentID); err != nil {
		return err
	}

	if err := s.validarPlanoSaas(ctx, tx, planID); err != nil {
		return err
	}

	baseVencimento, err := s.resolverBaseVencimento(ctx, tx, establishmentID)
	if err != nil {
		return err
	}

	novaDataVencimento := baseVencimento.AddDate(0, monthsDuration, 0)

	const upsert = `
INSERT INTO assinaturas_estabelecimentos (
    estabelecimento_id,
    plano_id,
    status,
    data_vencimento,
    atualizado_em
) VALUES ($1, $2, 'ATIVO', $3, NOW())
ON CONFLICT (estabelecimento_id) DO UPDATE SET
    plano_id = EXCLUDED.plano_id,
    status = 'ATIVO',
    data_vencimento = EXCLUDED.data_vencimento,
    atualizado_em = NOW()
`
	if _, err := tx.ExecContext(ctx, upsert, establishmentID, planID, novaDataVencimento); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return fmt.Errorf("referência inválida: %w", err)
		}
		return fmt.Errorf("atribuir plano ao estabelecimento: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar transação: %w", err)
	}

	return nil
}

// BuscarPlanoSaasPorID retorna plano para partial HTMX após cadastro.
func (s *PlanoSaasService) BuscarPlanoSaasPorID(ctx context.Context, planID string) (*PlanoSaas, error) {
	const query = `
SELECT id, nome, preco_mensal, limite_profissionais, ativo
FROM planos_saas WHERE id = $1
`
	var plano PlanoSaas
	if err := s.db.GetContext(ctx, &plano, query, planID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPlanoSaasNaoEncontrado
		}
		return nil, fmt.Errorf("buscar plano SaaS: %w", err)
	}
	return &plano, nil
}

// SetSubscriptionStatus altera o status da assinatura corporativa do salão.
func (s *PlanoSaasService) SetSubscriptionStatus(ctx context.Context, establishmentID, status string) error {
	switch status {
	case "ATIVO", "SUSPENSO", "PAGAMENTO_PENDENTE":
	default:
		return fmt.Errorf("status de assinatura inválido")
	}

	const update = `
UPDATE assinaturas_estabelecimentos
SET status = $2, atualizado_em = NOW()
WHERE estabelecimento_id = $1
`
	result, err := s.db.ExecContext(ctx, update, establishmentID, status)
	if err != nil {
		return fmt.Errorf("atualizar status da assinatura: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrEstabelecimentoNaoEncontrado
	}
	return nil
}

// RenewEstablishmentSubscription renova assinatura por N meses usando o plano atual.
func (s *PlanoSaasService) RenewEstablishmentSubscription(ctx context.Context, establishmentID string, months int) error {
	const query = `SELECT plano_id FROM assinaturas_estabelecimentos WHERE estabelecimento_id = $1`
	var planID string
	if err := s.db.GetContext(ctx, &planID, query, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPlanoSaasNaoEncontrado
		}
		return fmt.Errorf("buscar plano da assinatura: %w", err)
	}
	return s.AssignPlanToEstablishment(ctx, establishmentID, planID, months)
}

// ActivateEstablishment reativa salão e assinatura (se existir).
func (s *PlanoSaasService) ActivateEstablishment(ctx context.Context, estabelecimentos *EstabelecimentoService, establishmentID string) error {
	if err := estabelecimentos.ToggleEstablishmentStatus(ctx, establishmentID, true); err != nil {
		return err
	}
	_ = s.SetSubscriptionStatus(ctx, establishmentID, "ATIVO")
	return nil
}

// SuspendEstablishment suspende salão e assinatura (se existir).
func (s *PlanoSaasService) SuspendEstablishment(ctx context.Context, estabelecimentos *EstabelecimentoService, establishmentID string) error {
	if err := estabelecimentos.ToggleEstablishmentStatus(ctx, establishmentID, false); err != nil {
		return err
	}
	_ = s.SetSubscriptionStatus(ctx, establishmentID, "SUSPENSO")
	return nil
}

func (s *PlanoSaasService) validarEstabelecimento(ctx context.Context, tx *sqlx.Tx, establishmentID string) error {
	const query = `SELECT id FROM estabelecimentos WHERE id = $1 FOR UPDATE`
	var id string
	if err := tx.GetContext(ctx, &id, query, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEstabelecimentoNaoEncontrado
		}
		return fmt.Errorf("validar estabelecimento: %w", err)
	}
	return nil
}

func (s *PlanoSaasService) validarPlanoSaas(ctx context.Context, tx *sqlx.Tx, planID string) error {
	const query = `
SELECT id FROM planos_saas WHERE id = $1 AND ativo = TRUE FOR UPDATE
`
	var id string
	if err := tx.GetContext(ctx, &id, query, planID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPlanoSaasNaoEncontrado
		}
		return fmt.Errorf("validar plano SaaS: %w", err)
	}
	return nil
}

func (s *PlanoSaasService) resolverBaseVencimento(
	ctx context.Context,
	tx *sqlx.Tx,
	establishmentID string,
) (time.Time, error) {
	hoje := time.Now().UTC().Truncate(24 * time.Hour)

	const query = `
SELECT id, data_vencimento
FROM assinaturas_estabelecimentos
WHERE estabelecimento_id = $1
FOR UPDATE
`
	var assinatura assinaturaExistente
	err := tx.GetContext(ctx, &assinatura, query, establishmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return hoje, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("buscar assinatura existente: %w", err)
	}

	vencimento := assinatura.DataVencimento.UTC().Truncate(24 * time.Hour)
	if vencimento.After(hoje) {
		return vencimento, nil
	}

	return hoje, nil
}
