package security

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrAssinaturaVencidaOuSuspensa = errors.New("assinatura vencida ou suspensa")

type assinaturaStatus struct {
	Status          string    `db:"status"`
	DataVencimento  time.Time `db:"data_vencimento"`
}

// SaaSGuard valida assinaturas corporativas com cache em memória de curta duração.
type SaaSGuard struct {
	db    *sqlx.DB
	cache *subscriptionCache
}

func NewSaaSGuard(db *sqlx.DB, cacheTTL time.Duration) *SaaSGuard {
	return &SaaSGuard{
		db:    db,
		cache: newSubscriptionCache(cacheTTL),
	}
}

// InvalidateCache remove o cache de um estabelecimento após renovação de plano.
func (g *SaaSGuard) InvalidateCache(establishmentID string) {
	g.cache.invalidate(establishmentID)
}

// IsSubscriptionActive verifica se o estabelecimento pode operar na plataforma.
func (g *SaaSGuard) IsSubscriptionActive(ctx context.Context, establishmentID string) error {
	if allowed, ok := g.cache.get(establishmentID); ok {
		if allowed {
			return nil
		}
		return ErrAssinaturaVencidaOuSuspensa
	}

	allowed, err := g.checkDatabase(ctx, establishmentID)
	if err != nil {
		return err
	}

	g.cache.set(establishmentID, allowed)
	if !allowed {
		return ErrAssinaturaVencidaOuSuspensa
	}

	return nil
}

func (g *SaaSGuard) checkDatabase(ctx context.Context, establishmentID string) (bool, error) {
	const query = `
SELECT status, data_vencimento
FROM assinaturas_estabelecimentos
WHERE estabelecimento_id = $1
LIMIT 1
`
	var assinatura assinaturaStatus
	err := g.db.GetContext(ctx, &assinatura, query, establishmentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("consultar assinatura: %w", err)
	}

	if assinatura.Status == "SUSPENSO" {
		return false, nil
	}

	hoje := time.Now().UTC().Truncate(24 * time.Hour)
	vencimento := assinatura.DataVencimento.UTC().Truncate(24 * time.Hour)
	if hoje.After(vencimento) {
		return false, nil
	}

	return true, nil
}
