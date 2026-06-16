package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrCredenciaisInvalidas = errors.New("credenciais inválidas")
	ErrUsuarioInativo       = errors.New("usuário inativo")
)

type AuthService struct {
	db *sqlx.DB
}

func NewAuthService(db *sqlx.DB) *AuthService {
	return &AuthService{db: db}
}

type UserCredential struct {
	ID                string  `db:"id"`
	Email             string  `db:"email"`
	PasswordHash      string  `db:"password_hash"`
	Role              string  `db:"role"`
	EstabelecimentoID *string `db:"estabelecimento_id"`
	ProfissionalID    *string `db:"profissional_id"`
	Ativo             bool    `db:"ativo"`
}

type LoginResult struct {
	Token string          `json:"token"`
	User  UserCredential  `json:"user"`
}

// Login autentica e emite JWT unificado com escopo do perfil.
func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return nil, ErrCredenciaisInvalidas
	}

	user, err := s.buscarPorEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCredenciaisInvalidas
		}
		return nil, err
	}

	if !user.Ativo {
		return nil, ErrUsuarioInativo
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrCredenciaisInvalidas
	}

	if user.Role == security.RoleSuperAdmin && !strings.EqualFold(user.Email, security.SuperAdminEmail) {
		return nil, ErrCredenciaisInvalidas
	}

	claims := security.Claims{
		UserID:            user.ID,
		Email:             user.Email,
		Role:              user.Role,
		EstabelecimentoID: user.EstabelecimentoID,
		ProfissionalID:    user.ProfissionalID,
	}

	token, err := security.GenerateToken(claims, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("gerar token: %w", err)
	}

	return &LoginResult{Token: token, User: *user}, nil
}

// CreateUser cadastra credencial com hash bcrypt (uso administrativo / seed).
func (s *AuthService) CreateUser(ctx context.Context, email, password, role string, estabelecimentoID, profissionalID *string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return "", fmt.Errorf("email e senha são obrigatórios")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash da senha: %w", err)
	}

	const insert = `
INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, ativo)
VALUES ($1, $2, $3, $4, $5, TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, email, string(hash), role, estabelecimentoID, profissionalID); err != nil {
		return "", fmt.Errorf("criar usuário: %w", err)
	}
	return id, nil
}

func (s *AuthService) buscarPorEmail(ctx context.Context, email string) (*UserCredential, error) {
	const query = `
SELECT id, email, password_hash, role, estabelecimento_id, profissional_id, ativo
FROM users
WHERE LOWER(email) = LOWER($1)
`
	var user UserCredential
	if err := s.db.GetContext(ctx, &user, query, email); err != nil {
		return nil, err
	}
	return &user, nil
}
