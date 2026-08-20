package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agendaglow/agendaglow/internal/security"
	"github.com/agendaglow/agendaglow/internal/service"
)

const professionalAppointmentsPattern = "POST /api/v1/professional/appointments"

type stubAgendaAPI struct {
	lastEstablishmentID string
	lastProfissionalID  string
	lastOrigem          service.OrigemAgendamento
	result              service.ResultadoAgendamento
	err                 error
}

func (s *stubAgendaAPI) CriarAgendamento(
	_ context.Context,
	estabelecimentoID, _, _, profissionalID, _ string,
	_ []string,
	_ time.Time,
	origem service.OrigemAgendamento,
	_ bool,
) (service.ResultadoAgendamento, error) {
	s.lastEstablishmentID = estabelecimentoID
	s.lastProfissionalID = profissionalID
	s.lastOrigem = origem
	if s.err != nil {
		return service.ResultadoAgendamento{}, s.err
	}
	return s.result, nil
}

func (s *stubAgendaAPI) ValidarAgendamentoDaProfissional(context.Context, string, string, string) error {
	return nil
}

func (s *stubAgendaAPI) ConcluirAtendimentoProfissional(context.Context, string, string, string, *service.FinanceiroService) error {
	return nil
}

func (s *stubAgendaAPI) GetDashboardProfissional(context.Context, string, string, time.Time) (*service.DashboardProfissional, error) {
	return nil, nil
}

func profissionalToken(t *testing.T, establishmentID, profissionalID string) string {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")
	token, err := security.GenerateToken(security.Claims{
		UserID:            "user-prof",
		Email:             "prof@glow.local",
		Role:              security.RoleProfissional,
		EstabelecimentoID: &establishmentID,
		ProfissionalID:    &profissionalID,
	}, time.Hour)
	if err != nil {
		t.Fatalf("gerar token profissional: %v", err)
	}
	return token
}

func newProfessionalAppointmentsMux(t *testing.T, agenda agendaAPI) http.Handler {
	t.Helper()
	t.Setenv("JWT_SECRET", "segredo-de-teste-agendaglow-32b")
	h := NewBootstrapAPIHandler(nil, nil, agenda, nil, nil, nil, nil, nil, nil, nil)
	mux := http.NewServeMux()
	mux.Handle(professionalAppointmentsPattern, security.RequireProfissional(http.HandlerFunc(h.CreateProfessionalAppointment)))
	return mux
}

func professionalCreateRequest(token, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/professional/appointments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestProfessionalCreateAppointment_RegisteredPattern(t *testing.T) {
	mux := newProfessionalAppointmentsMux(t, &stubAgendaAPI{})
	_, pattern := mux.(*http.ServeMux).Handler(professionalCreateRequest("", `{}`))
	if pattern != professionalAppointmentsPattern {
		t.Fatalf("pattern = %q, want %q", pattern, professionalAppointmentsPattern)
	}
}

func TestProfessionalCreateAppointment_DonaForbidden(t *testing.T) {
	mux := newProfessionalAppointmentsMux(t, &stubAgendaAPI{})
	token := donaToken(t, "tenant-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, professionalCreateRequest(token, `{
		"cliente_nome":"Maria","cliente_telefone":"11999998888",
		"servico_id":"svc-1","data":"2026-08-21","hora_inicio":"14:00"
	}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["error"] != "forbidden" {
		t.Fatalf("error = %v, want forbidden", body["error"])
	}
}

func TestProfessionalCreateAppointment_Created201(t *testing.T) {
	stub := &stubAgendaAPI{result: service.ResultadoAgendamento{ID: "ag-1", Status: "AGENDADO"}}
	mux := newProfessionalAppointmentsMux(t, stub)
	token := profissionalToken(t, "tenant-1", "prof-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, professionalCreateRequest(token, `{
		"cliente_nome":"Maria Silva",
		"cliente_telefone":"11999998888",
		"servico_id":"svc-1",
		"adicional_ids":[],
		"data":"2026-08-21",
		"hora_inicio":"14:00",
		"aceita_adiantar":false
	}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if stub.lastProfissionalID != "prof-a" {
		t.Fatalf("profissional_id usado = %q, want prof-a", stub.lastProfissionalID)
	}
	if stub.lastEstablishmentID != "tenant-1" {
		t.Fatalf("estabelecimento_id = %q, want tenant-1", stub.lastEstablishmentID)
	}
	if stub.lastOrigem != service.OrigemInterno {
		t.Fatalf("origem = %q, want INTERNO", stub.lastOrigem)
	}
	var body service.ResultadoAgendamento
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != "ag-1" || body.Status != "AGENDADO" {
		t.Fatalf("body = %+v", body)
	}
}

func TestProfessionalCreateAppointment_Collision409(t *testing.T) {
	stub := &stubAgendaAPI{err: service.ErrColisaoHorario}
	mux := newProfessionalAppointmentsMux(t, stub)
	token := profissionalToken(t, "tenant-1", "prof-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, professionalCreateRequest(token, `{
		"cliente_nome":"Maria","cliente_telefone":"11999998888",
		"servico_id":"svc-1","data":"2026-08-21","hora_inicio":"14:00"
	}`))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["error"] != "slot_unavailable" {
		t.Fatalf("error = %v, want slot_unavailable", body["error"])
	}
}

func TestProfessionalCreateAppointment_RejectsOtherProfessionalID(t *testing.T) {
	stub := &stubAgendaAPI{result: service.ResultadoAgendamento{ID: "ag-x", Status: "AGENDADO"}}
	mux := newProfessionalAppointmentsMux(t, stub)
	token := profissionalToken(t, "tenant-1", "prof-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, professionalCreateRequest(token, `{
		"cliente_nome":"Maria","cliente_telefone":"11999998888",
		"profissional_id":"prof-b",
		"servico_id":"svc-1","data":"2026-08-21","hora_inicio":"14:00"
	}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if stub.lastProfissionalID != "" {
		t.Fatalf("CriarAgendamento não deveria ser chamado; got profissional_id=%q", stub.lastProfissionalID)
	}
}

func TestProfessionalCreateAppointment_IgnoresMatchingBodyProfessionalID(t *testing.T) {
	stub := &stubAgendaAPI{result: service.ResultadoAgendamento{ID: "ag-2", Status: "AGENDADO"}}
	mux := newProfessionalAppointmentsMux(t, stub)
	token := profissionalToken(t, "tenant-1", "prof-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, professionalCreateRequest(token, `{
		"cliente_nome":"Maria","cliente_telefone":"11999998888",
		"profissional_id":"prof-a",
		"servico_id":"svc-1","data":"2026-08-21","hora_inicio":"14:00"
	}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if stub.lastProfissionalID != "prof-a" {
		t.Fatalf("profissional_id = %q, want prof-a", stub.lastProfissionalID)
	}
}

func TestProfessionalCreateAppointment_InvalidDatetime400(t *testing.T) {
	mux := newProfessionalAppointmentsMux(t, &stubAgendaAPI{})
	token := profissionalToken(t, "tenant-1", "prof-a")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, professionalCreateRequest(token, `{
		"cliente_nome":"Maria","cliente_telefone":"11999998888",
		"servico_id":"svc-1","data":"21-08-2026","hora_inicio":"14:00"
	}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
