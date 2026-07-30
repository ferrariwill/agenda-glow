package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/agendaglow/agendaglow/internal/service"
	"github.com/jmoiron/sqlx"
)

func TestPublicManagementRoutesHideMalformedTokensAsNotFound(t *testing.T) {
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	h := NewPublicAppointmentsHandler(service.NewAgendaService(sqlx.NewDb(raw, "sqlmock")))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/public/appointments/manage/{token}", h.Manage)
	mux.HandleFunc("POST /api/v1/public/appointments/manage/{token}/cancel", h.CancelByManagementToken)

	tests := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{
			name:   "manage",
			method: http.MethodGet,
			path:   "/api/v1/public/appointments/manage/not-a-uuid",
		},
		{
			name:   "cancel",
			method: http.MethodPost,
			path:   "/api/v1/public/appointments/manage/not-a-uuid/cancel",
			body:   []byte(`{"motivo":""}`),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "cancel" {
				mock.ExpectBegin()
				mock.ExpectRollback()
			}
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if got := decodeField(t, rec.Body.String(), "error"); got != "appointment_not_found" {
				t.Fatalf("error=%q", got)
			}
		})
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
