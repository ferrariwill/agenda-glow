package security

import "net/http"

// TenantJWTMiddleware mantém compatibilidade — delega para RequireDona.
func TenantJWTMiddleware(next http.Handler) http.Handler {
	return RequireDona(next)
}
