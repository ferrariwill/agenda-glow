package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
)

// Mesmo padrão registrado em backend/cmd/api/main.go (contrato 4.1).
const toggleWhatsAppPattern = "PUT /api/v1/admin/establishments/{id}/toggle-whatsapp"

const flagQuery = "SELECT COALESCE(whatsapp_enabled, FALSE)"

func newToggleWhatsAppMux(t *testing.T) (http.Handler, *security.WhatsAppGate, sqlmock.Sqlmock) {
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
	mux.Handle(toggleWhatsAppPattern, security.RequireSuperAdmin(http.HandlerFunc(handler.ToggleWhatsApp)))
	return mux, gate, mock
}

func superAdminToken(t *testing.T) string {
	t.Helper()
	token, err := security.GenerateToken(security.Claims{
		UserID: "user-super",
		Email:  security.SuperAdminEmail,
		Role:   security.RoleSuperAdmin,
	}, time.Hour)
	if err != nil {
		t.Fatalf("gerar token super admin: %v", err)
	}
	return token
}

func donaToken(t *testing.T, establishmentID string) string {
	t.Helper()
	token, err := security.GenerateToken(security.Claims{
		UserID:            "user-dona",
		Email:             "dona@glow.local",
		Role:              security.RoleDona,
		EstabelecimentoID: &establishmentID,
	}, time.Hour)
	if err != nil {
		t.Fatalf("gerar token dona: %v", err)
	}
	return token
}

func toggleRequest(token, establishmentID, body string) *http.Request {
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/establishments/"+establishmentID+"/toggle-whatsapp",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decodificar resposta: %v", err)
	}
	return body
}

func TestToggleWhatsAppIsRegisteredUnderTheAdminPattern(t *testing.T) {
	mux, _, _ := newToggleWhatsAppMux(t)

	_, pattern := mux.(*http.ServeMux).Handler(
		toggleRequest("", "tenant-1", `{"whatsapp_enabled":true}`),
	)
	if pattern != toggleWhatsAppPattern {
		t.Fatalf("rota do toggle não registrada: pattern = %q", pattern)
	}
}

func TestToggleWhatsAppWithoutTokenIsUnauthorized(t *testing.T) {
	mux, _, _ := newToggleWhatsAppMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, toggleRequest("", "tenant-1", `{"whatsapp_enabled":true}`))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "unauthorized" {
		t.Fatalf("body = %#v", body)
	}
}

func TestToggleWhatsAppWithDonaTokenIsForbidden(t *testing.T) {
	mux, _, _ := newToggleWhatsAppMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, toggleRequest(donaToken(t, "tenant-1"), "tenant-1", `{"whatsapp_enabled":true}`))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["error"] != "forbidden" || body["required_role"] != security.RoleSuperAdmin {
		t.Fatalf("body = %#v", body)
	}
}

func TestToggleWhatsAppRequiresTheFlagInTheBody(t *testing.T) {
	mux, _, mock := newToggleWhatsAppMux(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, toggleRequest(superAdminToken(t), "tenant-1", `{}`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "missing_whatsapp_enabled" {
		t.Fatalf("body = %#v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("campo ausente não pode chegar ao banco: %v", err)
	}
}

func TestToggleWhatsAppUnknownEstablishmentIsNotFound(t *testing.T) {
	mux, _, mock := newToggleWhatsAppMux(t)
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE estabelecimentos")).
		WithArgs("missing", true).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, toggleRequest(superAdminToken(t), "missing", `{"whatsapp_enabled":true}`))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := decodeBody(t, rec); body["error"] != "not_found" {
		t.Fatalf("body = %#v", body)
	}
}

func TestToggleWhatsAppInvalidatesTheGateCache(t *testing.T) {
	mux, gate, mock := newToggleWhatsAppMux(t)

	mock.ExpectQuery(regexp.QuoteMeta(flagQuery)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))
	if err := gate.IsEnabled(context.Background(), "tenant-1"); !errors.Is(err, security.ErrWhatsAppDesativado) {
		t.Fatalf("estado inicial = %v", err)
	}

	mock.ExpectQuery(regexp.QuoteMeta("UPDATE estabelecimentos")).
		WithArgs("tenant-1", true).
		WillReturnRows(sqlmock.NewRows([]string{
			"estabelecimento_id", "nome_comercial", "whatsapp_enabled", "whatsapp_status", "connected_at",
		}).AddRow("tenant-1", "Studio Bella", true, "CONECTADO", nil))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, toggleRequest(superAdminToken(t), "tenant-1", `{"whatsapp_enabled":true}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["whatsapp_enabled"] != true || body["whatsapp_status"] != "CONECTADO" {
		t.Fatalf("body = %#v", body)
	}

	mock.ExpectQuery(regexp.QuoteMeta(flagQuery)).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(true))
	if err := gate.IsEnabled(context.Background(), "tenant-1"); err != nil {
		t.Fatalf("cache não foi invalidado no toggle: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetWhatsAppIntegrationStaysAvailableWhenFeatureIsDisabled(t *testing.T) {
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	defer rawDB.Close()

	db := sqlx.NewDb(rawDB, "sqlmock")
	handler := NewWhatsAppIntegrationHandler(service.NewEstabelecimentoService(db))

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id,")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "whatsapp_enabled", "whatsapp_status", "whatsapp_waba_id",
			"whatsapp_phone_number_id", "whatsapp_connected_at",
		}).AddRow("tenant-1", false, "DESCONECTADO", nil, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/whatsapp/integration", nil)
	req = req.WithContext(security.WithEstablishmentID(req.Context(), "tenant-1"))
	rec := httptest.NewRecorder()
	handler.GetIntegration(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (nunca pode ser 403)", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["whatsapp_enabled"] != false || body["state"] != "" || body["signup_url"] != "" {
		t.Fatalf("body = %#v", body)
	}
}
