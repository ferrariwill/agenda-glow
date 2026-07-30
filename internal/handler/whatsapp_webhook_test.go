package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agendaglow/agendaglow/internal/service"
)

func TestMapWhatsAppWebhookErrorPendingProfessionalApproval(t *testing.T) {
	recorder := httptest.NewRecorder()
	mapWhatsAppWebhookError(recorder, service.ErrAgendamentoAguardandoAprovacaoProfissional)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("status: got %d, want 409", recorder.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decodificar resposta: %v", err)
	}
	if body["error"] != "appointment_pending_professional_approval" {
		t.Fatalf("error: got %q, want appointment_pending_professional_approval", body["error"])
	}
}
