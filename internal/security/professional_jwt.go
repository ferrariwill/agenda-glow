package security

import "net/http"

// ProfessionalJWTMiddleware mantém compatibilidade — delega para RequireProfissional.
func ProfessionalJWTMiddleware(next http.Handler) http.Handler {
	return RequireProfissional(next)
}
