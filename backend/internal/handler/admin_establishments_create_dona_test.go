package handler

import (
	"bytes"
	"database/sql"
	"mime/multipart"
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
const uploadLogoPattern = "POST /api/v1/admin/establishments/{id}/logo"

func newAdminEstMux(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
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
	mux.Handle(uploadLogoPattern, security.RequireSuperAdmin(http.HandlerFunc(handler.UploadLogo)))
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

func expectEstablishmentLookup(mock sqlmock.Sqlmock, id string, found bool) {
	query := mock.ExpectQuery(regexp.QuoteMeta("WHERE e.id = $1"))
	if !found {
		query.WithArgs(id).WillReturnError(sql.ErrNoRows)
		return
	}
	query.WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{
		"id", "nome_comercial", "slug", "ativo", "data_cadastro",
		"whatsapp_enabled", "whatsapp_status",
		"plano_id", "plano_nome", "status_raw", "data_vencimento",
	}).AddRow(id, "Studio Bella", "studio-bella", true, time.Now(),
		false, "DESCONECTADO", nil, nil, nil, nil))
}

func TestCreateDonaWithoutTokenIsUnauthorized(t *testing.T) {
	mux, mock := newAdminEstMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest("", "tenant-1", `{"nome":"Maria","email":"maria@salao.com"}`))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonaWithDonaTokenIsForbidden(t *testing.T) {
	mux, mock := newAdminEstMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(donaToken(t, "tenant-1"), "tenant-1",
		`{"nome":"Maria","email":"maria@salao.com"}`))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonaMissingNome(t *testing.T) {
	mux, mock := newAdminEstMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), "tenant-1",
		`{"nome":"  ","email":"maria@salao.com"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "missing_nome" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonaMissingEmail(t *testing.T) {
	mux, mock := newAdminEstMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), "tenant-1",
		`{"nome":"Maria","email":""}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "missing_email" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonaUnknownEstablishment(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "missing", false)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), "missing",
		`{"nome":"Maria","email":"maria@salao.com"}`))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "not_found" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonaDuplicateEmail(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "tenant-1", true)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs("maria@salao.com", sqlmock.AnyArg(), security.RoleDona, "tenant-1", nil, "Maria Silva").
		WillReturnError(&pq.Error{Code: "23505"})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), "tenant-1",
		`{"nome":"Maria Silva","email":"Maria@Salao.com"}`))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "email_already_exists" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateDonaSuccess(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "tenant-1", true)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users")).
		WithArgs("maria@salao.com", sqlmock.AnyArg(), security.RoleDona, "tenant-1", nil, "Maria Silva").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-dona-1"))
	mock.ExpectExec(regexp.QuoteMeta("SET dona_nome = $2, dona_email = $3")).
		WithArgs("tenant-1", "Maria Silva", "maria@salao.com").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, newCreateDonaHTTPRequest(superAdminToken(t), "tenant-1",
		`{"nome":" Maria Silva ","email":"Maria@Salao.com"}`))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["user_id"] != "user-dona-1" ||
		body["email"] != "maria@salao.com" ||
		body["nome"] != "Maria Silva" ||
		body["senha_inicial"] != senhaInicialDona {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadLogoWithoutTokenIsUnauthorized(t *testing.T) {
	mux, mock := newAdminEstMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, multipartLogoRequest(t, "", "tenant-1", "logo.png", []byte("png-bytes")))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadLogoMissingFile(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "tenant-1", true)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("other", "x")
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/establishments/tenant-1/logo", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "missing_logo" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadLogoInvalidImageType(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "tenant-1", true)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, multipartLogoRequest(t, superAdminToken(t), "tenant-1", "logo.gif", []byte("gif")))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "invalid_image_type" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadLogoUnknownEstablishment(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "missing", false)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, multipartLogoRequest(t, superAdminToken(t), "missing", "logo.png", []byte("png")))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	if body := decodeBody(t, rec); body["error"] != "not_found" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadLogoFileTooLarge(t *testing.T) {
	mux, mock := newAdminEstMux(t)
	expectEstablishmentLookup(mock, "tenant-1", true)

	huge := bytes.Repeat([]byte("a"), maxLogoUploadBytes+1)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, multipartLogoRequest(t, superAdminToken(t), "tenant-1", "logo.png", huge))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["error"] != "file_too_large" && body["error"] != "invalid_multipart" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func multipartLogoRequest(t *testing.T, token, establishmentID, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("logo", filename)
	if err != nil {
		t.Fatalf("criar form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("escrever conteÃºdo: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("fechar multipart: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/establishments/"+establishmentID+"/logo",
		&buf,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestCreateDonaRoutesAreRegistered(t *testing.T) {
	mux, _ := newAdminEstMux(t)
	_, pattern := mux.(*http.ServeMux).Handler(newCreateDonaHTTPRequest("", "tenant-1", `{}`))
	if pattern != createDonaPattern {
		t.Fatalf("create-dona pattern = %q", pattern)
	}

	_, logoPattern := mux.(*http.ServeMux).Handler(
		multipartLogoRequest(t, "", "tenant-1", "logo.png", []byte("x")),
	)
	if logoPattern != uploadLogoPattern {
		t.Fatalf("logo pattern = %q", logoPattern)
	}
}
