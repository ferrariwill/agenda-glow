package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

// Login POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	result, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCredenciaisInvalidas):
			writeJSONError(w, http.StatusUnauthorized, "invalid_credentials")
		case errors.Is(err, service.ErrUsuarioInativo):
			writeJSONError(w, http.StatusForbidden, "user_inactive")
		default:
			writeJSONError(w, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	security.SetSessionCookie(w, result.Token, 24*time.Hour)
	writeJSON(w, http.StatusOK, loginResponse{
		Token: result.Token,
		Role:  result.User.Role,
	})
}

// LoginForm POST /login (form-urlencoded para páginas HTML)
func (h *AuthHandler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulário inválido", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	roleHint := strings.ToUpper(strings.TrimSpace(r.FormValue("role")))
	if roleHint == "" {
		roleHint = roleFromLoginPath(r.URL.Path)
	}

	result, err := h.auth.Login(r.Context(), email, password)
	if err != nil {
		http.Redirect(w, r, loginFallbackPath(roleHint), http.StatusSeeOther)
		return
	}

	if roleHint != "" && result.User.Role != roleHint {
		http.Redirect(w, r, loginFallbackPath(result.User.Role), http.StatusSeeOther)
		return
	}

	redirect := homePathForRole(result.User.Role)
	security.SetSessionCookie(w, result.Token, 24*time.Hour)
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

func loginFallbackPath(role string) string {
	switch role {
	case "SUPER_ADMIN":
		return "/login/superadmin?error=1"
	case "DONA":
		return "/login/dona?error=1"
	case "PROFISSIONAL":
		return "/login/profissional?error=1"
	default:
		return "/login?error=1"
	}
}

func homePathForRole(role string) string {
	switch role {
	case "SUPER_ADMIN":
		return "/superadmin/dashboard"
	case "DONA":
		return "/dashboard/gerencial"
	case "PROFISSIONAL":
		return "/dashboard/profissional"
	default:
		return "/"
	}
}

func roleFromLoginPath(path string) string {
	switch strings.TrimSuffix(path, "/") {
	case "/login/superadmin":
		return "SUPER_ADMIN"
	case "/login/dona":
		return "DONA"
	case "/login/profissional":
		return "PROFISSIONAL"
	default:
		return ""
	}
}
