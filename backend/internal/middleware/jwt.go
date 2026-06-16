package middleware

import (
	"net/http"

	"github.com/agendaglow/agendaglow/internal/security"
)

// JWTMiddleware restringe rotas ao Super Admin (ferrariwill@gmail.com).
func JWTMiddleware(next http.Handler) http.Handler {
	return security.RequireSuperAdmin(next)
}
