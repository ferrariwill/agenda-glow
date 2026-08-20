package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const changePasswordPattern = "POST /api/v1/auth/change-password"

func newChangePasswordMux(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	handler := NewAuthHandler(service.NewAuthService(db))

	mux := http.NewServeMux()
	mux.Handle(changePasswordPattern, security.AuthenticateMiddleware(http.HandlerFunc(handler.ChangePassword)))
	return mux, mock
}

func roleToken(t *testing.T, userID, email, role string, estabelecimentoID, profissionalID *string) string {
	t.Helper()
	token, err := security.GenerateToken(security.Claims{
		UserID:            userID,
		Email:             email,
		Role:              role,
		EstabelecimentoID: estabelecimentoID,
		ProfissionalID:    profissionalID,
	}, time.Hour)
	if err != nil {
		t.Fatalf("gerar token %s: %v", role, err)
	}
	return token
}

func newChangePasswordHTTPRequest(token, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func mustBcrypt(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	return string(hash)
}

func expectUserLookup(mock sqlmock.Sqlmock, userID, hash string, ativo bool) {
	rows := sqlmock.NewRows([]string{
		"id", "email", "password_hash", "role", "estabelecimento_id", "profissional_id", "ativo",
	}).AddRow(userID, "user@glow.local", hash, "DONA", "tenant-1", nil, ativo)
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id, email, password_hash, role, estabelecimento_id, profissional_id, ativo
FROM users
WHERE id = $1
`)).WithArgs(userID).WillReturnRows(rows)
}

func TestChangePasswordRouteIsRegistered(t *testing.T) {
	mux, _ := newChangePasswordMux(t)
	_, pattern := mux.(*http.ServeMux).Handler(newChangePasswordHTTPRequest("", `{}`))
	if pattern != changePasswordPattern {
		t.Fatalf("rota não registrada: pattern = %q", pattern)
	}
}

func TestChangePasswordWithoutTokenIsUnauthorized(t *testing.T) {
	mux, _ := newChangePasswordMux(t)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newChangePasswordHTTPRequest("", `{
		"current_password":"AgendaGlow@2026",
		"new_password":"NovaSenha@2026",
		"confirm_password":"NovaSenha@2026"
	}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestChangePasswordConfirmMismatchIsValidationError(t *testing.T) {
	mux, _ := newChangePasswordMux(t)
	est := "tenant-1"
	token := roleToken(t, "user-dona", "dona@glow.local", security.RoleDona, &est, nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newChangePasswordHTTPRequest(token, `{
		"current_password":"AgendaGlow@2026",
		"new_password":"NovaSenha@2026",
		"confirm_password":"OutraSenha@2026"
	}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "validation_error" {
		t.Fatalf("body = %#v", body)
	}
}

func TestChangePasswordWrongCurrentIsUnauthorized(t *testing.T) {
	mux, mock := newChangePasswordMux(t)
	est := "tenant-1"
	userID := "user-dona"
	token := roleToken(t, userID, "dona@glow.local", security.RoleDona, &est, nil)
	expectUserLookup(mock, userID, mustBcrypt(t, "AgendaGlow@2026"), true)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newChangePasswordHTTPRequest(token, `{
		"current_password":"senha-errada",
		"new_password":"NovaSenha@2026",
		"confirm_password":"NovaSenha@2026"
	}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "invalid_credentials" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestChangePasswordSuccessByRole(t *testing.T) {
	cases := []struct {
		name   string
		role   string
		email  string
		estID  *string
		profID *string
	}{
		{"DONA", security.RoleDona, "dona@glow.local", strPtr("tenant-1"), nil},
		{"PROFISSIONAL", security.RoleProfissional, "claudia@glow.local", strPtr("tenant-1"), strPtr("prof-1")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mux, mock := newChangePasswordMux(t)
			userID := "user-" + strings.ToLower(tc.role)
			token := roleToken(t, userID, tc.email, tc.role, tc.estID, tc.profID)
			hash := mustBcrypt(t, "AgendaGlow@2026")
			expectUserLookup(mock, userID, hash, true)
			mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET password_hash = $1 WHERE id = $2`)).
				WithArgs(sqlmock.AnyArg(), userID).
				WillReturnResult(sqlmock.NewResult(0, 1))

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, newChangePasswordHTTPRequest(token, `{
				"current_password":"AgendaGlow@2026",
				"new_password":"NovaSenha@2026",
				"confirm_password":"NovaSenha@2026"
			}`))
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("esperado body vazio, got %q", rec.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock: %v", err)
			}
		})
	}
}

func TestChangePasswordInactiveUserIsForbidden(t *testing.T) {
	mux, mock := newChangePasswordMux(t)
	est := "tenant-1"
	userID := "user-dona"
	token := roleToken(t, userID, "dona@glow.local", security.RoleDona, &est, nil)
	expectUserLookup(mock, userID, mustBcrypt(t, "AgendaGlow@2026"), false)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newChangePasswordHTTPRequest(token, `{
		"current_password":"AgendaGlow@2026",
		"new_password":"NovaSenha@2026",
		"confirm_password":"NovaSenha@2026"
	}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["error"] != "user_inactive" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func strPtr(s string) *string { return &s }
