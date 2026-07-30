package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildWhatsAppStateAndSignupURL(t *testing.T) {
	id := "a1000002-0002-4002-8002-000000000002"
	if got := BuildWhatsAppState(id); got != "beleza_"+id {
		t.Fatalf("state: got %q", got)
	}
	parsed, err := ParseWhatsAppState("beleza_" + id)
	if err != nil || parsed != id {
		t.Fatalf("parse: %v %q", err, parsed)
	}
	if _, err := ParseWhatsAppState("invalid"); err == nil {
		t.Fatal("expected invalid state error")
	}

	t.Setenv("META_WHATSAPP_EMBEDDED_SIGNUP_URL", "https://example.com/onboard?app_id=1")
	url := BuildWhatsAppEmbeddedSignupURL(id)
	if !strings.Contains(url, "state=beleza_"+id) {
		t.Fatalf("signup url missing state: %s", url)
	}
	if !strings.HasPrefix(url, "https://example.com/onboard?") {
		t.Fatalf("unexpected base: %s", url)
	}
}

func TestEnviarNotificacaoWhatsApp_ContractPayload(t *testing.T) {
	var gotPath string
	var gotKey string
	var gotBody whatsAppSendNotificationRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-API-Key")
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	t.Setenv("WHATSAPP_GATEWAY_URL", srv.URL)
	t.Setenv("WHATSAPP_GATEWAY_KEY", "test-key-qa")

	err := EnviarNotificacaoWhatsApp(context.Background(), WhatsAppNotificationInput{
		TenantID:      "tenant-1",
		PhoneNumber:   "5515999990000",
		TemplateName:  "lembrete_agenda",
		AppointmentID: "appt-1",
		Variables:     []string{"Ana", "Claudia", "Unhas", "29/07/2026 14:00"},
	})
	if err != nil {
		t.Fatalf("enviar: %v", err)
	}
	if gotPath != "/send-notification" {
		t.Fatalf("path: got %q want /send-notification", gotPath)
	}
	if gotKey != "test-key-qa" {
		t.Fatalf("X-API-Key: got %q", gotKey)
	}
	if gotBody.SistemaOrigem != "beleza" {
		t.Fatalf("sistema_origem: %q", gotBody.SistemaOrigem)
	}
	if gotBody.TenantID != "tenant-1" {
		t.Fatalf("tenant_id: %q", gotBody.TenantID)
	}
	if !gotBody.SimpleTemplate {
		t.Fatal("simple_template deve ser true")
	}
	if gotBody.LanguageCode != "pt_BR" {
		t.Fatalf("language_code: %q", gotBody.LanguageCode)
	}
	if gotBody.AppointmentID != "appt-1" {
		t.Fatalf("appointment_id: %q", gotBody.AppointmentID)
	}
	if len(gotBody.Variables) != 4 {
		t.Fatalf("variables: %#v", gotBody.Variables)
	}
}

func TestMoneyRoundTripComissao(t *testing.T) {
	entrada := 130.00
	if got := calcularComissao(entrada, 40); got != 52.00 {
		t.Fatalf("comissao 40%%: got %.2f want 52.00", got)
	}
	if got := calcularComissao(entrada, 45); got != 58.50 {
		t.Fatalf("comissao 45%%: got %.2f want 58.50", got)
	}
}
