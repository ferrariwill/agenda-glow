package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// EstablishmentSuperAdminView consolida dados do salão para o painel Super Admin.
type EstablishmentSuperAdminView struct {
	ID               string     `db:"id" json:"id"`
	NomeComercial    string     `db:"nome_comercial" json:"nome_comercial"`
	Slug             string     `db:"slug" json:"slug"`
	Ativo            bool       `db:"ativo" json:"ativo"`
	DataCadastro     time.Time  `db:"data_cadastro" json:"data_cadastro"`
	PlanoID          *string    `db:"plano_id" json:"plano_id,omitempty"`
	PlanoNome        *string    `db:"plano_nome" json:"plano_nome,omitempty"`
	StatusRaw        *string    `db:"status_raw" json:"-"`
	DataVencimento   *time.Time `db:"data_vencimento" json:"data_vencimento,omitempty"`
	StatusAssinatura string     `json:"status_assinatura"`
}

// ListEstablishmentsSuperAdmin lista salões com status de assinatura calculado.
func (s *EstabelecimentoService) ListEstablishmentsSuperAdmin(ctx context.Context) ([]EstablishmentSuperAdminView, error) {
	const query = `
SELECT
    e.id,
    e.nome_comercial,
    e.slug,
    e.ativo,
    e.data_cadastro,
    ae.plano_id,
    ps.nome AS plano_nome,
    ae.status AS status_raw,
    ae.data_vencimento
FROM estabelecimentos e
LEFT JOIN assinaturas_estabelecimentos ae ON ae.estabelecimento_id = e.id
LEFT JOIN planos_saas ps ON ps.id = ae.plano_id
ORDER BY e.data_cadastro DESC
`
	var lista []EstablishmentSuperAdminView
	if err := s.db.SelectContext(ctx, &lista, query); err != nil {
		return nil, fmt.Errorf("listar estabelecimentos super admin: %w", err)
	}
	if lista == nil {
		lista = []EstablishmentSuperAdminView{}
	}

	for i := range lista {
		lista[i].StatusAssinatura = calcularStatusAssinatura(lista[i])
	}

	return lista, nil
}

// GetEstablishmentSuperAdminByID retorna um salão para partial HTMX.
func (s *EstabelecimentoService) GetEstablishmentSuperAdminByID(ctx context.Context, id string) (*EstablishmentSuperAdminView, error) {
	const query = `
SELECT
    e.id,
    e.nome_comercial,
    e.slug,
    e.ativo,
    e.data_cadastro,
    ae.plano_id,
    ps.nome AS plano_nome,
    ae.status AS status_raw,
    ae.data_vencimento
FROM estabelecimentos e
LEFT JOIN assinaturas_estabelecimentos ae ON ae.estabelecimento_id = e.id
LEFT JOIN planos_saas ps ON ps.id = ae.plano_id
WHERE e.id = $1
`
	var est EstablishmentSuperAdminView
	if err := s.db.GetContext(ctx, &est, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEstabelecimentoNaoEncontrado
		}
		return nil, fmt.Errorf("buscar estabelecimento super admin: %w", err)
	}
	est.StatusAssinatura = calcularStatusAssinatura(est)
	return &est, nil
}

func calcularStatusAssinatura(est EstablishmentSuperAdminView) string {
	if !est.Ativo {
		return "SUSPENSO"
	}
	if est.StatusRaw != nil && *est.StatusRaw == "SUSPENSO" {
		return "SUSPENSO"
	}
	if est.DataVencimento == nil {
		return "VENCIDO"
	}
	hoje := time.Now().Truncate(24 * time.Hour)
	venc := est.DataVencimento.Truncate(24 * time.Hour)
	if venc.Before(hoje) {
		return "VENCIDO"
	}
	return "ATIVO"
}
