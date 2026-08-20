package handler

import (
	"bytes"
	"database/sql"
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
	"github.com/lib/pq"
)

const createDonaPattern = "POST /api/v1/admin/establishments/{id}/create-dona"

func newCreateDonaMux(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	handler := NewAdminEstablishmentsHandler(
		service.NewEstabelecimentoService(db),
		service.NewAuthService(db),
		security.NewWhatsAppGate(db, time.Minute),
	)

	mux := http.NewServeMux()
	mux.Handle(createDonaPattern, security.RequireSuperAdmin(http.HandlerFunc(handler.CreateDona)))
	return mux, mock
}

func newCreateDonaHTTPRequest(token, establishmentID, body string) *http.Request {
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/establishments/"+establishmentID+"/create-dona",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func expectGetEstablishmentSuperAdmin(mock sqlmock.Sqlmock, id string, found bool) {
	query := regexp.QuoteMeta(`
SELECT
    e.id,
    e.nome_comercial,
    e.slug,
    e.ativo,
    e.data_cadastro,
    COALESCE(e.whatsapp_enabled, FALSE) AS whatsapp_enabled,
    COALESCE(e.whatsapp_status, 'DESCONECTADO') AS whatsapp_status,
    ae.plano_id,
    ps.nome AS plano_nome,
    ae.status AS status_raw,
    ae.data_vencimento
FROM estabelecimentos e
LEFT JOIN assinaturas_estabelecimentos ae ON ae.estabelecimento_id = e.id
LEFT JOIN planos_saas ps ON ps.id = ae.plano_id
WHERE e.id = $1
`)
	if !found {
		mock.ExpectQuery(query).WithArgs(id).WillReturnError(sql.ErrNoRows)
		return
	}
	rows := sqlmock.NewRows([]string{
		"id", "nome_comercial", "slug", "ativo", "data_cadastro",
		"whatsapp_enabled", "whatsapp_status", "plano_id", "plano_nome", "status_raw", "data_vencimento",
	}).AddRow(id, "Salão Teste", "salao-teste", true, time.Now(), false, "DESCONECTADO", nil, nil, nil, nil)
	mock.ExpectQuery(query).WithArgs(id).WillReturnRows(rows)
}

func TestCreateDonaJSON_Success(t *testing.T) {
	mux, mock := newCreateDonaMux(t)
	estID := "11111111-1111-4111-8111-111111111111"
	expectGetEstablishmentSuperAdmin(mock, estID, true)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id FROM estabelecimentos WHERE id = $1 FOR UPDATE
`)).WithArgs(estID).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(estID))
	mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, nome, ativo)
VALUES ($1, $2, $3, $4, NULL, $5, TRUE)
RETURNING id
`)).WithArgs("maria@salao.com", sqlmock.AnyArg(), security.RoleDona, estID, "Maria Silva").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-dona-1"))
	mock.ExpectExec(regexp.QuoteMeta(`
UPDATE estabelecimentos
SET dona_nome = $2, dona_email = $3
WHERE id = $1
`)).WithArgs(estID, "Maria Silva", "maria@salao.com").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), estID,
		`{"nome":" Maria Silva ","email":"Maria@Salao.com"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out createDonaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.UserID != "user-dona-1" || out.Email != "maria@salao.com" || out.Nome != "Maria Silva" {
		t.Fatalf("resposta inesperada: %+v", out)
	}
	if out.SenhaInicial != senhaInicialDona {
		t.Fatalf("senha_inicial=%q", out.SenhaInicial)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestCreateDonaJSON_ValidationAndAuth(t *testing.T) {
	mux, mock := newCreateDonaMux(t)
	estID := "11111111-1111-4111-8111-111111111111"

	t.Run("missing_nome", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), estID, `{"nome":"","email":"a@b.com"}`))
		assertHandlerJSONError(t, rec, http.StatusBadRequest, "missing_nome")
	})

	t.Run("missing_email", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), estID, `{"nome":"A","email":" "}`))
		assertHandlerJSONError(t, rec, http.StatusBadRequest, "missing_email")
	})

	t.Run("invalid_json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), estID, `{`))
		assertHandlerJSONError(t, rec, http.StatusBadRequest, "invalid_json")
	})

	t.Run("not_found", func(t *testing.T) {
		expectGetEstablishmentSuperAdmin(mock, estID, false)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), estID,
			`{"nome":"A","email":"a@b.com"}`))
		assertHandlerJSONError(t, rec, http.StatusNotFound, "not_found")
	})

	t.Run("email_already_exists", func(t *testing.T) {
		expectGetEstablishmentSuperAdmin(mock, estID, true)
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`
SELECT id FROM estabelecimentos WHERE id = $1 FOR UPDATE
`)).WithArgs(estID).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(estID))
		mock.ExpectQuery(regexp.QuoteMeta(`
INSERT INTO users (email, password_hash, role, estabelecimento_id, profissional_id, nome, ativo)
VALUES ($1, $2, $3, $4, NULL, $5, TRUE)
RETURNING id
`)).WillReturnError(&pq.Error{Code: "23505"})
		mock.ExpectRollback()

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), estID,
			`{"nome":"A","email":"a@b.com"}`))
		assertHandlerJSONError(t, rec, http.StatusConflict, "email_already_exists")
	})

	t.Run("forbidden_dona", func(t *testing.T) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, newCreateDonaHTTPRequest(donaToken(t, estID), estID,
			`{"nome":"A","email":"a@b.com"}`))
		if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
			t.Fatalf("esperado 401/403, obtido %d", rec.Code)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestUploadLogoJSON_Validation(t *testing.T) {
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	handler := NewAdminEstablishmentsHandler(
		service.NewEstabelecimentoService(db),
		service.NewAuthService(db),
		security.NewWhatsAppGate(db, time.Minute),
	)
	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/admin/establishments/{id}/logo",
		security.RequireSuperAdmin(http.HandlerFunc(handler.UploadLogo)))

	estID := "11111111-1111-4111-8111-111111111111"

	t.Run("not_found", func(t *testing.T) {
		expectGetEstablishmentSuperAdmin(mock, estID, false)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/establishments/"+estID+"/logo", nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		assertHandlerJSONError(t, rec, http.StatusNotFound, "not_found")
	})

	t.Run("missing_logo", func(t *testing.T) {
		expectGetEstablishmentSuperAdmin(mock, estID, true)
		var body bytes.Buffer
		// multipart vazio (sem campo logo)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/establishments/"+estID+"/logo", &body)
		req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("esperado 400, obtido %d (%s)", rec.Code, rec.Body.String())
		}
		code := decodeErrorCode(t, rec.Body.Bytes())
		if code != "missing_logo" && code != "invalid_multipart" {
			t.Fatalf("esperado missing_logo|invalid_multipart, obtido %q", code)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func assertHandlerJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status esperado %d, obtido %d (%s)", wantStatus, rec.Code, rec.Body.String())
	}
	if got := decodeErrorCode(t, rec.Body.Bytes()); got != wantCode {
		t.Fatalf("error code esperado %q, obtido %q (%s)", wantCode, got, rec.Body.String())
	}
}

func decodeErrorCode(t *testing.T, body []byte) string {
	t.Helper()
	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decodificar erro: %v (%s)", err, body)
	}
	return out.Error
}
