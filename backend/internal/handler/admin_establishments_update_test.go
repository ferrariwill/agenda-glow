package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
)

const updateEstablishmentPattern = "PUT /api/v1/admin/establishments/{id}"

func newUpdateEstablishmentMux(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	gate := security.NewWhatsAppGate(db, time.Minute)
	handler := NewAdminEstablishmentsHandler(service.NewEstabelecimentoService(db), service.NewAuthService(db), gate)

	mux := http.NewServeMux()
	mux.Handle(updateEstablishmentPattern, security.RequireSuperAdmin(http.HandlerFunc(handler.Update)))
	return mux, mock
}

func updateRequest(token, establishmentID, body string) *http.Request {
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/establishments/"+establishmentID,
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func expectAdminIdentidadeHappyPath(mock sqlmock.Sqlmock, id, nome, slug string, logo any) {
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug = \\$1 AND id <> \\$2").
		WithArgs(slug, id).
		WillReturnError(sql.ErrNoRows)
	if logo == nil {
		mock.ExpectQuery("UPDATE estabelecimentos").
			WithArgs(id, nome, slug).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nome_comercial", "slug", "logo_url"}).
				AddRow(id, nome, slug, nil))
	} else {
		mock.ExpectQuery("UPDATE estabelecimentos").
			WithArgs(id, nome, slug, logo).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nome_comercial", "slug", "logo_url"}).
				AddRow(id, nome, slug, logo))
	}
	mock.ExpectCommit()
}

func TestUpdateEstablishmentIsRegisteredUnderTheAdminPattern(t *testing.T) {
	mux, _ := newUpdateEstablishmentMux(t)

	_, pattern := mux.(*http.ServeMux).Handler(
		updateRequest("", "tenant-1", `{"nome_comercial":"Studio","slug":"studio"}`),
	)
	if pattern != updateEstablishmentPattern {
		t.Fatalf("rota update não registrada: pattern = %q", pattern)
	}
}

func TestUpdateEstablishmentHappyPathActive(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	expectAdminIdentidadeHappyPath(mock, "tenant-1", "Studio Glow", "studio-glow", nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["id"] != "tenant-1" || body["slug"] != "studio-glow" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentHappyPathInactive(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	expectAdminIdentidadeHappyPath(mock, "tenant-off", "Studio Glow", "studio-glow", nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-off",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentSetsLogoURL(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	logo := "https://cdn.example/logo.png"
	expectAdminIdentidadeHappyPath(mock, "tenant-1", "Studio Glow", "studio-glow", logo)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow","logo_url":"https://cdn.example/logo.png"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["logo_url"] != logo {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentOmitsLogoKeepsCurrent(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	// omit logo_url → UPDATE sem coluna logo (3 args)
	expectAdminIdentidadeHappyPath(mock, "tenant-1", "Studio Glow", "studio-glow", nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow"}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentNullLogoKeepsCurrent(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	expectAdminIdentidadeHappyPath(mock, "tenant-1", "Studio Glow", "studio-glow", nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow","logo_url":null}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentEmptyLogoClears(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-1"))
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug = \\$1 AND id <> \\$2").
		WithArgs("studio-glow", "tenant-1").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("UPDATE estabelecimentos").
		WithArgs("tenant-1", "Studio Glow", "studio-glow").
		WillReturnRows(sqlmock.NewRows([]string{"id", "nome_comercial", "slug", "logo_url"}).
			AddRow("tenant-1", "Studio Glow", "studio-glow", nil))
	mock.ExpectCommit()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow","logo_url":""}`))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if _, hasLogo := body["logo_url"]; hasLogo {
		t.Fatalf("esperava logo omitido após clear, body=%#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentInvalidSlug(t *testing.T) {
	mux, _ := newUpdateEstablishmentMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"INVALID"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "invalid_slug" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUpdateEstablishmentMissingNome(t *testing.T) {
	mux, _ := newUpdateEstablishmentMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"  ","slug":"studio-glow"}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "missing_nome_comercial" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUpdateEstablishmentNotFound(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "missing",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow"}`))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "not_found" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentSlugConflict(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE id = \\$1 FOR UPDATE").
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-1"))
	mock.ExpectQuery("SELECT id FROM estabelecimentos WHERE slug = \\$1 AND id <> \\$2").
		WithArgs("outro", "tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-2"))
	mock.ExpectRollback()

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"outro"}`))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "slug_already_exists" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateEstablishmentInvalidJSON(t *testing.T) {
	mux, _ := newUpdateEstablishmentMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1", `{`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "invalid_json" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUpdateEstablishmentWithDonaTokenIsForbidden(t *testing.T) {
	mux, _ := newUpdateEstablishmentMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(donaToken(t, "tenant-1"), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow"}`))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestUpdateEstablishmentResponseShape(t *testing.T) {
	mux, mock := newUpdateEstablishmentMux(t)
	logo := "https://cdn.example/a.png"
	expectAdminIdentidadeHappyPath(mock, "tenant-1", "Studio Glow", "studio-glow", logo)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, updateRequest(superAdminToken(t), "tenant-1",
		`{"nome_comercial":"Studio Glow","slug":"studio-glow","logo_url":"https://cdn.example/a.png"}`))

	var resp updateEstablishmentResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != "tenant-1" || resp.NomeComercial != "Studio Glow" || resp.Slug != "studio-glow" {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.LogoURL == nil || *resp.LogoURL != logo {
		t.Fatalf("logo = %#v", resp.LogoURL)
	}
}
