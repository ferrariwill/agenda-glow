package security

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

var ErrWhatsAppDesativado = errors.New("recurso de whatsapp desativado para o estabelecimento")

type WhatsAppGate struct {
	db    *sqlx.DB
	cache *subscriptionCache
}

func NewWhatsAppGate(db *sqlx.DB, cacheTTL time.Duration) *WhatsAppGate {
	return &WhatsAppGate{
		db:    db,
		cache: newSubscriptionCache(cacheTTL),
	}
}

func (g *WhatsAppGate) InvalidateCache(establishmentID string) {
	g.cache.invalidate(establishmentID)
}

func (g *WhatsAppGate) IsEnabled(ctx context.Context, establishmentID string) error {
	if enabled, ok := g.cache.get(establishmentID); ok {
		if enabled {
			return nil
		}
		return ErrWhatsAppDesativado
	}

	const query = `
SELECT COALESCE(whatsapp_enabled, FALSE)
FROM estabelecimentos
WHERE id = $1
`
	var enabled bool
	if err := g.db.GetContext(ctx, &enabled, query, establishmentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			g.cache.set(establishmentID, false)
			return ErrWhatsAppDesativado
		}
		return fmt.Errorf("consultar liberação whatsapp: %w", err)
	}

	g.cache.set(establishmentID, enabled)
	if !enabled {
		return ErrWhatsAppDesativado
	}
	return nil
}

func RequireWhatsAppEnabled(gate *WhatsAppGate) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			establishmentID, ok := EstablishmentIDFromContext(r.Context())
			if !ok {
				writeJSON(w, http.StatusUnauthorized, map[string]string{
					"error": "missing_establishment_context",
				})
				return
			}
			if err := gate.IsEnabled(r.Context(), establishmentID); err != nil {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"error":   "whatsapp_feature_disabled",
					"message": "Recurso de WhatsApp desativado para este estabelecimento. Contate o administrador",
				})
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}
