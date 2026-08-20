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
	"github.com/lib/pq"
)

var (
	ErrCredenciaisInvalidas = errors.New("credenciais inválidas")
	ErrUsuarioInativo       = errors.New("usuário inativo")
	ErrEmailJaCadastrado    = errors.New("e-mail já cadastrado")
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

// ChangePassword atualiza a senha do próprio usuário autenticado (por ID do JWT).
func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || currentPassword == "" || newPassword == "" {
		return ErrCredenciaisInvalidas
	}

	user, err := s.buscarPorID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCredenciaisInvalidas
		}
		return err
	}

	if !user.Ativo {
		return ErrUsuarioInativo
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrCredenciaisInvalidas
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash da senha: %w", err)
	}

	const update = `UPDATE users SET password_hash = $1 WHERE id = $2`
	if _, err := s.db.ExecContext(ctx, update, string(hash), user.ID); err != nil {
		return fmt.Errorf("atualizar senha: %w", err)
	}
	return nil
}

// CreateUser cadastra credencial com hash bcrypt (uso administrativo / seed).
func (s *AuthService) CreateUser(ctx context.Context, email, password, role string, estabelecimentoID, profissionalID *string) (string, error) {
	return s.CreateUserWithNome(ctx, email, password, role, "", estabelecimentoID, profissionalID)
}

// CreateUserWithNome cadastra credencial incluindo users.nome.
func (s *AuthService) CreateUserWithNome(
	ctx context.Context,
	email, password, role, nome string,
	estabelecimentoID, profissionalID *string,
) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	nome = strings.TrimSpace(nome)
	if email == "" || password == "" {
		return "", fmt.Errorf("email e senha são obrigatórios")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash da senha: %w", err)
	}

	var nomeArg any
	if nome != "" {
		nomeArg = nome
	}

	const insert = `
INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, nome, ativo)
VALUES ($1, $2, $3, $4, $5, $6, TRUE)
RETURNING id
`
	var id string
	if err := s.db.GetContext(ctx, &id, insert, email, string(hash), role, estabelecimentoID, profissionalID, nomeArg); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", ErrEmailJaCadastrado
		}
		return "", fmt.Errorf("criar usuário: %w", err)
	}
	return id, nil
}

// CreateDonaForEstablishment cria usuária DONA com nome e atualiza dona_nome/dona_email do salão.
func (s *AuthService) CreateDonaForEstablishment(
	ctx context.Context,
	estabelecimentoID, nome, email, password string,
) (string, error) {
	estabelecimentoID = strings.TrimSpace(estabelecimentoID)
	nome = strings.TrimSpace(nome)
	email = strings.TrimSpace(strings.ToLower(email))
	if estabelecimentoID == "" {
		return "", fmt.Errorf("estabelecimento_id é obrigatório")
	}
	if nome == "" {
		return "", fmt.Errorf("nome é obrigatório")
	}
	if email == "" || password == "" {
		return "", fmt.Errorf("email e senha são obrigatórios")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash da senha: %w", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	const lockEstabelecimento = `
SELECT id FROM estabelecimentos WHERE id = $1 FOR UPDATE
`
	var estID string
	if err := tx.GetContext(ctx, &estID, lockEstabelecimento, estabelecimentoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrEstabelecimentoNaoEncontrado
		}
		return "", fmt.Errorf("bloquear estabelecimento: %w", err)
	}

	const insert = `
INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, nome, ativo)
VALUES ($1, $2, $3, $4, NULL, $5, TRUE)
RETURNING id
`
	var userID string
	if err := tx.GetContext(ctx, &userID, insert, email, string(hash), security.RoleDona, estabelecimentoID, nome); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return "", ErrEmailJaCadastrado
		}
		return "", fmt.Errorf("criar usuária dona: %w", err)
	}

	const updateDona = `
UPDATE estabelecimentos
SET dona_nome = $2, dona_email = $3
WHERE id = $1
`
	if _, err := tx.ExecContext(ctx, updateDona, estabelecimentoID, nome, email); err != nil {
		return "", fmt.Errorf("atualizar contato da dona: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("confirmar transação: %w", err)
	}
	return userID, nil
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

func (s *AuthService) buscarPorID(ctx context.Context, id string) (*UserCredential, error) {
	const query = `
SELECT id, email, password_hash, role, estabelecimento_id, profissional_id, ativo
FROM users
WHERE id = $1
`
	var user UserCredential
	if err := s.db.GetContext(ctx, &user, query, id); err != nil {
		return nil, err
	}
	return &user, nil
}
