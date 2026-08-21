package handler

import (
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

func newAvisosMux(t *testing.T) (http.Handler, sqlmock.Sqlmock) {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")

	rawDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = rawDB.Close() })

	db := sqlx.NewDb(rawDB, "sqlmock")
	svc := service.NewAvisoService(db)
	admin := NewAdminAvisosHandler(svc)
	me := NewMeNotificacoesHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/admin/avisos", security.RequireSuperAdmin(http.HandlerFunc(admin.Create)))
	mux.Handle("GET /api/v1/admin/avisos", security.RequireSuperAdmin(http.HandlerFunc(admin.List)))
	mux.Handle("PATCH /api/v1/admin/avisos/{id}", security.RequireSuperAdmin(http.HandlerFunc(admin.Patch)))
	mux.Handle("GET /api/v1/me/notificacoes", security.AuthenticateMiddleware(http.HandlerFunc(me.List)))
	mux.Handle("POST /api/v1/me/notificacoes/{id}/read", security.AuthenticateMiddleware(http.HandlerFunc(me.MarkRead)))
	return mux, mock
}

func avisoProfissionalToken(t *testing.T, establishmentID, profissionalID string) string {
	t.Helper()
	token, err := security.GenerateToken(security.Claims{
		UserID:            "user-prof",
		Email:             "prof@glow.local",
		Role:              security.RoleProfissional,
		EstabelecimentoID: &establishmentID,
		ProfissionalID:    &profissionalID,
	}, time.Hour)
	if err != nil {
		t.Fatalf("token profissional: %v", err)
	}
	return token
}

func expectInboxList(mock sqlmock.Sqlmock, userID, role string, estID any, limit int, rows *sqlmock.Rows, unread int) {
	mock.ExpectQuery(`titulo AS title`).
		WithArgs(userID, role, estID, limit).
		WillReturnRows(rows)
	mock.ExpectQuery(`al\.read_at IS NULL`).
		WithArgs(userID, role, estID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(unread))
}

func TestAdminAvisos_CreateAllTenants_DonaSeesProfissionalDoesNot(t *testing.T) {
	mux, mock := newAvisosMux(t)
	avisoID := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	createdAt := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	expires := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	corpo := "Sistema indisponível domingo 2h–4h."

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO avisos_globais`).
		WithArgs("Manutenção programada", sqlmock.AnyArg(), "WARNING", "ALL_TENANTS", "user-super", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "titulo", "corpo", "severidade", "audience_tipo", "ativo", "created_by", "criado_em", "expires_at",
		}).AddRow(avisoID, "Manutenção programada", corpo, "WARNING", "ALL_TENANTS", true, "user-super", createdAt, expires))
	mock.ExpectCommit()

	rec := httptest.NewRecorder()
	body := `{
		"titulo":"Manutenção programada",
		"corpo":"Sistema indisponível domingo 2h–4h.",
		"severidade":"WARNING",
		"audience_tipo":"ALL_TENANTS",
		"estabelecimento_ids":[],
		"expires_at":"2026-09-01T03:00:00Z"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/avisos", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	estID := "11111111-1111-4111-8111-111111111111"
	expectInboxList(mock, "user-dona", security.RoleDona, estID, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}).
			AddRow(avisoID, "Manutenção programada", corpo, createdAt, nil),
		1)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dona inbox status=%d body=%s", rec.Code, rec.Body.String())
	}
	var inbox service.InboxResult
	if err := json.Unmarshal(rec.Body.Bytes(), &inbox); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(inbox.Items) != 1 || inbox.UnreadCount != 1 {
		t.Fatalf("inbox=%+v", inbox)
	}

	expectInboxList(mock, "user-prof", security.RoleProfissional, estID, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}),
		0)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+avisoProfissionalToken(t, estID, "22222222-2222-4222-8222-222222222222"))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("prof inbox status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &inbox); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(inbox.Items) != 0 || inbox.UnreadCount != 0 {
		t.Fatalf("profissional não deve ver avisos: %+v", inbox)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestAdminAvisos_CreateEstabelecimentos_OnlyListedDona(t *testing.T) {
	mux, mock := newAvisosMux(t)
	avisoID := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	estA := "11111111-1111-4111-8111-111111111111"
	estB := "33333333-3333-4333-8333-333333333333"
	createdAt := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM estabelecimentos WHERE id = ANY($1)`)).
		WithArgs(pq.Array([]string{estA})).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`INSERT INTO avisos_globais`).
		WithArgs("Só salão A", nil, "INFO", "ESTABELECIMENTOS", "user-super", nil).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "titulo", "corpo", "severidade", "audience_tipo", "ativo", "created_by", "criado_em", "expires_at",
		}).AddRow(avisoID, "Só salão A", nil, "INFO", "ESTABELECIMENTOS", true, "user-super", createdAt, nil))
	mock.ExpectExec(`INSERT INTO aviso_estabelecimentos`).
		WithArgs(avisoID, estA).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/avisos", strings.NewReader(
		`{"titulo":"Só salão A","audience_tipo":"ESTABELECIMENTOS","estabelecimento_ids":["`+estA+`"]}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rec.Code, rec.Body.String())
	}

	expectInboxList(mock, "user-dona", security.RoleDona, estA, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}).
			AddRow(avisoID, "Só salão A", nil, createdAt, nil),
		1)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estA))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("dona A status=%d body=%s", rec.Code, rec.Body.String())
	}

	expectInboxList(mock, "user-dona", security.RoleDona, estB, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}),
		0)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estB))
	mux.ServeHTTP(rec, req)
	var inbox service.InboxResult
	_ = json.Unmarshal(rec.Body.Bytes(), &inbox)
	if len(inbox.Items) != 0 {
		t.Fatalf("dona B não deve ver aviso: %+v", inbox)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestAdminAvisos_SuperAdminsAudience(t *testing.T) {
	mux, mock := newAvisosMux(t)
	avisoID := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	createdAt := time.Now().UTC()

	expectInboxList(mock, "user-super", security.RoleSuperAdmin, nil, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}).
			AddRow(avisoID, "Ops", "interno", createdAt, nil),
		1)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	estID := "11111111-1111-4111-8111-111111111111"
	expectInboxList(mock, "user-dona", security.RoleDona, estID, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}),
		0)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	var inbox service.InboxResult
	_ = json.Unmarshal(rec.Body.Bytes(), &inbox)
	if len(inbox.Items) != 0 {
		t.Fatalf("dona não deve ver SUPER_ADMINS: %+v", inbox)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestAdminAvisos_InactiveRemainsInAdminList(t *testing.T) {
	mux, mock := newAvisosMux(t)
	avisoID := "dddddddd-dddd-4ddd-8ddd-dddddddddddd"
	createdAt := time.Now().UTC()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM avisos_globais WHERE TRUE`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`FROM avisos_globais\s+WHERE TRUE`).
		WithArgs(50, 0).WillReturnRows(sqlmock.NewRows([]string{
		"id", "titulo", "corpo", "severidade", "audience_tipo", "ativo", "created_by", "criado_em", "expires_at",
	}).AddRow(avisoID, "Off", nil, "INFO", "ALL_TENANTS", false, "user-super", createdAt, nil))
	mock.ExpectQuery(`FROM aviso_estabelecimentos`).
		WithArgs(avisoID).WillReturnRows(sqlmock.NewRows([]string{"estabelecimento_id"}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/avisos", nil)
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var out adminAvisosListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v", err)
	}
	if out.Total != 1 || len(out.Items) != 1 || out.Items[0].Ativo {
		t.Fatalf("admin list=%+v", out)
	}

	estID := "11111111-1111-4111-8111-111111111111"
	expectInboxList(mock, "user-dona", security.RoleDona, estID, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}),
		0)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	var inbox service.InboxResult
	_ = json.Unmarshal(rec.Body.Bytes(), &inbox)
	if len(inbox.Items) != 0 {
		t.Fatalf("inbox não deve listar inativo: %+v", inbox)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestMeNotificacoes_MarkReadIdempotentAndUnreadDecreases(t *testing.T) {
	mux, mock := newAvisosMux(t)
	avisoID := "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
	estID := "11111111-1111-4111-8111-111111111111"
	createdAt := time.Now().UTC()
	readAt := createdAt.Add(5 * time.Minute)

	mock.ExpectQuery(`a\.id = \$4`).
		WithArgs("user-dona", security.RoleDona, estID, avisoID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}).
			AddRow(avisoID, "Hi", "body", createdAt, nil))
	mock.ExpectQuery(`INSERT INTO aviso_leituras`).
		WithArgs(avisoID, "user-dona").
		WillReturnRows(sqlmock.NewRows([]string{"read_at"}).AddRow(readAt))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/notificacoes/"+avisoID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mark status=%d body=%s", rec.Code, rec.Body.String())
	}

	mock.ExpectQuery(`a\.id = \$4`).
		WithArgs("user-dona", security.RoleDona, estID, avisoID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}).
			AddRow(avisoID, "Hi", "body", createdAt, readAt))
	mock.ExpectQuery(`INSERT INTO aviso_leituras`).
		WithArgs(avisoID, "user-dona").
		WillReturnRows(sqlmock.NewRows([]string{"read_at"}).AddRow(readAt))

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/me/notificacoes/"+avisoID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mark2 status=%d", rec.Code)
	}

	expectInboxList(mock, "user-dona", security.RoleDona, estID, 50,
		sqlmock.NewRows([]string{"id", "title", "body", "created_at", "read_at"}).
			AddRow(avisoID, "Hi", "body", createdAt, readAt),
		0)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me/notificacoes", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	var inbox service.InboxResult
	_ = json.Unmarshal(rec.Body.Bytes(), &inbox)
	if inbox.UnreadCount != 0 {
		t.Fatalf("unread_count=%d", inbox.UnreadCount)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestAdminAvisos_ForbiddenForDonaAndUnauthorized(t *testing.T) {
	mux, _ := newAvisosMux(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/avisos", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sem auth status=%d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/avisos", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, "11111111-1111-4111-8111-111111111111"))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("dona admin status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMeNotificacoes_MarkReadCrossAudienceReturns404(t *testing.T) {
	mux, mock := newAvisosMux(t)
	avisoID := "ffffffff-ffff-4fff-8fff-ffffffffffff"
	estID := "11111111-1111-4111-8111-111111111111"

	mock.ExpectQuery(`a\.id = \$4`).
		WithArgs("user-dona", security.RoleDona, estID, avisoID).
		WillReturnError(sql.ErrNoRows)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/notificacoes/"+avisoID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+donaToken(t, estID))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestAdminAvisos_CreateValidationErrors(t *testing.T) {
	mux, _ := newAvisosMux(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/avisos", strings.NewReader(`{`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_json") {
		t.Fatalf("invalid_json: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/admin/avisos", strings.NewReader(
		`{"titulo":"","audience_tipo":"ALL_TENANTS"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+superAdminToken(t))
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_titulo") {
		t.Fatalf("invalid_titulo: %d %s", rec.Code, rec.Body.String())
	}
}
