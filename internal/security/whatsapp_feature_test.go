package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func newWhatsAppTestGate(t *testing.T) (*WhatsAppGate, sqlmock.Sqlmock) {
	t.Helper()
	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })
	return NewWhatsAppGate(sqlx.NewDb(rawDB, "sqlmock"), time.Minute), mock
}

func TestWhatsAppGateTreatsUnknownTenantAsDisabled(t *testing.T) {
	gate, mock := newWhatsAppTestGate(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	if err := gate.IsEnabled(context.Background(), "missing"); !errors.Is(err, ErrWhatsAppDesativado) {
		t.Fatalf("erro = %v", err)
	}
}

func TestWhatsAppGateInvalidationMakesToggleImmediate(t *testing.T) {
	gate, mock := newWhatsAppTestGate(t)
	query := regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")
	mock.ExpectQuery(query).WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))
	if err := gate.IsEnabled(context.Background(), "tenant-1"); !errors.Is(err, ErrWhatsAppDesativado) {
		t.Fatalf("primeira consulta = %v", err)
	}

	gate.InvalidateCache("tenant-1")
	mock.ExpectQuery(query).WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(true))
	if err := gate.IsEnabled(context.Background(), "tenant-1"); err != nil {
		t.Fatalf("consulta após invalidar = %v", err)
	}
}

func TestRequireWhatsAppEnabledReturnsStandardForbiddenBody(t *testing.T) {
	gate, mock := newWhatsAppTestGate(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/integration/start", nil)
	req = req.WithContext(WithEstablishmentID(req.Context(), "tenant-1"))
	rec := httptest.NewRecorder()
	RequireWhatsAppEnabled(gate)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler não deveria ser chamado")
	})(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "whatsapp_feature_disabled" ||
		body["message"] != "Recurso de WhatsApp desativado para este estabelecimento. Contate o administrador" {
		t.Fatalf("body = %#v", body)
	}
}
