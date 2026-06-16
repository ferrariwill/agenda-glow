package security

import "context"

type contextKey string

const establishmentIDKey contextKey = "estabelecimento_id"
const professionalIDKey contextKey = "profissional_id"

// WithEstablishmentID injeta o tenant no contexto (middleware de auth do salão).
func WithEstablishmentID(ctx context.Context, establishmentID string) context.Context {
	return context.WithValue(ctx, establishmentIDKey, establishmentID)
}

// EstablishmentIDFromContext extrai o estabelecimento_id do contexto da requisição.
func EstablishmentIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(establishmentIDKey).(string)
	return id, ok && id != ""
}

// WithProfessionalID injeta o profissional_id no contexto (middleware da profissional parceira).
func WithProfessionalID(ctx context.Context, professionalID string) context.Context {
	return context.WithValue(ctx, professionalIDKey, professionalID)
}

// ProfessionalIDFromContext extrai o profissional_id do contexto da requisição.
func ProfessionalIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(professionalIDKey).(string)
	return id, ok && id != ""
}
