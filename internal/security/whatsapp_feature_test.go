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

// newWhatsAppStartChain reproduz a cadeia de POST /api/v1/whatsapp/integration/start
// registrada em backend/cmd/api/main.go: papel → assinatura SaaS → flag do recurso.
func newWhatsAppStartChain(t *testing.T) (http.Handler, sqlmock.Sqlmock, *bool) {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	reached := false
	next := func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		writeJSON(w, http.StatusOK, map[string]string{"status": "PENDENTE"})
	}
	saasValidation := SaaSValidationMiddleware(NewSaaSGuard(db, time.Minute))
	whatsAppFeature := RequireWhatsAppEnabled(NewWhatsAppGate(db, time.Minute))

	return RequireDona(http.HandlerFunc(saasValidation(whatsAppFeature(next)))), mock, &reached
}

func startConnectionRequest(t *testing.T, establishmentID string) *http.Request {
	t.Helper()
	token, err := GenerateToken(Claims{
		UserID:            "user-dona",
		Email:             "dona@glow.local",
		Role:              RoleDona,
		EstabelecimentoID: &establishmentID,
	}, time.Hour)
	if err != nil {
		t.Fatalf("gerar token dona: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/whatsapp/integration/start", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func expectSubscription(mock sqlmock.Sqlmock, establishmentID, status string, vencimento time.Time) {
	mock.ExpectQuery(regexp.QuoteMeta("FROM assinaturas_estabelecimentos")).
		WithArgs(establishmentID).
		WillReturnRows(sqlmock.NewRows([]string{"status", "data_vencimento"}).AddRow(status, vencimento))
}

func TestStartConnectionAnswersPaymentRequiredBeforeCheckingTheFlag(t *testing.T) {
	chain, mock, reached := newWhatsAppStartChain(t)
	expectSubscription(mock, "tenant-1", "SUSPENSO", time.Now().Add(30*24*time.Hour))

	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, startConnectionRequest(t, "tenant-1"))

	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "subscription_expired_or_suspended" {
		t.Fatalf("body = %#v", body)
	}
	if *reached {
		t.Fatal("handler não deveria ser alcançado")
	}
	// Nenhuma consulta à flag foi registrada: a assinatura barra antes.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStartConnectionIsForbiddenWhenOnlyTheFlagIsOff(t *testing.T) {
	chain, mock, reached := newWhatsAppStartChain(t)
	expectSubscription(mock, "tenant-1", "ATIVO", time.Now().Add(30*24*time.Hour))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(false))

	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, startConnectionRequest(t, "tenant-1"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rec.Code)
	}
	if *reached {
		t.Fatal("handler não deveria ser alcançado")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStartConnectionPassesWithActiveSubscriptionAndFlagOn(t *testing.T) {
	chain, mock, reached := newWhatsAppStartChain(t)
	expectSubscription(mock, "tenant-1", "ATIVO", time.Now().Add(30*24*time.Hour))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(whatsapp_enabled, FALSE)")).
		WithArgs("tenant-1").
		WillReturnRows(sqlmock.NewRows([]string{"whatsapp_enabled"}).AddRow(true))

	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, startConnectionRequest(t, "tenant-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !*reached {
		t.Fatal("handler deveria ser alcançado")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
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
