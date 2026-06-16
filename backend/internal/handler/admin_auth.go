package handler

import (
	"net/http"
	"os"
	"strings"
)

func AdminAuth(next http.Handler) http.Handler {
	expectedKey := os.Getenv("ADMIN_API_KEY")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedKey == "" {
			http.Error(w, "Autenticação administrativa não configurada", http.StatusInternalServerError)
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if token == "" || token != expectedKey {
			http.Error(w, "Não autorizado", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
