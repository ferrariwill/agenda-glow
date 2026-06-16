package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	RoleSuperAdmin  = "SUPER_ADMIN"
	RoleDona        = "DONA"
	RoleProfissional = "PROFISSIONAL"

	SuperAdminEmail = "ferrariwill@gmail.com"

	defaultTokenTTL   = 24 * time.Hour
	SessionCookieName = "agendaglow_session"
)

var (
	ErrTokenInvalido     = errors.New("token inválido")
	ErrTokenExpirado     = errors.New("token expirado")
	ErrJWTNaoConfigurado = errors.New("JWT_SECRET não configurado")
	ErrAcessoNegado      = errors.New("acesso negado ao recurso")
)

const sessionClaimsKey contextKey = "session_claims"

// Claims é o payload unificado do JWT para todos os perfis.
type Claims struct {
	UserID            string  `json:"user_id"`
	Email             string  `json:"email"`
	Role              string  `json:"role"`
	EstabelecimentoID *string `json:"estabelecimento_id,omitempty"`
	ProfissionalID    *string `json:"profissional_id,omitempty"`
	Exp               int64   `json:"exp"`
}

// GenerateToken emite JWT assinado com escopo completo do usuário autenticado.
func GenerateToken(claims Claims, ttl time.Duration) (string, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", err
	}
	if claims.UserID == "" || claims.Email == "" || claims.Role == "" {
		return "", fmt.Errorf("claims incompletos para emissão do token")
	}
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	if claims.Exp == 0 {
		claims.Exp = time.Now().Add(ttl).Unix()
	}

	if err := validateClaimsScope(claims); err != nil {
		return "", err
	}

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + payload

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature, nil
}

// ParseToken valida assinatura e expiração do JWT.
func ParseToken(token string) (*Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalido
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, ErrTokenInvalido
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrTokenInvalido
	}

	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, ErrTokenInvalido
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpirado
	}

	if err := validateClaimsScope(claims); err != nil {
		return nil, ErrTokenInvalido
	}

	return &claims, nil
}

func validateClaimsScope(claims Claims) error {
	switch claims.Role {
	case RoleSuperAdmin:
		if claims.EstabelecimentoID != nil || claims.ProfissionalID != nil {
			return fmt.Errorf("super admin não deve ter escopo de tenant")
		}
	case RoleDona:
		if claims.EstabelecimentoID == nil || *claims.EstabelecimentoID == "" {
			return fmt.Errorf("dona requer estabelecimento_id")
		}
		if claims.ProfissionalID != nil {
			return fmt.Errorf("dona não deve ter profissional_id")
		}
	case RoleProfissional:
		if claims.EstabelecimentoID == nil || *claims.EstabelecimentoID == "" {
			return fmt.Errorf("profissional requer estabelecimento_id")
		}
		if claims.ProfissionalID == nil || *claims.ProfissionalID == "" {
			return fmt.Errorf("profissional requer profissional_id")
		}
	default:
		return fmt.Errorf("role inválida: %s", claims.Role)
	}
	return nil
}

func jwtSecret() (string, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return "", ErrJWTNaoConfigurado
	}
	return secret, nil
}

// AuthenticateMiddleware extrai e valida o JWT, injetando Claims no contexto.
func AuthenticateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			respondUnauthorized(w, r)
			return
		}

		claims, err := ParseToken(token)
		if err != nil {
			respondUnauthorized(w, r)
			return
		}

		ctx := WithSessionClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireSuperAdmin restringe rotas ao Super Admin autorizado.
func RequireSuperAdmin(next http.Handler) http.Handler {
	return AuthenticateMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			respondUnauthorized(w, r)
			return
		}
		if claims.Role != RoleSuperAdmin || !strings.EqualFold(claims.Email, SuperAdminEmail) {
			respondForbidden(w, r, claims.Role, RoleSuperAdmin)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequireDona restringe rotas à dona do salão (painel financeiro e configuração).
func RequireDona(next http.Handler) http.Handler {
	return AuthenticateMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			respondUnauthorized(w, r)
			return
		}
		if claims.Role != RoleDona {
			respondForbidden(w, r, claims.Role, RoleDona)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequireProfissional restringe rotas à profissional parceira autenticada.
func RequireProfissional(next http.Handler) http.Handler {
	return AuthenticateMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok {
			respondUnauthorized(w, r)
			return
		}
		if claims.Role != RoleProfissional {
			respondForbidden(w, r, claims.Role, RoleProfissional)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// WithSessionClaims injeta claims e IDs derivados no contexto da requisição.
func WithSessionClaims(ctx context.Context, claims *Claims) context.Context {
	ctx = context.WithValue(ctx, sessionClaimsKey, claims)
	if claims.EstabelecimentoID != nil && *claims.EstabelecimentoID != "" {
		ctx = WithEstablishmentID(ctx, *claims.EstabelecimentoID)
	}
	if claims.ProfissionalID != nil && *claims.ProfissionalID != "" {
		ctx = WithProfessionalID(ctx, *claims.ProfissionalID)
	}
	return ctx
}

// ClaimsFromContext retorna o payload JWT autenticado.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(sessionClaimsKey).(*Claims)
	return claims, ok && claims != nil
}

func bearerToken(r *http.Request) string {
	if token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")); token != "" {
		return token
	}
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

// SetSessionCookie grava o JWT em cookie HttpOnly para navegação HTML.
func SetSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func respondUnauthorized(w http.ResponseWriter, r *http.Request) {
	if wantsHTML(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
}

func respondForbidden(w http.ResponseWriter, r *http.Request, currentRole, requiredRole string) {
	_ = currentRole
	if wantsHTML(r) {
		http.Redirect(w, r, loginPathForRole(requiredRole), http.StatusSeeOther)
		return
	}
	writeJSON(w, http.StatusForbidden, map[string]string{
		"error":         "forbidden",
		"required_role": requiredRole,
	})
}

func wantsHTML(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

func loginPathForRole(role string) string {
	switch role {
	case RoleSuperAdmin:
		return "/login/superadmin"
	case RoleDona:
		return "/login/dona"
	case RoleProfissional:
		return "/login/profissional"
	default:
		return "/login"
	}
}

// LoginRedirectForRole devolve a URL de login do perfil informado.
func LoginRedirectForRole(role string) string {
	return loginPathForRole(role)
}
