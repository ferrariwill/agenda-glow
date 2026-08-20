package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
)

func newSuppliesMux(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("criar mock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	h := NewBootstrapAPIHandler(
		nil, nil, nil, nil, nil, nil, nil, nil,
		service.NewInsumoService(db),
		nil,
	)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/supplies", security.RequireDona(http.HandlerFunc(h.ListInsumos)))
	mux.Handle("POST /api/v1/supplies", security.RequireDona(http.HandlerFunc(h.CreateInsumo)))
	mux.Handle("PUT /api/v1/supplies/{id}", security.RequireDona(http.HandlerFunc(h.UpdateInsumo)))
	mux.Handle("POST /api/v1/supplies/{id}/adjust", security.RequireDona(http.HandlerFunc(h.AdjustInsumoEstoque)))
	return mux, mock
}

func TestCreateInsumo_ModoEmbalagensCalculaVolumeTotal(t *testing.T) {
	mux, mock := newSuppliesMux(t)
	tenant := "a1000002-0002-4002-8002-000000000002"
	token := donaToken(t, tenant)

	mock.ExpectQuery(`(?s)INSERT INTO insumos.+RETURNING id`).
		WithArgs(tenant, "Óleo Facial", "", "ESTETICA", 500.0, 50.0, 100.0, 500.0, 0.18, "ml", "", "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("insumo-1"))

	body := `{
		"nome":"Óleo Facial",
		"categoria":"ESTETICA",
		"unidade":"ml",
		"conteudo_por_embalagem":50,
		"quantidade_embalagens":10,
		"estoque_minimo":100,
		"estoque_ideal":500,
		"valor_unitario":0.18
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/supplies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s; want 201", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] != "insumo-1" {
		t.Fatalf("id = %q; want insumo-1", resp["id"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestAdjustInsumo_DeltaEmbalagens(t *testing.T) {
	mux, mock := newSuppliesMux(t)
	tenant := "a1000002-0002-4002-8002-000000000002"
	token := donaToken(t, tenant)
	insumoID := "insumo-1"

	cols := []string{
		"id", "estabelecimento_id", "nome", "marca", "categoria", "quantidade",
		"conteudo_por_embalagem", "estoque_minimo", "estoque_ideal", "valor_unitario",
		"unidade", "imagem_url", "instrucoes_uso", "ativo",
	}
	mock.ExpectQuery(`(?s)SELECT.+FROM insumos.+WHERE id = \$1 AND estabelecimento_id = \$2`).
		WithArgs(insumoID, tenant).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			insumoID, tenant, "Óleo", nil, "ESTETICA", 500.0, 50.0, 100.0, 500.0, 0.18, "ml", nil, nil, true,
		))

	mock.ExpectExec(`(?s)UPDATE insumos.+quantidade = GREATEST.+WHERE id = \$1 AND estabelecimento_id = \$2`).
		WithArgs(insumoID, tenant, -50.0).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/supplies/"+insumoID+"/adjust",
		strings.NewReader(`{"delta_embalagens":-1}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s; want 200", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestAdjustInsumo_TenantIsolation(t *testing.T) {
	mux, mock := newSuppliesMux(t)
	tenant := "a1000002-0002-4002-8002-000000000002"
	token := donaToken(t, tenant)
	insumoID := "insumo-outro-tenant"

	mock.ExpectExec(`(?s)UPDATE insumos.+quantidade = GREATEST.+WHERE id = \$1 AND estabelecimento_id = \$2`).
		WithArgs(insumoID, tenant, -30.0).
		WillReturnResult(sqlmock.NewResult(0, 0))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/supplies/"+insumoID+"/adjust",
		strings.NewReader(`{"delta":-30}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s; want 404", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestListInsumos_IncluiDerivadoEmbalagens(t *testing.T) {
	mux, mock := newSuppliesMux(t)
	tenant := "a1000002-0002-4002-8002-000000000002"
	token := donaToken(t, tenant)

	cols := []string{
		"id", "estabelecimento_id", "nome", "marca", "categoria", "quantidade",
		"conteudo_por_embalagem", "estoque_minimo", "estoque_ideal", "valor_unitario",
		"unidade", "imagem_url", "instrucoes_uso", "ativo",
	}
	mock.ExpectQuery(`(?s)SELECT.+FROM insumos.+WHERE estabelecimento_id = \$1 AND ativo = TRUE`).
		WithArgs(tenant).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(
			"insumo-1", tenant, "Shampoo X", nil, "CUIDADOS_CAPILARES", 500.0, 50.0,
			100.0, 500.0, 0.18, "ml", nil, nil, true,
		))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/supplies", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s; want 200", rec.Code, rec.Body.String())
	}
	var list []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len = %d; want 1", len(list))
	}
	if list[0]["quantidade"] != float64(500) {
		t.Fatalf("quantidade = %v; want 500", list[0]["quantidade"])
	}
	if list[0]["conteudo_por_embalagem"] != float64(50) {
		t.Fatalf("conteudo = %v; want 50", list[0]["conteudo_por_embalagem"])
	}
	if list[0]["quantidade_embalagens"] != float64(10) {
		t.Fatalf("embalagens = %v; want 10", list[0]["quantidade_embalagens"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
